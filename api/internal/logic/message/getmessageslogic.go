// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessagesLogic {
	return &GetMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessagesLogic) GetMessages(req *types.GetMessagesRequest) (resp *types.MessagesResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.MessageRpc.GetMessages(ctx, &pb.GetMessagesRequest{
		ConversationId: req.ConversationId,
		Limit:          int32(req.Limit),
		Offset:         int32(req.Offset),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call MessageRpc func GetMessages: "+err.Error())
	}

	messages := make([]types.Message, 0)
	for _, msg := range rpcResp.Messages {
		messages = append(messages, types.Message{
			MsgId:          msg.MsgId,
			ConversationId: msg.ConversationId,
			SenderId:       msg.SenderId,
			Content:        msg.Content,
			MsgType:        int(msg.MsgType),
			Timestamp:      msg.Timestamp,
		})
	}

	return &types.MessagesResponse{
		Messages: messages,
	}, nil
}
