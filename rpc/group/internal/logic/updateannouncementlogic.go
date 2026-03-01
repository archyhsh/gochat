package logic

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAnnouncementLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAnnouncementLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAnnouncementLogic {
	return &UpdateAnnouncementLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateAnnouncementLogic) UpdateAnnouncement(in *pb.UpdateAnnouncementRequest) (*pb.UpdateAnnouncementResponse, error) {
	// TODO: Get userID from context
	userID := int64(1)
	member, err := l.svcCtx.GroupMemberModel.FindMemberByGroupIdAndUserId(l.ctx, in.GroupId, userID)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.UpdateAnnouncementResponse{
				Base: &pb.BaseResponse{Code: 403, Message: "Not a member of this group"},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to query member status")
	}

	if member.Role < 1 {
		return &pb.UpdateAnnouncementResponse{
			Base: &pb.BaseResponse{Code: 403, Message: "Permission denied"},
		}, nil
	}

	group, err := l.svcCtx.GroupModel.FindValidGroupsByGroupId(l.ctx, in.GroupId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	group.Announcement = in.Content
	err = l.svcCtx.GroupModel.Update(l.ctx, group)
	if err != nil {
		l.Errorf("UpdateAnnouncement failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to update announcement")
	}

	if l.svcCtx.Config.Producer != nil {
		event := map[string]interface{}{
			"type":      "group_event",
			"action":    "updateAnnouncement",
			"group_id":  in.GroupId,
			"timestamp": time.Now().Unix(),
		}
		data, _ := json.Marshal(event)
		key := strconv.FormatInt(in.GroupId, 10)
		err = l.svcCtx.Config.Producer.Send([]byte(key), data)
		if err != nil {
			l.Errorf("Failed to send Kafka event for UpdateAnnouncement: %v", err)
		}
	}

	return &pb.UpdateAnnouncementResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
