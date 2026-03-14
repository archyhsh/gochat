package group

import (
	"context"
	"errors"
	"strconv"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/group/groupservice"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
)

type HandleGroupRequestLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHandleGroupRequestLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleGroupRequestLogic {
	return &HandleGroupRequestLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HandleGroupRequestLogic) HandleGroupRequest(req *types.HandleGroupRequest) (resp *types.CommonResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("user not login")
	}
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)
	_, err = l.svcCtx.GroupRpc.HandleGroupRequest(ctx, &groupservice.HandleGroupRequestRequest{
		RequestId: req.RequestId,
		Accept:    req.Accept,
	})
	if err != nil {
		return nil, err
	}

	return &types.CommonResponse{
		Message: "Success",
	}, nil
}
