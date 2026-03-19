package message

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

type SendMessageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageLogic {
	return &SendMessageLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendMessageLogic) SendMessage(req *types.SendMessageRequest) (resp *types.SendMessageResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, fmt.Errorf("unauthorized")
	}

	// Block check for private messages
	if req.ReceiverId > 0 && req.GroupId == 0 {
		checkResp, err := l.svcCtx.RelationRpc.CheckFriend(l.ctx, &pb.CheckFriendRequest{
			UserId:   req.ReceiverId,
			FriendId: userId,
		})
		if err == nil {
			if !checkResp.IsFriend {
				return nil, fmt.Errorf("you are not friends with this user")
			}
			if checkResp.IsBlocked {
				return nil, fmt.Errorf("you have been blocked by this user")
			}
		}
	}

	msgId := strconv.FormatInt(snowflake.MustNextID(), 10)
	now := time.Now().UnixMilli()

	event := &pb.ChatMessageEvent{
		MsgId:          msgId,
		ConversationId: req.ConversationId,
		SenderId:       userId,
		ReceiverId:     req.ReceiverId,
		GroupId:        req.GroupId,
		Content:        req.Content,
		MsgType:        int32(req.MsgType),
		Timestamp:      now,
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return nil, err
	}

	err = l.svcCtx.KafkaProducer.Send([]byte(req.ConversationId), data)
	if err != nil {
		l.Errorf("Failed to send message to Kafka after retries: %v", err)
		return nil, err
	}

	return &types.SendMessageResponse{
		MsgId:     msgId,
		Timestamp: now,
	}, nil
}
