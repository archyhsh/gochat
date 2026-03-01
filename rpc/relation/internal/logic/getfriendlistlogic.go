package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendListLogic {
	return &GetFriendListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFriendListLogic) GetFriendList(in *pb.GetFriendListRequest) (*pb.GetFriendListResponse, error) {
	UserId := int64(1) // TODO: get user id from context
	friendships, err := l.svcCtx.FriendshipModel.FindNormalFriendListByUserId(l.ctx, UserId)
	if err != nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "Failed to get friend list")
	}
	if len(friendships) == 0 {
		return &pb.GetFriendListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Friends: []*pb.UserAvatar{}}, nil
	}
	var friendIds []int64
	remarkMap := make(map[int64]string)
	for _, f := range friendships {
		friendIds = append(friendIds, f.FriendId)
		remarkMap[f.FriendId] = f.Remark
	}
	userResp, err := l.svcCtx.UserRpc.GetUsersByIds(l.ctx, &userservice.GetUsersByIdsRequest{
		UserIds: friendIds,
	})
	if err != nil {
		l.Errorf("批量拉取用户资料失败: %v", err)
		var friendList []*pb.UserAvatar
		for _, u := range userResp.Users {
			friendList = append(friendList, &pb.UserAvatar{
				UserId: u.Id,
				Remark: remarkMap[u.Id],
			})
		}
		return &pb.GetFriendListResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}, Friends: friendList}, status.Error(codes.Internal, "batch get friend info failed, return basic info")
	}

	var friendList []*pb.UserAvatar
	for _, u := range userResp.Users {
		friendList = append(friendList, &pb.UserAvatar{
			UserId:   u.Id,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Remark:   remarkMap[u.Id],
		})
	}
	return &pb.GetFriendListResponse{
		Base:    &pb.BaseResponse{Code: 200, Message: "Success"},
		Friends: friendList,
	}, nil
}
