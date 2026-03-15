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
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
	}

	// Piggybacking: Fetch versions
	userResp, err := h.svcCtx.UserRpc.GetUser(ctx, &userservice.GetUserRequest{UserId: event.SenderId})
	if err == nil && userResp.User != nil {
		event.SenderInfoVersion = userResp.User.InfoVersion
	}

	if event.GroupId > 0 {
		groupResp, err := h.svcCtx.GroupRpc.GetGroupInfo(ctx, &groupservice.GetGroupInfoRequest{GroupId: event.GroupId})
		if err == nil && groupResp.Group != nil {
			event.GroupMetaVersion = groupResp.Group.MetaVersion
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
		action, _ := raw["action"].(string)
		if action == "update_remark" {
			return h.handleRemarkUpdate(ctx, raw)
		}
		return h.handleFriendEvent(ctx, raw)
	case "group_event":
		return h.handleGroupEvent(ctx, raw)
	}
	return nil
}

func (h *MessageConsumerHandler) handleRemarkUpdate(ctx context.Context, event map[string]interface{}) error {
	userId := h.toInt64(event["user_id"])
	friendId := h.toInt64(event["friend_id"])
	remark, _ := event["remark"].(string)
	version := h.toInt64(event["version"])

	evt := &pb.ChatMessageEvent{
		MsgId:           strconv.FormatInt(time.Now().UnixNano(), 10),
		SenderId:        friendId,
		MsgType:         15, // Remark Sync Signal
		Content:         remark,
		Timestamp:       time.Now().UnixMilli(),
		TargetIds:       []int64{userId},
		RelationVersion: version,
	}
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
			MsgType:   10,
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
			MsgType:   11,
			Content:   "declined your friend request",
			Timestamp: time.Now().UnixMilli(),
			TargetIds: []int64{fromId},
		}
		h.pushToGateways(ctx, evt)
		return nil
	}

	if action == "delete" {
		evt := &pb.ChatMessageEvent{
			MsgId:     strconv.FormatInt(time.Now().UnixNano(), 10),
			SenderId:  fromId,
			MsgType:   14, // Signal: Friend Deleted
			Content:   "deleted you from friend list",
			Timestamp: time.Now().UnixMilli(),
			TargetIds: []int64{toId},
		}
		h.pushToGateways(ctx, evt)
		return nil
	}

	if action != "accept" {
		return nil
	}

	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	convId := h.getPrivateConvId(fromId, toId)
	evt := &pb.ChatMessageEvent{
		MsgId:          msgId,
		ConversationId: convId,
		SenderId:       0,
		MsgType:        6,
		Content:        "You are now friends. Say hello!",
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      []int64{fromId, toId},
	}
	l := NewSaveMessageLogic(ctx, h.svcCtx)
	_, _ = l.SaveMessage(&pb.SaveMessageRequest{Message: evt})
	h.pushToGateways(ctx, evt)
	return nil
}

func (h *MessageConsumerHandler) handleGroupEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	groupId := h.toInt64(event["group_id"])
	actorId := h.toInt64(event["user_id"])

	var content string
	var targets []int64
	var signalType int32 = 0

	switch action {
	case "create":
		content = "Group created"
		targets = []int64{actorId}
	case "join":
		content = "A new member joined"
		targets = []int64{actorId}
	case "reject":
		// Handle Group Application Rejection
		sig := &pb.ChatMessageEvent{
			MsgId:     strconv.FormatInt(time.Now().UnixNano(), 10),
			SenderId:  0,  // System
			MsgType:   16, // Signal: Group Request Rejected
			GroupId:   groupId,
			Content:   "rejected",
			Timestamp: time.Now().UnixMilli(),
			TargetIds: []int64{actorId}, // Notify applicant
		}
		h.pushToGateways(ctx, sig)
		return nil
	case "invite":
		var inviteeIds []int64
		targets = []int64{actorId}
		if rawIds, ok := event["member_ids"].([]interface{}); ok {
			for _, rid := range rawIds {
				uid := h.toInt64(rid)
				inviteeIds = append(inviteeIds, uid)
				targets = append(targets, uid)
			}
		}

		var inviteeNames []string
		if len(inviteeIds) > 0 {
			userResp, _ := h.svcCtx.UserRpc.GetUsersByIds(ctx, &userservice.GetUsersByIdsRequest{UserIds: inviteeIds})
			if userResp != nil {
				for _, u := range userResp.Users {
					inviteeNames = append(inviteeNames, u.Nickname)
				}
			}
		}
		content = fmt.Sprintf("%s has been invited to the group", strings.Join(inviteeNames, ", "))
	case "quit", "kick":
		content = "Member left the group"
		signalType = 12
		targets = []int64{actorId}
	case "dismiss":
		content = "The group has been dismissed"
		signalType = 13
	case "update_announcement":
		content = "Group announcement updated"
	default:
		return nil
	}

	convId := fmt.Sprintf("group_%d", groupId)
	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	evt := &pb.ChatMessageEvent{
		MsgId: msgId, ConversationId: convId, SenderId: 0, GroupId: groupId,
		MsgType: 6, Content: content, Timestamp: time.Now().UnixMilli(), TargetIds: targets,
	}
	l := NewSaveMessageLogic(ctx, h.svcCtx)
	_, err := l.SaveMessage(&pb.SaveMessageRequest{
		Message: evt,
	})
	if err == nil {
		h.pushToGateways(ctx, evt)
	}

	if signalType > 0 {
		sig := &pb.ChatMessageEvent{
			MsgId: strconv.FormatInt(time.Now().UnixNano(), 10), ConversationId: convId,
			MsgType: signalType, Content: action, Timestamp: time.Now().UnixMilli(), TargetIds: targets,
			GroupId: groupId,
		}
		if action == "dismiss" {
			resp, _ := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &groupservice.GetGroupMembersRequest{GroupId: groupId})
			if resp != nil {
				for _, m := range resp.Members {
					sig.TargetIds = append(sig.TargetIds, m.UserId)
				}
			}
		}
		h.pushToGateways(ctx, sig)
	}
	return nil
}

func (h *MessageConsumerHandler) pushToGateways(ctx context.Context, event *pb.ChatMessageEvent) {
	var targetUsers []int64
	if len(event.TargetIds) > 0 {
		targetUsers = event.TargetIds
	} else if event.GroupId > 0 {
		resp, err := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &groupservice.GetGroupMembersRequest{GroupId: event.GroupId})
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

		// Filter blocked users for private messages
		if event.GroupId == 0 && event.SenderId > 0 && uid != event.SenderId {
			checkResp, err := h.svcCtx.RelationRpc.CheckFriend(ctx, &pb.CheckFriendRequest{
				UserId:   uid,
				FriendId: event.SenderId,
			})
			if err == nil && checkResp.IsBlocked {
				continue
			}
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
		"user_ids":            userIds,
		"conversation_id":     event.ConversationId,
		"msg_id":              event.MsgId,
		"sender_id":           event.SenderId,
		"content":             event.Content,
		"msg_type":            int(event.MsgType),
		"timestamp":           event.Timestamp,
		"sender_info_version": event.SenderInfoVersion,
		"group_meta_version":  event.GroupMetaVersion,
		"relation_version":    event.RelationVersion,
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.svcCtx.HttpClient.Do(req)
	if err == nil {
		resp.Body.Close()
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
