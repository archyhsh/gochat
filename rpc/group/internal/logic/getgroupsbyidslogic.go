package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGroupsByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupsByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupsByIdsLogic {
	return &GetGroupsByIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupsByIdsLogic) GetGroupsByIds(in *pb.GetGroupsByIdsRequest) (*pb.GetGroupsByIdsResponse, error) {
	groups, err := l.svcCtx.GroupModel.FindByIds(l.ctx, in.GroupIds)
	if err != nil {
		l.Errorf("GetGroupsByIds failed: ids=%v, error=%v", in.GroupIds, err)
		return nil, err
	}

	var pbGroups []*pb.Group
	for _, g := range groups {
		pbGroups = append(pbGroups, &pb.Group{
			Id:           g.Id,
			Name:         g.Name,
			Avatar:       g.Avatar,
			Description:  g.Description,
			Announcement: g.Announcement,
			OwnerId:      g.OwnerId,
			Status:       int32(g.Status),
			CreatedAt:    g.CreatedAt.Unix(),
			UpdatedAt:    g.UpdatedAt.Unix(),
			MetaVersion:  g.MetaVersion,
		})
	}

	return &pb.GetGroupsByIdsResponse{
		Base:   &pb.BaseResponse{Code: 200, Message: "Success"},
		Groups: pbGroups,
	}, nil
}
