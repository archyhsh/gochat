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

type SearchUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchUsersLogic) SearchUsers(req *types.SearchRequest) (resp *types.SearchResponse, err error) {
	rpcResp, err := l.svcCtx.UserRpc.SearchUsers(l.ctx, &pb.SearchUsersRequest{
		Keyword: req.Keyword,
		Limit:   int32(req.Limit),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call UserRpc func SearchUsers"+err.Error())
	}
	var users []types.User
	for _, u := range rpcResp.Users {
		users = append(users, types.User{
			Id:       u.Id,
			Username: u.Username,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Phone:    u.Phone,
			Email:    u.Email,
			Gender:   int(u.Gender),
		})
	}
	return &types.SearchResponse{
		Users: users,
	}, nil
}
