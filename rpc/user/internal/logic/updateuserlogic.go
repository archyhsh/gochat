package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateUserLogic) UpdateUser(in *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	// TODO: get userId from context
	userId := int64(1)
	userInfo, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to find user info: "+err.Error())
	}
	if userInfo == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	userInfo.Id = userId
	userInfo.Nickname = in.Nickname
	userInfo.Avatar = in.Avatar
	userInfo.Phone = in.Phone
	userInfo.Email = in.Email
	userInfo.Gender = int64(in.Gender)
	err = l.svcCtx.UserModel.Update(l.ctx, userInfo)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update user info: "+err.Error())
	}

	return &pb.UpdateUserResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		User: &pb.User{Id: userId, Nickname: in.Nickname, Avatar: in.Avatar, Phone: in.Phone, Email: in.Email, Gender: in.Gender},
	}, nil
}
