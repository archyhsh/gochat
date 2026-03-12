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

type UpdateGroupNicknameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateGroupNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGroupNicknameLogic {
	return &UpdateGroupNicknameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateGroupNicknameLogic) UpdateGroupNickname(req *types.UpdateGroupNicknameRequest) (resp *types.CommonResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}

	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	_, err = l.svcCtx.GroupRpc.UpdateGroupNickname(ctx, &pb.UpdateGroupNicknameRequest{
		GroupId:  req.GroupId,
		UserId:   userId,
		Nickname: req.Nickname,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to update group nickname: "+err.Error())
	}

	return &types.CommonResponse{
		Message: "Success",
	}, nil
}
