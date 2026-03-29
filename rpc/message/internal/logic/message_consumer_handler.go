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

	"sync"

	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/protobuf/proto"

	"github.com/IBM/sarama"
)

type MessageConsumerHandler struct {
	svcCtx *svc.ServiceContext
	logx.Logger
	userCache     sync.Map // uid -> version
	groupCache    sync.Map // gid -> version
	relationCache sync.Map // convId -> version
}

// eventImpact represents the outcome of a system event
type eventImpact struct {
	ConversationId string
	GroupId        int64
	Content        string
	Targets        []int64
	SignalType     int32
	Broadcast      bool // Whether to push to all members
}

func NewMessageConsumerHandler(svcCtx *svc.ServiceContext) *MessageConsumerHandler {
	return &MessageConsumerHandler{
		svcCtx: svcCtx,
		Logger: logx.WithContext(context.Background()),
	}
}

// --- Main Entry ---

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

// --- Chat Message Handler ---

func (h *MessageConsumerHandler) handleChatMessage(ctx context.Context, data []byte) error {
	var event pb.ChatMessageEvent
	if err := proto.Unmarshal(data, &event); err != nil {
		if err := json.Unmarshal(data, &event); err != nil {
			return err
		}
	}

	// Attach metadata versions from cache
	event.SenderInfoVersion = h.getUserVersion(ctx, event.SenderId)
	if event.GroupId > 0 {
		event.GroupMetaVersion = h.getGroupVersion(ctx, event.GroupId)
	} else if event.ReceiverId > 0 {
		event.RelationVersion = h.getRelationVersion(ctx, event.SenderId, event.ReceiverId)
	}

	l := NewSaveMessageLogic(ctx, h.svcCtx)
	resp, err := l.SaveMessage(&pb.SaveMessageRequest{Message: &event})
	if err == nil && resp != nil {
		event.Sequence = resp.Sequence
		h.pushToGateways(ctx, &event)
	}
	return err
}

// --- System Event Router ---

func (h *MessageConsumerHandler) handleSystemEvent(ctx context.Context, data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	eventType, _ := raw["type"].(string)
	switch eventType {
	case "user_event":
		return h.handleUserEvent(ctx, raw)
	case "friend_event":
		return h.handleFriendEvent(ctx, raw)
	case "group_event":
		return h.handleGroupEvent(ctx, raw)
	}
	return nil
}

// --- User Profile Handler ---

func (h *MessageConsumerHandler) handleUserEvent(ctx context.Context, event map[string]interface{}) error {
	userId := h.toInt64(event["user_id"])
	version := h.toInt64(event["info_version"])
	if userId <= 0 {
		h.Errorf("[handleUserEvent] invalid userId: %d", userId)
		return nil
	}

	// 1. Invalidate Caches
	h.userCache.Delete(userId)
	key := fmt.Sprintf("cache:user:version:%d", userId)
	_, _ = h.svcCtx.Redis.Del(key)

	// 2. Precise Notification: Find people having conversation with this user
	uids, err := h.svcCtx.UserConversationModel.GetUsersByPeerId(ctx, userId)
	if err == nil && len(uids) > 0 {
		sig := &pb.ChatMessageEvent{
			MsgId:             strconv.FormatInt(time.Now().UnixNano(), 10),
			SenderId:          userId,
			MsgType:           18, // METADATA_INVALIDATION
			Content:           "profile_updated",
			Timestamp:         time.Now().UnixMilli(),
			TargetIds:         uids,
			SenderInfoVersion: version,
		}
		h.pushToGateways(ctx, sig)
	}

	return nil
}

// --- Friend/Relation Handlers ---

func (h *MessageConsumerHandler) handleFriendEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	if action == "" {
		h.Errorf("[handleFriendEvent] missing action in event: %v", event)
		return nil
	}

	impact := &eventImpact{}

	switch action {
	case "update_remark":
		return h.handleFriendRemarkUpdate(ctx, event)
	case "apply":
		impact.Content = event["message"].(string)
		impact.Targets = []int64{h.toInt64(event["to_user_id"])}
		impact.SignalType = 10
	case "reject":
		impact.Content = "declined your friend request"
		impact.Targets = []int64{h.toInt64(event["from_user_id"])}
		impact.SignalType = 11
	case "delete":
		impact.Content = "deleted you from friend list"
		impact.Targets = []int64{h.toInt64(event["to_user_id"])}
		impact.SignalType = 14
	case "accept":
		return h.handleFriendAccept(ctx, event)
	default:
		return nil
	}

	return h.processEventMessage(ctx, impact)
}

