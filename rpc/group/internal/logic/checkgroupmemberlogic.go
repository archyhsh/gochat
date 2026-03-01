package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/zeromicro/go-zero/core/logx"
)

type CheckGroupMemberLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckGroupMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckGroupMemberLogic {
	return &CheckGroupMemberLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckGroupMemberLogic) CheckGroupMember(in *pb.CheckGroupMemberRequest) (*pb.CheckGroupMemberResponse, error) {
	members, err := l.svcCtx.GroupMemberModel.FindOneByGroupIdUserId(l.ctx, in.GroupId, in.UserId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.CheckGroupMemberResponse{IsMember: false, Role: 0}, nil
		}
		l.Logger.Error("Failed to get group members", "groupID", in.GroupId, "userID", in.UserId, "error", err)
		return nil, err
	}
	if members == nil {
		return &pb.CheckGroupMemberResponse{IsMember: false, Role: 0}, nil
	}
	return &pb.CheckGroupMemberResponse{IsMember: true, Role: int32(members.Role)}, nil
}
