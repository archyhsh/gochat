package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetUsersByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUsersByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsersByIdsLogic {
	return &GetUsersByIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUsersByIdsLogic) GetUsersByIds(in *pb.GetUsersByIdsRequest) (*pb.GetUsersByIdsResponse, error) {
	users, err := l.svcCtx.UserModel.SearchUsersByIds(l.ctx, in.UserIds)
	if err != nil {
		return nil, status.Error(codes.Internal, "批量查询用户失败")
	}
	var pbUsers []*pb.User
	for _, u := range users {
		pbUsers = append(pbUsers, &pb.User{
			Id:       u.Id,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
		})
	}
	return &pb.GetUsersByIdsResponse{
		Base:  &pb.BaseResponse{Code: 200},
		Users: pbUsers,
	}, nil
}