func (h *MessageConsumerHandler) handleFriendRemarkUpdate(ctx context.Context, event map[string]interface{}) error {
	userId := h.toInt64(event["user_id"])
	friendId := h.toInt64(event["friend_id"])
	version := h.toInt64(event["version"])

	// Update cache
	convId := h.getPrivateConvId(userId, friendId)
	key := fmt.Sprintf("cache:relation:version:%s", convId)
	_ = h.svcCtx.Redis.Setex(key, strconv.FormatInt(version, 10), 3600*24)
	h.relationCache.Store(convId, version)

	// Send Remark Sync Signal
	sig := &pb.ChatMessageEvent{
		MsgId:           strconv.FormatInt(time.Now().UnixNano(), 10),
		SenderId:        friendId,
		MsgType:         15,
		Content:         event["remark"].(string),
		Timestamp:       time.Now().UnixMilli(),
		TargetIds:       []int64{userId},
		RelationVersion: version,
	}
	h.pushToGateways(ctx, sig)
	return nil
}

func (h *MessageConsumerHandler) handleFriendAccept(ctx context.Context, event map[string]interface{}) error {
	fromId := h.toInt64(event["from_user_id"])
	toId := h.toInt64(event["to_user_id"])
	impact := &eventImpact{
		ConversationId: h.getPrivateConvId(fromId, toId),
		Content:        "You are now friends. Say hello!",
		Targets:        []int64{fromId, toId},
	}
	return h.processEventMessage(ctx, impact)
}

// --- Group Event Handlers ---

func (h *MessageConsumerHandler) handleGroupEvent(ctx context.Context, event map[string]interface{}) error {
	action, _ := event["action"].(string)
	groupId := h.toInt64(event["group_id"])
	actorId := h.toInt64(event["user_id"])

	if action == "" || groupId <= 0 {
		h.Errorf("[handleGroupEvent] invalid event data: action=%s, groupId=%d", action, groupId)
		return nil
	}

	// Invalidate cache
	if groupId > 0 {
		h.groupCache.Delete(groupId)
		key := fmt.Sprintf("cache:group:version:%d", groupId)
		_, _ = h.svcCtx.Redis.Del(key)
	}

	impact := &eventImpact{
		ConversationId: fmt.Sprintf("group_%d", groupId),
		GroupId:        groupId,
	}

	switch action {
	case "create":
		impact.Content = "Group created"
		impact.Targets = []int64{actorId}
	case "join":
		impact.Content = "A new member joined"
		impact.Targets = []int64{actorId}
	case "reject":
		impact.Content = "rejected"
		impact.Targets = []int64{actorId}
		impact.SignalType = 16
	case "invite":
		return h.handleGroupInvite(ctx, event, impact)
	case "quit", "kick":
		impact.Content = "Member left the group"
		impact.Targets = []int64{actorId}
		impact.SignalType = 12
	case "dismiss":
		impact.Content = "The group has been dismissed"
		impact.Broadcast = true
		impact.SignalType = 13
	case "update_announcement":
		impact.Content = "Group announcement updated"
		impact.Broadcast = true
	default:
		return nil
	}

	return h.processEventMessage(ctx, impact)
}

func (h *MessageConsumerHandler) handleGroupInvite(ctx context.Context, event map[string]interface{}, impact *eventImpact) error {
	actorId := h.toInt64(event["user_id"])
	var inviteeIds []int64
	impact.Targets = []int64{actorId}

	if rawIds, ok := event["member_ids"].([]interface{}); ok {
		for _, rid := range rawIds {
			uid := h.toInt64(rid)
			inviteeIds = append(inviteeIds, uid)
			impact.Targets = append(impact.Targets, uid)
		}
	}

	var inviteeNames []string
	if len(inviteeIds) > 0 {
		userResp, _ := h.svcCtx.UserRpc.GetUsersByIds(ctx, &pb.GetUsersByIdsRequest{UserIds: inviteeIds})
		if userResp != nil {
			for _, u := range userResp.Users {
				inviteeNames = append(inviteeNames, u.Nickname)
			}
		}
	}
	impact.Content = fmt.Sprintf("%s has been invited to the group", strings.Join(inviteeNames, ", "))
	return h.processEventMessage(ctx, impact)
}

