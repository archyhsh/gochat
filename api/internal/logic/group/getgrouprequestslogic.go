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

type GetGroupRequestsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetGroupRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGroupRequestsLogic {
	return &GetGroupRequestsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGroupRequestsLogic) GetGroupRequests() (resp *types.GetGroupRequestsResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, errors.New("user not login")
	}
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.GroupRpc.GetGroupRequests(ctx, &groupservice.GetGroupRequestsRequest{
		GroupId: 0,
	})
	if err != nil {
		return nil, err
	}

	var requests []types.GroupRequest
	for _, r := range rpcResp.Requests {
		requests = append(requests, types.GroupRequest{
			Id:        r.Id,
			GroupId:   r.GroupId,
			UserId:    r.UserId,
			Message:   r.Message,
			Status:    int(r.Status),
			CreatedAt: r.CreatedAt,
			Nickname:  r.Nickname,
			Avatar:    r.Avatar,
		})
	}

	return &types.GetGroupRequestsResponse{
		Requests: requests,
	}, nil
}
