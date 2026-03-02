// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

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

type GetCurrentUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentUserLogic {
	return &GetCurrentUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentUserLogic) GetCurrentUser() (resp *types.User, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.UserRpc.GetCurrentUser(ctx, &pb.GetCurrentUserRequest{})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call userRPC func GetCurrentUser"+err.Error())
	}

	return &types.User{
		Id:       rpcResp.User.Id,
		Username: rpcResp.User.Username,
		Nickname: rpcResp.User.Nickname,
		Avatar:   rpcResp.User.Avatar,
		Phone:    rpcResp.User.Phone,
		Email:    rpcResp.User.Email,
		Gender:   int(rpcResp.User.Gender),
	}, nil
}
