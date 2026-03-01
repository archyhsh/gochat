package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetCurrentUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentUserLogic {
	return &GetCurrentUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetCurrentUserLogic) GetCurrentUser(in *pb.GetCurrentUserRequest) (*pb.GetCurrentUserResponse, error) {
	// todo: get userId from context
	userId := int64(1)
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "invalid user!")
	}
	return &pb.GetCurrentUserResponse{
		Base: &pb.BaseResponse{Code: 200},
		User: &pb.User{
			Id:       user.Id,
			Username: user.Username,
			Nickname: user.Nickname,
			Phone:    user.Phone,
			Email:    user.Email,
		},
	}, nil
}
