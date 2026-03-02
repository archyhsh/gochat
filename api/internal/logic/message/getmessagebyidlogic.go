// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package message

import (
	"context"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessageByIdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMessageByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageByIdLogic {
	return &GetMessageByIdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMessageByIdLogic) GetMessageById(req *types.GetMessageByIdRequest) (resp *types.Message, err error) {
	rpcResp, err := l.svcCtx.MessageRpc.GetMessageByID(l.ctx, &pb.GetMessageByIDRequest{
		MsgId: req.MsgId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call MessageRpc func GetMessageById: "+err.Error())
	}
	return &types.Message{
		MsgId:          rpcResp.Message.MsgId,
		ConversationId: rpcResp.Message.ConversationId,
		SenderId:       rpcResp.Message.SenderId,
		Content:        rpcResp.Message.Content,
		MsgType:        int(rpcResp.Message.MsgType),
		Timestamp:      rpcResp.Message.Timestamp,
	}, nil
}
