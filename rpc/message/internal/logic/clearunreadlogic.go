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

	conv, err := l.svcCtx.ConversationModel.FindOneByConversationId(l.ctx, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "Conversation not found")
		}
		return nil, status.Error(codes.Internal, "Internal database error")
	}

	err = l.svcCtx.UserConversationModel.UpdateReadSequence(l.ctx, userId, in.ConversationId, conv.LatestSeq)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to clear unread: "+err.Error())
	}

	return &pb.ClearUnreadResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
