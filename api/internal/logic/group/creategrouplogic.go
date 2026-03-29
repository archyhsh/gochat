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

type CreateGroupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateGroupLogic) CreateGroup(req *types.CreateGroupRequest) (resp *types.GroupInfo, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.GroupRpc.CreateGroup(ctx, &pb.CreateGroupRequest{
		Name:        req.Name,
		Avatar:      req.Avatar,
		Description: req.Description,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func creategroup"+err.Error())
	}
	return &types.GroupInfo{
		GroupId:     rpcResp.Group.Id,
		Name:        rpcResp.Group.Name,
		Avatar:      rpcResp.Group.Avatar,
		Description: rpcResp.Group.Description,
		OwnerId:     rpcResp.Group.OwnerId,
	}, nil
}
