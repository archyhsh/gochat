package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/group/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateGroupNicknameLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateGroupNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGroupNicknameLogic {
	return &UpdateGroupNicknameLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateGroupNicknameLogic) UpdateGroupNickname(in *pb.UpdateGroupNicknameRequest) (*pb.UpdateGroupNicknameResponse, error) {
	err := l.svcCtx.GroupMemberModel.UpdateNickname(l.ctx, in.GroupId, in.UserId, in.Nickname)
	if err != nil {
		l.Errorf("UpdateGroupNickname failed: %v", err)
		return nil, err
	}

	// We no longer increment Group Meta Version for individual nickname changes
	// to avoid massive Kafka events and cache invalidations in large groups.

	return &pb.UpdateGroupNicknameResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
