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

	"github.com/IBM/sarama"
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

	msg := &sarama.ProducerMessage{
		Topic: l.svcCtx.Config.Kafka.Topic,
		Value: sarama.ByteEncoder(data),
		Key:   sarama.StringEncoder(req.ConversationId),
	}

	_, _, err = l.svcCtx.KafkaProducer.SendMessage(msg)
	if err != nil {
		l.Errorf("Failed to send message to Kafka: %v", err)
		return nil, err
	}

	return &types.SendMessageResponse{
		MsgId:     msgId,
		Timestamp: now,
	}, nil
}
