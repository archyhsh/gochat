package user

import (
	"context"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ForgotPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewForgotPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForgotPasswordLogic {
	return &ForgotPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ForgotPasswordLogic) ForgotPassword(req *types.ForgotPasswordRequest) (resp *types.CommonResponse, err error) {
	_, err = l.svcCtx.UserRpc.ForgotPassword(l.ctx, &pb.ForgotPasswordRequest{
		Username:    req.Username,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return nil, err
	}

	return &types.CommonResponse{
		Message: "Success",
	}, nil
}
