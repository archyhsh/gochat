package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, in.Username)
	if err != nil {
		return nil, status.Error(codes.NotFound, "Cannot find username")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid password")
	}
	Token, err := l.svcCtx.JwtManager.GenerateToken(user.Id, user.Username)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to generate token")
	}
	return &pb.LoginResponse{
		Base:  &pb.BaseResponse{Code: 200, Message: "Login successful"},
		Token: Token,
		User: &pb.User{
			Id:       user.Id,
			Nickname: user.Nickname,
		},
	}, nil
}
