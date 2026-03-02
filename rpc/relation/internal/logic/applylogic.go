package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyLogic {
	return &ApplyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ApplyLogic) Apply(in *pb.ApplyRequest) (*pb.ApplyResponse, error) {
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

	apply, err := l.svcCtx.FriendApplyModel.FindPendingApplyByFromAndTo(l.ctx, userId, in.ToUserId)
	if err != nil && err != model.ErrNotFound {
		return nil, status.Error(codes.Internal, "failed to check existing apply")
	}
	if apply != nil {
		return nil, status.Error(codes.AlreadyExists, "existing pending request")
	}
	_, err = l.svcCtx.FriendApplyModel.Insert(l.ctx, &model.FriendApply{
		FromUserId: userId,
		ToUserId:   in.ToUserId,
		Message:    in.Message,
		Status:     0,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to insert apply")
	}
	return &pb.ApplyResponse{Base: &pb.BaseResponse{Code: 200, Message: "Success"}}, nil
}
