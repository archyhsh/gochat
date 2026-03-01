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

type CheckFriendLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckFriendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckFriendLogic {
	return &CheckFriendLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckFriendLogic) CheckFriend(in *pb.CheckFriendRequest) (*pb.CheckFriendResponse, error) {
	friendship, err := l.svcCtx.FriendshipModel.FindOneByUserIdFriendId(l.ctx, in.UserId, in.FriendId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.CheckFriendResponse{
				IsFriend:  false,
				IsBlocked: false,
			}, nil
		} else {
			return nil, status.Error(codes.Internal, "failed to check friend")
		}
	}
	return &pb.CheckFriendResponse{
		IsFriend:  true,
		IsBlocked: friendship.Status == 1,
	}, nil
}