// --- Universal Execution Engine ---

func (h *MessageConsumerHandler) processEventMessage(ctx context.Context, impact *eventImpact) error {
	if impact == nil || impact.Content == "" {
		return nil
	}

	evt := &pb.ChatMessageEvent{
		MsgId:          strconv.FormatInt(snowflake.MustNextID(), 10),
		ConversationId: impact.ConversationId,
		SenderId:       0, // System
		GroupId:        impact.GroupId,
		MsgType:        6, // System message
		Content:        impact.Content,
		Timestamp:      time.Now().UnixMilli(),
		TargetIds:      impact.Targets,
	}

	// 1. Persistence (Save to history)
	l := NewSaveMessageLogic(ctx, h.svcCtx)
	resp, _ := l.SaveMessage(&pb.SaveMessageRequest{Message: evt})
	if resp != nil {
		evt.Sequence = resp.Sequence
	}

	// 2. Resolve broadcast members if needed
	if impact.Broadcast && impact.GroupId > 0 {
		resp, err := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &pb.GetGroupMembersRequest{GroupId: impact.GroupId})
		if err == nil {
			for _, m := range resp.Members {
				evt.TargetIds = append(evt.TargetIds, m.UserId)
			}
		}
	}

	// 3. Real-time Pushing
	if len(evt.TargetIds) > 0 || impact.Broadcast {
		h.pushToGateways(ctx, evt)
	}

	// 4. Signal Pushing (if separate from content)
	if impact.SignalType > 0 {
		sig := &pb.ChatMessageEvent{
			MsgId:          strconv.FormatInt(time.Now().UnixNano(), 10),
			ConversationId: impact.ConversationId,
			MsgType:        impact.SignalType,
			Content:        impact.Content,
			Timestamp:      time.Now().UnixMilli(),
			TargetIds:      evt.TargetIds,
			GroupId:        impact.GroupId,
			Sequence:       evt.Sequence,
		}
		h.pushToGateways(ctx, sig)
	}

	return nil
}

// --- Infrastructure & Pushing ---

func (h *MessageConsumerHandler) pushToGateways(ctx context.Context, event *pb.ChatMessageEvent) {
	var targetUsers []int64
	if len(event.TargetIds) > 0 {
		targetUsers = event.TargetIds
	} else if event.GroupId > 0 {
		resp, err := h.svcCtx.GroupRpc.GetGroupMembers(ctx, &pb.GetGroupMembersRequest{GroupId: event.GroupId})
		if err == nil {
			for _, m := range resp.Members {
				targetUsers = append(targetUsers, m.UserId)
			}
		}
	} else {
		targetUsers = []int64{event.SenderId, event.ReceiverId}
	}

	if len(targetUsers) == 0 {
		return
	}

	addrMap, err := h.svcCtx.Router.BatchFind(ctx, targetUsers)
	if err != nil {
		h.Errorf("Failed to batch find routes: %v", err)
		return
	}

	gwMap := make(map[string][]int64)
	for uid, addr := range addrMap {
		if uid <= 0 || addr == "" {
			continue
		}
		// Block check for private messages
		if event.GroupId == 0 && event.SenderId > 0 && uid != event.SenderId {
			checkResp, err := h.svcCtx.RelationRpc.CheckFriend(ctx, &pb.CheckFriendRequest{UserId: uid, FriendId: event.SenderId})
			if err == nil && checkResp.IsBlocked {
				continue
			}
		}
		gwMap[addr] = append(gwMap[addr], uid)
	}

	for addr, uids := range gwMap {
		// Calculate unread per user if private, or push common event
		go h.sendBatchPush(ctx, addr, uids, event)
	}
}

func (h *MessageConsumerHandler) sendBatchPush(ctx context.Context, addr string, uids []int64, event *pb.ChatMessageEvent) {
	// For performance, we send the same event to the gateway, but gateway needs to know target UIDs.
	// In the future, we could customize the payload per UID if unread counts differ.
	h.sendPushRequest(ctx, addr, uids, event)
}

