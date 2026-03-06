package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/chat/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PushToUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPushToUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PushToUserLogic {
	return &PushToUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// PushToUser pushes a message to a locally connected user.
func (l *PushToUserLogic) PushToUser(in *pb.PushRequest) (*pb.PushResponse, error) {
	err := l.svcCtx.Manager.SendToUser(in.UserId, in.Message)
	if err != nil {
		l.Debugf("Local push failed for user %d: %v", in.UserId, err)
		return &pb.PushResponse{
			Base: &pb.BaseResponse{
				Code:    404,
				Message: "User not connected to this instance",
			},
		}, nil
	}

	l.Infof("Successfully pushed message to user %d locally", in.UserId)
	return &pb.PushResponse{
		Base: &pb.BaseResponse{
			Code:    200,
			Message: "Pushed successfully",
		},
	}, nil
}
