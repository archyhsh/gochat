// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

import (
	"context"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAnnouncementLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAnnouncementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAnnouncementLogic {
	return &GetAnnouncementLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAnnouncementLogic) GetAnnouncement(req *types.GetAnnouncementRequest) (resp *types.CommonResponse, err error) {
	rpcResp, err := l.svcCtx.GroupRpc.GetAnnouncement(l.ctx, &pb.GetAnnouncementRequest{
		GroupId: req.Id,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func GetAnnouncement"+err.Error())
	}
	return &types.CommonResponse{
		Message: rpcResp.Announcement,
	}, nil
}
