package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMessagesLogic {
	return &GetMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMessagesLogic) GetMessages(in *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	//todo: get userId from context
	userId := int64(1)
	uc, err := l.svcCtx.UserConversationModel.FindUserConversationsByUserIdAndConversationId(userId, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "cannot find the conversation, please check the status")
		}
		return nil, status.Error(codes.Internal, "failed to find the conversation for the user")
	}
	currTime := uc.LastMsgTime
	remainingLimit := in.Limit
	currentOffset := in.Offset
	var messages []*pb.ChatMessage
	for _ = range 50 {
		msgTable := "message_" + currTime.Format("200601")
		count, err := l.svcCtx.MessageTemplateModel.CountByTable(l.ctx, msgTable, in.ConversationId)
		if err != nil {
			// if the table does not exist, we just skip it and continue to the previous month
			currTime = currTime.AddDate(0, -1, 0)
			continue
		}
		if int64(currentOffset) >= count {
			currentOffset = currentOffset - int32(count)
		} else {
			take := remainingLimit
			msgPage, err := l.svcCtx.MessageTemplateModel.FindPageByTable(l.ctx, msgTable, in.ConversationId, take, currentOffset)
			if err != nil {
				return nil, status.Error(codes.Internal, "failed to find messages")
			}
			for _, msg := range msgPage {
				messages = append(messages, &pb.ChatMessage{
					MsgId:          msg.MsgId,
					ConversationId: msg.ConversationId,
					SenderId:       msg.SenderId,
					ReceiverId:     msg.ReceiverId,
					GroupId:        msg.GroupId,
					MsgType:        int32(msg.MsgType),
					Content:        msg.Content.String,
					Status:         int32(msg.Status),
					Timestamp:      msg.CreatedAt.Unix(),
				})
			}
			remainingLimit = remainingLimit - int32(len(msgPage))
			currentOffset = 0
		}
		if remainingLimit <= 0 {
			break
		}
		currTime = currTime.AddDate(0, -1, 0)
	}

	return &pb.GetMessagesResponse{
		Base:     &pb.BaseResponse{Code: 200, Message: "Success"},
		Messages: messages,
	}, nil
}
