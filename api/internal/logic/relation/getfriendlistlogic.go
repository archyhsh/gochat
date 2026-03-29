// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package relation

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

type GetFriendListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendListLogic {
	return &GetFriendListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFriendListLogic) GetFriendList() (resp *types.FriendListResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.RelationRpc.GetFriendList(ctx, &pb.GetFriendListRequest{})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call relationRPC func GetFriendList"+err.Error())
	}

	friendList := make([]types.FriendInfo, 0)
	for _, friend := range rpcResp.Friends {
		friendList = append(friendList, types.FriendInfo{
			UserId:   friend.UserId,
			Nickname: friend.Nickname,
			Avatar:   friend.Avatar,
			Remark:   friend.Remark,
		})
	}

	return &types.FriendListResponse{
		Friends: friendList,
	}, nil
}
