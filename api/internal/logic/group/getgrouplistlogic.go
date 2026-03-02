// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

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

type GetGroupListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetGroupListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupListLogic {
	return &GetGroupListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupListLogic) GetGroupList() (resp *types.GroupListResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.GroupRpc.GetGroupList(ctx, &pb.GetGroupListRequest{})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func GetGroupList"+err.Error())
	}
	var groups []types.GroupInfo
	for _, group := range rpcResp.Groups {
		groups = append(groups, types.GroupInfo{
			GroupId: group.GroupId,
			Name:    group.Name,
			Avatar:  group.Avatar,
		})
	}
	return &types.GroupListResponse{
		Groups: groups,
	}, nil
}
