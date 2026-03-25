package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

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
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}

	err = l.svcCtx.UserConversationModel.Restore(l.ctx, userId, in.ConversationId)
	if err != nil {
		l.Errorf("RestoreConversation failed: %v", err)
		return nil, err
	}

	return &pb.RestoreConversationResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
