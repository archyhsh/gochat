// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserLogic) GetUser(req *types.GetUserRequest) (resp *types.User, err error) {
	rpcResp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &pb.GetUserRequest{
		UserId: req.Id,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call UserRpc func GetUser"+err.Error())
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
