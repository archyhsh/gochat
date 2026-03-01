package logic

import (
	"context"
	"time"

	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/internal/svc"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"google.golang.org/grpc/codes"
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
	// todo: get userId from context
	userId := int64(1)
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
