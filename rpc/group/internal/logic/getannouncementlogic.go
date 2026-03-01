package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAnnouncementLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAnnouncementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAnnouncementLogic {
	return &GetAnnouncementLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAnnouncementLogic) GetAnnouncement(in *pb.GetAnnouncementRequest) (*pb.GetAnnouncementResponse, error) {
	group, err := l.svcCtx.GroupModel.FindValidGroupsByGroupId(l.ctx, in.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.GetAnnouncementResponse{
				Base: &pb.BaseResponse{Code: 404, Message: "Group not found"},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	return &pb.GetAnnouncementResponse{
		Base:         &pb.BaseResponse{Code: 200, Message: "Success"},
		Announcement: group.Announcement,
	}, nil
}
