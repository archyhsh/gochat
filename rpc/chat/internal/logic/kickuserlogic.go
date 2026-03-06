package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/chat/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type KickUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewKickUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickUserLogic {
	return &KickUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// KickUser disconnects a locally connected user.
func (l *KickUserLogic) KickUser(in *pb.KickRequest) (*pb.KickResponse, error) {
	conns := l.svcCtx.Manager.GetUserConnections(in.UserId)
	if len(conns) == 0 {
		return &pb.KickResponse{
			Base: &pb.BaseResponse{
				Code:    404,
				Message: "User not connected to this instance",
			},
		}, nil
	}

	for _, conn := range conns {
		l.Infof("Kicking user %d from connection %s. Reason: %s", in.UserId, conn.GetID(), in.Reason)
		_ = conn.Close()
	}

	return &pb.KickResponse{
		Base: &pb.BaseResponse{
			Code:    200,
			Message: "User kicked successfully",
		},
	}, nil
}
