package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"github.com/archyhsh/gochat/rpc/user/model"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, err := l.svcCtx.UserModel.FindOneByUsername(l.ctx, in.Username)
	if err != nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "system error")
	}
	if user != nil {
		return nil, status.Error(codes.AlreadyExists, "Username already exists")
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	userModel := &model.User{
		Username: in.Username,
		Nickname: in.Nickname,
		Password: string(hashedPassword),
		Status:   1,
	}
	res, err := l.svcCtx.UserModel.Insert(l.ctx, userModel)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to create user")
	}
	newID, _ := res.LastInsertId()
	return &pb.RegisterResponse{
		Base: &pb.BaseResponse{Code: 200},
		User: &pb.User{
			Id:       newID,
			Username: userModel.Username,
			Nickname: userModel.Nickname,
			Gender:   0,
		},
	}, nil
}
