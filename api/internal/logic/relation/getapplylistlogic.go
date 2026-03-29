// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package relation

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetApplyListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetApplyListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApplyListLogic {
	return &GetApplyListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetApplyListLogic) GetApplyList() (resp *types.ApplyListResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.RelationRpc.GetApplyList(ctx, &pb.GetApplyListRequest{})
	if err != nil {
		return nil, err
	}
	var applies []types.ApplyInfo
	for _, apply := range rpcResp.Applies {
		applies = append(applies, types.ApplyInfo{
			Id:         apply.Id,
			FromUserId: apply.FromUserId,
			Message:    apply.Message,
			Status:     int(apply.Status),
		})
	}
	return &types.ApplyListResponse{
		Applies: applies,
	}, nil
}
