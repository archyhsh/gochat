package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFriendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFriendLogic {
	return &DeleteFriendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteFriendLogic) DeleteFriend(in *pb.DeleteFriendRequest) (*pb.DeleteFriendResponse, error) {
	// Todo: get userId from context
	userId := int64(1)
	if userId == in.FriendId {
		return nil, status.Error(codes.InvalidArgument, "cannot delete yourself")
	}
	_, err := l.svcCtx.FriendshipModel.FindOneByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err != nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "cannot find friendship")
	}
	if err != nil && err == model.ErrNotFound {
		return nil, status.Error(codes.NotFound, "target is not your friend")
	}

	err = l.svcCtx.FriendshipModel.DeleteFriendshipByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete friend")
	}
	return &pb.DeleteFriendResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}}, nil
}
