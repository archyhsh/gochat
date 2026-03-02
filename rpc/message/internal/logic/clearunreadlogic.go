package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClearUnreadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewClearUnreadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearUnreadLogic {
	return &ClearUnreadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ClearUnreadLogic) ClearUnread(in *pb.ClearUnreadRequest) (*pb.ClearUnreadResponse, error) {
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
	userConversation, err := l.svcCtx.UserConversationModel.FindUserConversationsByUserIdAndConversationId(userId, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user conversation not found")
		} else {
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	userConversation.UnreadCount = 0
	err = l.svcCtx.UserConversationModel.Update(l.ctx, userConversation)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update user conversation")
	}
	return &pb.ClearUnreadResponse{Base: &pb.BaseResponse{Code: 200, Message: "success"}}, nil
}
