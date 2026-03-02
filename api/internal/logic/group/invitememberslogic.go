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

type InviteMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInviteMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteMembersLogic {
	return &InviteMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InviteMembersLogic) InviteMembers(req *types.InviteRequest) (resp *types.CommonResponse, err error) {
	_, err = l.svcCtx.GroupRpc.InviteMembers(l.ctx, &pb.InviteMembersRequest{
		GroupId:   req.Id,
		MemberIds: req.MemberIds,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func InviteMembers"+err.Error())
	}
	return &types.CommonResponse{
		Message: "Invite members successfully",
	}, nil
}