func (h *MessageConsumerHandler) sendPushRequest(ctx context.Context, gwAddr string, userIds []int64, event *pb.ChatMessageEvent) {
	url := fmt.Sprintf("http://%s/internal/push", gwAddr)

	// In cluster mode, we send a rich payload to enable "Pull-free" UI updates.
	// For private messages, we try to fetch the specific unread count from Redis for the receiver.
	unreadMap := make(map[int64]int64)
	if event.GroupId == 0 && event.ReceiverId > 0 {
		key := fmt.Sprintf("unread:cnt:%d:%s", event.ReceiverId, event.ConversationId)
		val, err := h.svcCtx.Redis.GetCtx(ctx, key)
		if err == nil && val != "" {
			u, _ := strconv.ParseInt(val, 10, 64)
			unreadMap[event.ReceiverId] = u
		}
	}

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
		"sequence":            event.Sequence,
		"unread_map":          unreadMap, // uid -> unread_count
	}

	data, _ := json.Marshal(payload)

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
		if err != nil {
			h.Errorf("Failed to create push request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Inject Trace Context into HTTP headers
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

		resp, err := h.svcCtx.HttpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * 200 * time.Millisecond)
		}
	}
	dlqKey := "queue:push:failed"
	_, _ = h.svcCtx.Redis.Lpush(dlqKey, string(data))
}

// --- Helpers & Caching ---

func (h *MessageConsumerHandler) toInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	if f, ok := val.(float64); ok {
		return int64(f)
	}
	if i, ok := val.(int64); ok {
		return i
	}
	if i, ok := val.(int); ok {
		return int64(i)
	}
	if s, ok := val.(string); ok {
		id, _ := strconv.ParseInt(s, 10, 64)
		return id
	}
	h.Debugf("[toInt64] unexpected type for value: %T (%v)", val, val)
	return 0
}

func (h *MessageConsumerHandler) getPrivateConvId(uid1, uid2 int64) string {
	if uid1 < uid2 {
		return fmt.Sprintf("conv_%d_%d", uid1, uid2)
	}
	return fmt.Sprintf("conv_%d_%d", uid2, uid1)
}

func (h *MessageConsumerHandler) getUserVersion(ctx context.Context, userId int64) int64 {
	if userId <= 0 {
		return 0
	}
	if v, ok := h.userCache.Load(userId); ok {
		return v.(int64)
	}
	key := fmt.Sprintf("cache:user:version:%d", userId)
	val, err := h.svcCtx.Redis.Get(key)
	if err == nil && val != "" {
		v, _ := strconv.ParseInt(val, 10, 64)
		h.userCache.Store(userId, v)
		return v
	}
	userResp, err := h.svcCtx.UserRpc.GetUser(ctx, &pb.GetUserRequest{UserId: userId})
	if err == nil && userResp.User != nil {
		version := userResp.User.InfoVersion
		_ = h.svcCtx.Redis.Setex(key, strconv.FormatInt(version, 10), 3600*24)
		h.userCache.Store(userId, version)
		return version
	}
	return 0
}

func (h *MessageConsumerHandler) getGroupVersion(ctx context.Context, groupId int64) int64 {
	if groupId <= 0 {
		return 0
	}
	if v, ok := h.groupCache.Load(groupId); ok {
		return v.(int64)
	}
	key := fmt.Sprintf("cache:group:version:%d", groupId)
	val, err := h.svcCtx.Redis.Get(key)
	if err == nil && val != "" {
		v, _ := strconv.ParseInt(val, 10, 64)
		h.groupCache.Store(groupId, v)
		return v
	}
	groupResp, err := h.svcCtx.GroupRpc.GetGroupInfo(ctx, &pb.GetGroupInfoRequest{GroupId: groupId})
	if err == nil && groupResp.Group != nil {
		version := groupResp.Group.MetaVersion
		_ = h.svcCtx.Redis.Setex(key, strconv.FormatInt(version, 10), 3600*24)
		h.groupCache.Store(groupId, version)
		return version
	}
	return 0
}

func (h *MessageConsumerHandler) getRelationVersion(ctx context.Context, uid1, uid2 int64) int64 {
	convId := h.getPrivateConvId(uid1, uid2)
	if v, ok := h.relationCache.Load(convId); ok {
		return v.(int64)
	}
	key := fmt.Sprintf("cache:relation:version:%s", convId)
	val, err := h.svcCtx.Redis.Get(key)
	if err == nil && val != "" {
		v, _ := strconv.ParseInt(val, 10, 64)
		h.relationCache.Store(convId, v)
		return v
	}
	return 0
}
