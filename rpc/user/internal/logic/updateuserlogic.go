package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/user/internal/svc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}
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
	userInfo.InfoVersion = time.Now().UnixNano()
	err = l.svcCtx.UserModel.Update(l.ctx, userInfo)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update user info: "+err.Error())
	}

	// Send global nickname update event
	if l.svcCtx.Producer != nil {
		event := map[string]interface{}{
			"type":      "nickname_update",
			"user_id":   userId,
			"nickname":  in.Nickname,
			"version":   userInfo.InfoVersion,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		_ = l.svcCtx.Producer.Send([]byte(strconv.FormatInt(userId, 10)), data)
	}

	return &pb.UpdateUserResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		User: &pb.User{
			Id:          userId,
			Nickname:    in.Nickname,
			Avatar:      in.Avatar,
			Phone:       in.Phone,
			Email:       in.Email,
			Gender:      in.Gender,
			InfoVersion: userInfo.InfoVersion,
		},
	}, nil
}
