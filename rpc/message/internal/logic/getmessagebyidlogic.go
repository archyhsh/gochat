package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessageByIDLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMessageByIDLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessageByIDLogic {
	return &GetMessageByIDLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMessageByIDLogic) GetMessageByID(in *pb.GetMessageByIDRequest) (*pb.GetMessageByIDResponse, error) {
	msgIdInt, err := strconv.ParseInt(in.MsgId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id format")
	}

	milli, _, _ := snowflake.ParseID(msgIdInt)
	msgTime := time.UnixMilli(milli)
	targetTable := "message_" + msgTime.Format("200601")

	msg, err := l.svcCtx.MessageTemplateModel.FindOneByTableAndMessageId(l.ctx, targetTable, in.MsgId)
	if err != nil {
		if err == model.ErrNotFound {
			return &pb.GetMessageByIDResponse{
				Base: &pb.BaseResponse{Code: 404, Message: "Message not found"},
			}, nil
		}
		return nil, status.Error(codes.Internal, "Failed to query message"+err.Error())
	}

	return &pb.GetMessageByIDResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
		Message: &pb.ChatMessage{
			MsgId:          msg.MsgId,
			ConversationId: msg.ConversationId,
			SenderId:       msg.SenderId,
			ReceiverId:     msg.ReceiverId,
			GroupId:        msg.GroupId,
			MsgType:        int32(msg.MsgType),
			Content:        msg.Content,
			Status:         int32(msg.Status),
			Timestamp:      msg.CreatedAt.UnixMilli(),
			Sequence:       msg.SequenceId,
		},
	}, nil
}
