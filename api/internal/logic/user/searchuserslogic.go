// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"strings"

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
		// Mask sensitive data for public search
		maskedUsername := ""
		if len(u.Username) > 4 {
			maskedUsername = u.Username[:2] + "****" + u.Username[len(u.Username)-2:]
		} else if len(u.Username) > 0 {
			maskedUsername = u.Username[:1] + "****"
		}

		maskedPhone := ""
		if len(u.Phone) >= 7 {
			maskedPhone = u.Phone[:3] + "****" + u.Phone[len(u.Phone)-4:]
		}

		maskedEmail := ""
		if parts := strings.Split(u.Email, "@"); len(parts) == 2 {
			if len(parts[0]) > 2 {
				maskedEmail = parts[0][:2] + "****@" + parts[1]
			} else {
				maskedEmail = "****@" + parts[1]
			}
		}

		users = append(users, types.User{
			Id:       u.Id,
			Username: maskedUsername,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Phone:    maskedPhone,
			Email:    maskedEmail,
			Gender:   int(u.Gender),
		})
	}
	return &types.SearchResponse{
		Users: users,
	}, nil
}
