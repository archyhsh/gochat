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

type SearchGroupsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchGroupsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchGroupsLogic {
	return &SearchGroupsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchGroupsLogic) SearchGroups(req *types.SearchGroupsRequest) (resp *types.GroupListResponse, err error) {
	rpcResp, err := l.svcCtx.GroupRpc.SearchGroups(l.ctx, &pb.SearchGroupsRequest{
		Keyword: req.Keyword,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func SearchGroups: "+err.Error())
	}
	var groups []types.GroupInfo
	for _, g := range rpcResp.Groups {
		groups = append(groups, types.GroupInfo{
			GroupId: g.GroupId,
			Name:    g.Name,
			Avatar:  g.Avatar,
			OwnerId: g.OwnerId,
		})
	}
	return &types.GroupListResponse{
		Groups: groups,
	}, nil
}
