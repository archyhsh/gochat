package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRestoreConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreConversationLogic {
	return &RestoreConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RestoreConversationLogic) RestoreConversation(in *pb.RestoreConversationRequest) (*pb.RestoreConversationResponse, error) {
	err := l.svcCtx.UserConversationModel.Restore(l.ctx, in.UserId, in.ConversationId)
	if err != nil {
		l.Errorf("RestoreConversation failed: %v", err)
		return nil, err
	}

	return &pb.RestoreConversationResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
