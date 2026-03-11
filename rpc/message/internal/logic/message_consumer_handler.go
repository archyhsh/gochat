package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"

	"github.com/IBM/sarama"
)

type MessageConsumerHandler struct {
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMessageConsumerHandler(svcCtx *svc.ServiceContext) *MessageConsumerHandler {
	return &MessageConsumerHandler{
		svcCtx: svcCtx,
		Logger: logx.WithContext(context.Background()),
	}
}

func (h *MessageConsumerHandler) Handle(ctx context.Context, message *sarama.ConsumerMessage) error {
	switch message.Topic {
	case h.svcCtx.Config.Kafka.Topics.Message:
		return h.handleChatMessage(ctx, message.Value)
	case h.svcCtx.Config.Kafka.Topics.Group:
		return h.handleSystemEvent(ctx, message.Value)
	default:
		h.Errorf("Unknown topic: %s", message.Topic)
	}
	return nil
}

func (h *MessageConsumerHandler) handleChatMessage(ctx context.Context, data []byte) error {
	var event pb.ChatMessageEvent
	err := proto.Unmarshal(data, &event)
	if err != nil {
		h.Debugf("Proto unmarshal failed, trying JSON: %v", err)
		if err := json.Unmarshal(data, &event); err != nil {
			h.Errorf("Failed to unmarshal chat message: %v", err)
			return err
		}
	}

	l := NewSaveMessageLogic(ctx, h.svcCtx)
	_, err = l.SaveMessage(&pb.SaveMessageRequest{
		Message: &event,
	})
	if err == nil {
		h.pushToGateways(ctx, &event)
	}
	return err
}

func (h *MessageConsumerHandler) handleSystemEvent(ctx context.Context, data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	eventType, _ := raw["type"].(string)
	if eventType == "friend_event" {
		return h.handleFriendEvent(ctx, raw)
	}
	return h.handleGroupEvent(ctx, raw)
}

func (h *MessageConsumerHandler) handleFriendEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	if action != "accept" {
		return nil
	}

	fromId := h.toInt64(event["from_user_id"])
	toId := h.toInt64(event["to_user_id"])

	fromName := h.getUserNickname(ctx, fromId)
	toName := h.getUserNickname(ctx, toId)

	content := fmt.Sprintf("You and %s are now friends. Say hello!", toName)
	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	convId := h.getPrivateConvId(fromId, toId)

	l := NewSaveMessageLogic(ctx, h.svcCtx)
	eventA := &pb.ChatMessageEvent{
		MsgId:          msgId,
		ConversationId: convId,
		SenderId:       0,
		ReceiverId:     toId,
		MsgType:        6,
		Content:        content,
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      []int64{fromId},
	}
	_, err := l.SaveMessage(&pb.SaveMessageRequest{
		Message: eventA,
	})
	if err == nil {
		h.pushToGateways(ctx, eventA)
	}

	contentTo := fmt.Sprintf("You and %s are now friends. Say hello!", fromName)
	msgIdTo := strconv.FormatInt(snowflake.MustNextID(), 10)
	eventB := &pb.ChatMessageEvent{
		MsgId:          msgIdTo,
		ConversationId: convId,
		SenderId:       0,
		ReceiverId:     fromId,
		MsgType:        6,
		Content:        contentTo,
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      []int64{toId},
	}
	_, err = l.SaveMessage(&pb.SaveMessageRequest{
		Message: eventB,
	})
	if err == nil {
		h.pushToGateways(ctx, eventB)
	}

	return err
}

