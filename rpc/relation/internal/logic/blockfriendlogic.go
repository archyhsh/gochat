package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type BlockFriendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBlockFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BlockFriendLogic {
	return &BlockFriendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BlockFriendLogic) BlockFriend(in *pb.BlockFriendRequest) (*pb.BlockFriendResponse, error) {
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

	if userId == in.FriendId {
		return nil, status.Error(codes.InvalidArgument, "cannot block yourself")
	}
	friendship, err := l.svcCtx.FriendshipModel.FindOneByUserIdFriendId(l.ctx, userId, in.FriendId)
	if err == nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "cannot find friendship")
	}
	if err == nil && err == model.ErrNotFound {
		return nil, status.Error(codes.NotFound, "target is not your friend")
	}
	friendship.Status = 1
	err = l.svcCtx.FriendshipModel.Update(l.ctx, friendship)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to block friend")
	}
	return &pb.BlockFriendResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}}, nil
}
