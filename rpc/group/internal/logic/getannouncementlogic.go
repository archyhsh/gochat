package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	// Authentication: Extract user_id from metadata
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found")
	}
	userId, _ := strconv.ParseInt(userIdStrs[0], 10, 64)

	// get announcement of a valid group in group model
	group, err := l.svcCtx.GroupModel.FindValidGroupsByGroupId(l.ctx, in.GroupId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.GetAnnouncementResponse{
				Base: &pb.BaseResponse{Code: 404, Message: "Group not found"},
			}, nil
		}
		return nil, status.Error(codes.Internal, "failed to query group")
	}

	// Authorization: Check if requester is a member
	caller, _ := l.svcCtx.GroupMemberModel.FindOneByGroupIdUserId(l.ctx, in.GroupId, userId)
	if caller == nil {
		return nil, status.Error(codes.PermissionDenied, "access denied: not a group member")
	}

	return &pb.GetAnnouncementResponse{
		Base:         &pb.BaseResponse{Code: 200, Message: "Success"},
		Announcement: group.Announcement,
	}, nil
}
