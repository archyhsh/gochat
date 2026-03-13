package logic

import (
	"context"
	"time"

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

	// Increment group meta version for announcement/nick changes
	group, err := l.svcCtx.GroupModel.FindOne(l.ctx, in.GroupId)
	var metaVersion int64
	if err == nil && group != nil {
		metaVersion = time.Now().UnixNano()
		group.MetaVersion = metaVersion
		_ = l.svcCtx.GroupModel.Update(l.ctx, group)
	}

	return &pb.UpdateGroupNicknameResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
