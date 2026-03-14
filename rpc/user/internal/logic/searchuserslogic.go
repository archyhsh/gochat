package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchUsersLogic) SearchUsers(in *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	if len(in.Keyword) == 0 {
		return &pb.SearchUsersResponse{
			Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		}, nil
	}

	users, err := l.svcCtx.UserModel.SearchUsersByName(l.ctx, in.Keyword)
	if err != nil {
		l.Errorf("SearchUsers failed: keyword=%s, error=%v", in.Keyword, err)
		return nil, status.Error(codes.Internal, "failed to search users")
	}

	var userSummaries []*pb.User
	for _, u := range users {
		userSummaries = append(userSummaries, &pb.User{
			Id:       u.Id,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Phone:    u.Phone,
			Email:    u.Email,
			Gender:   int32(u.Gender),
		})
	}

	return &pb.SearchUsersResponse{
		Base:  &pb.BaseResponse{Code: 200, Message: "Success"},
		Users: userSummaries,
	}, nil
}
