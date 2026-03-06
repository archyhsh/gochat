package logic

import (
	"context"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "user_id not found in metadata")
	}
	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid user_id in metadata")
	}

	var currTime time.Time
	uc, err := l.svcCtx.UserConversationModel.FindOneByUserIdConversationId(l.ctx, userId, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			_, errGlob := l.svcCtx.ConversationModel.FindOneByConversationId(l.ctx, in.ConversationId)
			if errGlob != nil {
				if errGlob == model.ErrNotFound {
					return nil, status.Error(codes.NotFound, "Conversation not found")
				}
				return nil, status.Error(codes.Internal, "failed to query global conversation")
			}
			currTime = time.Now()
		} else {
			return nil, status.Error(codes.Internal, "failed to query user conversation: "+err.Error())
		}
	} else {
		currTime = uc.LastMsgTime
	}

	remainingLimit := in.Limit
	cursorSeq := int64(in.LastSequence)
	var allMessages []*pb.ChatMessage

	for i := 0; i < 12; i++ {
		tableName := "message_" + currTime.Format("200601")

		msgs, err := l.svcCtx.MessageTemplateModel.FindPageByTable(l.ctx, tableName, in.ConversationId, cursorSeq, remainingLimit)
		if err != nil {
			currTime = currTime.AddDate(0, -1, 0)
			continue
		}

		for _, m := range msgs {
			allMessages = append(allMessages, &pb.ChatMessage{
				MsgId:          m.MsgId,
				ConversationId: m.ConversationId,
				SenderId:       m.SenderId,
				ReceiverId:     m.ReceiverId,
				GroupId:        m.GroupId,
				MsgType:        int32(m.MsgType),
				Content:        m.Content,
				Status:         int32(m.Status),
				Timestamp:      m.CreatedAt.UnixMilli(),
				Sequence:       m.SequenceId,
			})
		}

		remainingLimit -= int32(len(msgs))
		if remainingLimit <= 0 {
			break
		}

		cursorSeq = 0
		currTime = currTime.AddDate(0, -1, 0)
	}

	return &pb.GetMessagesResponse{
		Base:     &pb.BaseResponse{Code: 200, Message: "Success"},
		Messages: allMessages,
	}, nil
}
