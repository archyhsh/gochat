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

type GetGroupInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetGroupInfoLogic) GetGroupInfo(in *pb.GetGroupInfoRequest) (*pb.GetGroupInfoResponse, error) {
	// get unblacklisted group info in group model
	group, err := l.svcCtx.GroupModel.FindValidGroupsByGroupId(l.ctx, in.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.GetGroupInfoResponse{
				Base: &pb.BaseResponse{Code: 404, Message: "Group not found"},
			}, nil
		}
		l.Errorf("GetGroupInfo failed: groupID=%d, error=%v", in.GroupId, err)
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	if group.Status == 0 {
		return &pb.GetGroupInfoResponse{
			Base: &pb.BaseResponse{Code: 400, Message: "Group is dismissed"},
		}, nil
	}

	return &pb.GetGroupInfoResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		Group: &pb.Group{
			Id:           group.Id,
			Name:         group.Name,
			Avatar:       group.Avatar,
			Description:  group.Description,
			Announcement: group.Announcement,
			OwnerId:      group.OwnerId,
			Status:       int32(group.Status),
			CreatedAt:    group.CreatedAt.Unix(),
			UpdatedAt:    group.UpdatedAt.Unix(),
		},
	}, nil
}
