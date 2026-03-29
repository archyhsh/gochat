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

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.User, err error) {
	rpcResp, err := l.svcCtx.UserRpc.Register(l.ctx, &pb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Nickname: req.Nickname,
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call UserRpc func register"+err.Error())
	}

	return &types.User{
		Id:       rpcResp.User.Id,
		Username: rpcResp.User.Username,
		Nickname: rpcResp.User.Nickname,
		Avatar:   rpcResp.User.Avatar,
		Email:    rpcResp.User.Email,
		Phone:    rpcResp.User.Phone,
		Gender:   int(rpcResp.User.Gender),
	}, nil
}
