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

type GetGroupMembersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupMembersLogic {
	return &GetGroupMembersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupMembersLogic) GetGroupMembers(req *types.GetGroupMembersRequest) (resp *types.GroupMembersResponse, err error) {
	rpcResp, err := l.svcCtx.GroupRpc.GetGroupMembers(l.ctx, &pb.GetGroupMembersRequest{
		GroupId: req.GroupId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func GetGroupMembers"+err.Error())
	}
	var members []types.GroupMember
	for _, member := range rpcResp.Members {
		members = append(members, types.GroupMember{
			UserId:   member.UserId,
			Nickname: member.Nickname,
			Avatar:   member.Avatar,
			Role:     int(member.Role),
		})
	}
	return &types.GroupMembersResponse{
		Members: members,
	}, nil
}
