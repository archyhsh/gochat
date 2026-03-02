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

type UpdateRemarkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRemarkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRemarkLogic {
	return &UpdateRemarkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRemarkLogic) UpdateRemark(req *types.UpdateRemarkRequest) (resp *types.CommonResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	_, err = l.svcCtx.RelationRpc.UpdateRemark(ctx, &pb.UpdateRemarkRequest{
		FriendId: req.FriendId,
		Remark:   req.Remark,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call relationRPC func UpdateRemark"+err.Error())
	}
	return &types.CommonResponse{
		Message: "update your remark for the friend successfully",
	}, nil
}
