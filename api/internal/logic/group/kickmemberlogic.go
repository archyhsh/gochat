// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package group

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

type KickMemberLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewKickMemberLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KickMemberLogic {
	return &KickMemberLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KickMemberLogic) KickMember(req *types.KickMemberRequest) (resp *types.CommonResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	_, err = l.svcCtx.GroupRpc.KickGroupMember(ctx, &pb.KickGroupMemberRequest{
		GroupId:  req.GroupId,
		MemberId: req.MemberId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call groupRPC func KickGroupMember"+err.Error())
	}

	return &types.CommonResponse{
		Message: "Member kicked successfully",
	}, nil
}
