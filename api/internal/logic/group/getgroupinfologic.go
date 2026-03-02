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

type GetGroupInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetGroupInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupInfoLogic {
	return &GetGroupInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupInfoLogic) GetGroupInfo(req *types.GetGroupInfoRequest) (resp *types.GroupInfo, err error) {
	rpcResp, err := l.svcCtx.GroupRpc.GetGroupInfo(l.ctx, &pb.GetGroupInfoRequest{
		GroupId: req.GroupId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call GroupRpc func GetGroupInfo"+err.Error())
	}
	return &types.GroupInfo{
		GroupId:      rpcResp.Group.Id,
		Name:         rpcResp.Group.Name,
		Avatar:       rpcResp.Group.Avatar,
		Description:  rpcResp.Group.Description,
		Announcement: rpcResp.Group.Announcement,
		OwnerId:      rpcResp.Group.OwnerId,
	}, nil
}
