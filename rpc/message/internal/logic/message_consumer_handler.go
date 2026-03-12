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
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
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
	case h.svcCtx.Config.Kafka.Topics.User:
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
	switch eventType {
	case "friend_event":
		return h.handleFriendEvent(ctx, raw)
	case "group_event":
		return h.handleGroupEvent(ctx, raw)
	case "nickname_update":
		return h.handleNicknameUpdate(ctx, raw)
	case "group_nickname_update":
		return h.handleGroupNicknameUpdate(ctx, raw)
	}
	return nil
}

func (h *MessageConsumerHandler) handleNicknameUpdate(ctx context.Context, event map[string]interface{}) error {
	userId := h.toInt64(event["user_id"])
	nickname, _ := event["nickname"].(string)

	evt := &pb.ChatMessageEvent{
		MsgId:     strconv.FormatInt(time.Now().UnixNano(), 10),
		SenderId:  userId,
		MsgType:   14, // Nickname Update Signal
		Content:   nickname,
		Timestamp: time.Now().UnixMilli(),
		TargetIds: []int64{userId}, // Self (for multi-device)
	}

	// 1. Notify self
	h.pushToGateways(ctx, evt)

	// 2. Notify friends
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	outCtx := metadata.NewOutgoingContext(ctx, md)

	resp, err := h.svcCtx.RelationRpc.GetFriendList(outCtx, &relationservice.GetFriendListRequest{})
	if err == nil && resp != nil {
		var friendIds []int64
		for _, f := range resp.Friends {
			friendIds = append(friendIds, f.UserId)
		}
		if len(friendIds) > 0 {
			evt.TargetIds = friendIds
			h.pushToGateways(ctx, evt)
		}
	}

	return nil
}

func (h *MessageConsumerHandler) handleGroupNicknameUpdate(ctx context.Context, event map[string]interface{}) error {
	groupId := h.toInt64(event["group_id"])
	userId := h.toInt64(event["user_id"])
	nickname, _ := event["nickname"].(string)

	evt := &pb.ChatMessageEvent{
		MsgId:          strconv.FormatInt(time.Now().UnixNano(), 10),
		ConversationId: fmt.Sprintf("group_%d", groupId),
		SenderId:       userId,
		GroupId:        groupId,
		MsgType:        14, // Nickname Update Signal
		Content:        nickname,
		Timestamp:      time.Now().UnixMilli(),
	}

	// pushToGateways handles group member lookup automatically
	h.pushToGateways(ctx, evt)
	return nil
}

func (h *MessageConsumerHandler) handleFriendEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	fromId := h.toInt64(event["from_user_id"])
	toId := h.toInt64(event["to_user_id"])

	if action == "apply" {
		evt := &pb.ChatMessageEvent{
			MsgId:     strconv.FormatInt(time.Now().UnixNano(), 10),
			SenderId:  fromId,
			MsgType:   10, // Friend Apply
			Content:   event["message"].(string),
			Timestamp: time.Now().UnixMilli(),
			TargetIds: []int64{toId},
		}
		h.pushToGateways(ctx, evt)
		return nil
	}

	if action == "reject" {
		evt := &pb.ChatMessageEvent{
			MsgId:     strconv.FormatInt(time.Now().UnixNano(), 10),
			SenderId:  toId,
			MsgType:   11, // Friend Rejected
			Content:   "declined your friend request",
			Timestamp: time.Now().UnixMilli(),
			TargetIds: []int64{fromId},
		}
		h.pushToGateways(ctx, evt)
		return nil
	}

	if action != "accept" {
		return nil
	}

	content := "You are now friends. Say hello!"
	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	convId := h.getPrivateConvId(fromId, toId)

	l := NewSaveMessageLogic(ctx, h.svcCtx)
	evt := &pb.ChatMessageEvent{
		MsgId:          msgId,
		ConversationId: convId,
		SenderId:       0,
		ReceiverId:     0,
		MsgType:        6,
		Content:        content,
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      []int64{fromId, toId},
	}
	_, err := l.SaveMessage(&pb.SaveMessageRequest{
		Message: evt,
	})
	if err == nil {
		h.pushToGateways(ctx, evt)
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
	var signalMsgType int32 = 0

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
		signalMsgType = 12
		targets = []int64{actorId}
	case "kick":
		content = fmt.Sprintf("Admin removed %s from the group", h.getUserNickname(ctx, h.toInt64(event["user_id"])))
		signalMsgType = 12
		targets = []int64{actorId}
	case "dismiss":
		content = "This group has been dismissed by the owner"
		signalMsgType = 13
	case "update_announcement":
		content = fmt.Sprintf("%s updated the group announcement", actorName)
	default:
		return nil
	}

	convId := fmt.Sprintf("group_%d", groupId)
	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
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

	if signalMsgType > 0 {
		sigEvt := &pb.ChatMessageEvent{
			MsgId:          strconv.FormatInt(time.Now().UnixNano(), 10),
			ConversationId: convId,
			SenderId:       0,
			MsgType:        signalMsgType,
			Content:        action,
			Timestamp:      time.Now().UnixMilli(),
			TargetIds:      targets,
		}
		if action == "dismiss" {
			resp, _ := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &groupservice.GetGroupMembersRequest{GroupId: groupId})
			if resp != nil {
				var allIds []int64
				for _, m := range resp.Members {
					allIds = append(allIds, m.UserId)
				}
				sigEvt.TargetIds = allIds
			}
		}
		h.pushToGateways(ctx, sigEvt)
	}

	return err
}

func (h *MessageConsumerHandler) pushToGateways(ctx context.Context, event *pb.ChatMessageEvent) {
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
		targetUsers = []int64{event.SenderId, event.ReceiverId}
	}

	gwMap := make(map[string][]int64)
	for _, uid := range targetUsers {
		if uid <= 0 {
			continue
		}
		addr, err := h.svcCtx.Router.Find(ctx, uid)
		if err == nil && addr != "" {
			gwMap[addr] = append(gwMap[addr], uid)
		}
	}

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
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.svcCtx.HttpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
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
	userResp, err := h.svcCtx.UserRpc.GetUser(ctx, &userservice.GetUserRequest{UserId: userId})
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
