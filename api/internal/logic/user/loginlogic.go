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

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	rpcResp, err := l.svcCtx.UserRpc.Login(l.ctx, &pb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call UserRpc func login"+err.Error())
	}
	return &types.LoginResponse{
		Token: rpcResp.Token,
		User: types.User{
			Id:       rpcResp.User.Id,
			Username: rpcResp.User.Username,
			Nickname: rpcResp.User.Nickname,
			Avatar:   rpcResp.User.Avatar,
			Email:    rpcResp.User.Email,
			Phone:    rpcResp.User.Phone,
			Gender:   int(rpcResp.User.Gender),
		},
	}, nil
}
