// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package relation

import (
	"context"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type HandleApplyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHandleApplyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HandleApplyLogic {
	return &HandleApplyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HandleApplyLogic) HandleApply(req *types.HandleApplyRequest) (resp *types.CommonResponse, err error) {
	_, err = l.svcCtx.RelationRpc.HandleApply(l.ctx, &pb.HandleApplyRequest{
		ApplyId: req.ApplyId,
		Accept:  req.Accept,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call RelationRpc func HandleApply")
	}
	var message string
	if req.Accept {
		message = "You have accepted the friend request"
	} else {
		message = "You have rejected the friend request"
	}
	return &types.CommonResponse{
		Message: message,
	}, nil
}
