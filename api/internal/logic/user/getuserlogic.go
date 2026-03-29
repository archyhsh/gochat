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
	userId, _ := l.ctx.Value("user_id").(int64)

	rpcResp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &pb.GetUserRequest{
		UserId: req.Id,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call UserRpc func GetUser"+err.Error())
	}

	// Privacy Masking: Only owner can see full details
	username := rpcResp.User.Username
	phone := rpcResp.User.Phone
	email := rpcResp.User.Email

	if userId != req.Id {
		// Mask Username
		if len(username) > 4 {
			username = username[:2] + "****" + username[len(username)-2:]
		} else if len(username) > 0 {
			username = username[:1] + "****"
		}
		// Mask Phone
		if len(phone) >= 7 {
			phone = phone[:3] + "****" + phone[len(phone)-4:]
		} else {
			phone = "****"
		}
		// Mask Email
		if parts := strings.Split(email, "@"); len(parts) == 2 {
			if len(parts[0]) > 2 {
				email = parts[0][:2] + "****@" + parts[1]
			} else {
				email = "****@" + parts[1]
			}
		} else {
			email = "****"
		}
	}

	return &types.User{
		Id:       rpcResp.User.Id,
		Username: username,
		Nickname: rpcResp.User.Nickname,
		Avatar:   rpcResp.User.Avatar,
		Phone:    phone,
		Email:    email,
		Gender:   int(rpcResp.User.Gender),
	}, nil
}
