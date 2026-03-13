package message

import (
	"context"
	"fmt"
	"strconv"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/message/messageservice"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
)

type RestoreConversationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreConversationLogic {
	return &RestoreConversationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestoreConversationLogic) RestoreConversation(req *types.RestoreConversationRequest) (resp *types.CommonResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, fmt.Errorf("user not login")
	}

	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	_, err = l.svcCtx.MessageRpc.RestoreConversation(ctx, &messageservice.RestoreConversationRequest{
		UserId:         userId,
		ConversationId: req.ConversationId,
	})
	if err != nil {
		return nil, err
	}

	return &types.CommonResponse{
		Message: "Success",
	}, nil
}