func (h *MessageConsumerHandler) handleGroupEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	groupId := h.toInt64(event["group_id"])
	actorId := h.toInt64(event["user_id"])
	actorName := h.getUserNickname(ctx, actorId)

	var content string
	var targets []int64

	switch action {
	case "create":
		content = fmt.Sprintf("%s created the group", actorName)
		targets = []int64{actorId}
	case "join":
		content = fmt.Sprintf("%s joined the group", actorName)
		if intro, ok := event["intro"].(string); ok && intro != "" {
			content += fmt.Sprintf(". Intro: %s", intro)
		}
		targets = []int64{actorId}
	case "invite":
		var inviteeNames []string
		if rawIds, ok := event["member_ids"].([]interface{}); ok {
			for _, rid := range rawIds {
				uid := int64(rid.(float64))
				inviteeNames = append(inviteeNames, h.getUserNickname(ctx, uid))
				targets = append(targets, uid)
			}
		}
		content = fmt.Sprintf("%s invited %s to the group", actorName, strings.Join(inviteeNames, ", "))
	case "quit":
		content = fmt.Sprintf("%s quit the group", actorName)
	case "kick":
		content = fmt.Sprintf("Admin removed %s from the group", h.getUserNickname(ctx, h.toInt64(event["user_id"])))
	case "dismiss":
		content = "This group has been dismissed by the owner"
	case "update_announcement":
		content = fmt.Sprintf("%s updated the group announcement", actorName)
	default:
		return nil
	}

	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	convId := fmt.Sprintf("group_%d", groupId)

	l := NewSaveMessageLogic(ctx, h.svcCtx)
	evt := &pb.ChatMessageEvent{
		MsgId:          msgId,
		ConversationId: convId,
		SenderId:       0,
		GroupId:        groupId,
		MsgType:        6,
		Content:        content,
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      targets,
	}
	_, err := l.SaveMessage(&pb.SaveMessageRequest{
		Message: evt,
	})
	if err == nil {
		h.pushToGateways(ctx, evt)
	}
	return err
}

func (h *MessageConsumerHandler) pushToGateways(ctx context.Context, event *pb.ChatMessageEvent) {
	// 1. Identify all target users
	var targetUsers []int64
	if len(event.TargetIds) > 0 {
		targetUsers = event.TargetIds
	} else if event.GroupId > 0 {
		resp, err := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &groupservice.GetGroupMembersRequest{
			GroupId: event.GroupId,
		})
		if err == nil {
			for _, m := range resp.Members {
				targetUsers = append(targetUsers, m.UserId)
			}
		}
	} else {
		// Private chat
		targetUsers = []int64{event.SenderId, event.ReceiverId}
	}

	// 2. Group users by gateway address
	gwMap := make(map[string][]int64)
	for _, uid := range targetUsers {
		addr, err := h.svcCtx.Router.Find(ctx, uid)
		if err == nil && addr != "" {
			gwMap[addr] = append(gwMap[addr], uid)
		}
	}

	// 3. Call internal push API of each gateway in parallel
	for addr, uids := range gwMap {
		go h.sendPushRequest(addr, uids, event)
	}
}

func (h *MessageConsumerHandler) sendPushRequest(gwAddr string, userIds []int64, event *pb.ChatMessageEvent) {
	url := fmt.Sprintf("http://%s/internal/push", gwAddr)
	payload := map[string]interface{}{
		"user_ids":        userIds,
		"conversation_id": event.ConversationId,
		"msg_id":          event.MsgId,
		"sender_id":       event.SenderId,
		"content":         event.Content,
		"msg_type":        int(event.MsgType),
		"timestamp":       event.Timestamp,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		h.Errorf("Failed to create push request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.svcCtx.HttpClient.Do(req)
	if err != nil {
		h.Errorf("Failed to push to gateway %s: %v", gwAddr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.Errorf("Gateway %s returned status %d for push", gwAddr, resp.StatusCode)
	}
}

func (h *MessageConsumerHandler) toInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	if f, ok := val.(float64); ok {
		return int64(f)
	}
	if s, ok := val.(string); ok {
		id, _ := strconv.ParseInt(s, 10, 64)
		return id
	}
	return 0
}

func (h *MessageConsumerHandler) getUserNickname(ctx context.Context, userId int64) string {
	if userId <= 0 {
		return "System"
	}
	userResp, err := h.svcCtx.UserRpc.GetUser(ctx, &userservice.GetUserRequest{
		UserId: userId,
	})
	if err == nil && userResp.User != nil {
		return userResp.User.Nickname
	}
	return fmt.Sprintf("User %d", userId)
}

func (h *MessageConsumerHandler) getPrivateConvId(uid1, uid2 int64) string {
	if uid1 < uid2 {
		return fmt.Sprintf("conv_%d_%d", uid1, uid2)
	}
	return fmt.Sprintf("conv_%d_%d", uid2, uid1)
}
