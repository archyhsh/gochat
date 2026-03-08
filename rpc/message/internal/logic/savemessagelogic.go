package logic

import (
	"context"
	"time"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type SaveMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveMessageLogic {
	return &SaveMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SaveMessageLogic) SaveMessage(in *pb.SaveMessageRequest) (*pb.SaveMessageResponse, error) {
	tableName := "message_" + time.UnixMilli(in.Message.Timestamp).Format("200601")
	err := l.svcCtx.MessageTemplateModel.CheckTableExist(l.ctx, tableName)
	if err != nil {
		l.Errorf("Failed to ensure table %s exists: %v", tableName, err)
		return nil, status.Error(codes.Internal, "Internal database error")
	}

	var newSeq int64
	err = l.svcCtx.SqlConn.TransactCtx(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		msgModel := &model.MessageTemplate{
			MsgId:     in.Message.MsgId,
			Content:   in.Message.Content,
			MsgType:   int64(in.Message.MsgType),
			SenderId:  in.Message.SenderId,
			CreatedAt: time.UnixMilli(in.Message.Timestamp),
		}

		convType := int32(1)
		targetId := in.Message.ReceiverId
		if in.Message.GroupId > 0 {
			convType = 2
			targetId = in.Message.GroupId
		}

		seq, err := l.svcCtx.ConversationModel.UpdateSeq(ctx, s, in.Message.ConversationId, convType, targetId, msgModel)
		if err != nil {
			return status.Error(codes.Internal, "fail to update conversation table: "+err.Error())
		}
		newSeq = seq
		msgModel.ConversationId = in.Message.ConversationId
		msgModel.ReceiverId = in.Message.ReceiverId
		msgModel.GroupId = in.Message.GroupId
		msgModel.SequenceId = newSeq
		msgModel.Status = 0

		err = l.svcCtx.MessageTemplateModel.InsertToTable(ctx, s, tableName, msgModel)
		if err != nil {
			return status.Error(codes.Internal, "fail to insert msg: "+err.Error())
		}
		updatedUsers := make(map[int64]bool)
		if in.Message.SenderId > 0 {
			peerId := in.Message.ReceiverId
			if in.Message.GroupId > 0 {
				peerId = in.Message.GroupId
			}
			err = l.svcCtx.UserConversationModel.UpdateNewPrivateMsg(ctx, s, in.Message.SenderId, peerId, in.Message.ConversationId, msgModel, false)
			if err != nil {
				return status.Error(codes.Internal, "fail to init sender bookmark: "+err.Error())
			}
			updatedUsers[in.Message.SenderId] = true
		}
		targets := in.Message.TargetIds
		if in.Message.GroupId == 0 && len(targets) == 0 && in.Message.ReceiverId > 0 {
			targets = []int64{in.Message.ReceiverId}
		}

		for _, tid := range targets {
			if tid <= 0 || updatedUsers[tid] {
				continue
			}
			incUnread := tid != in.Message.SenderId
			peerId := in.Message.SenderId
			if in.Message.GroupId > 0 {
				peerId = in.Message.GroupId
			} else if peerId == 0 {
				peerId = in.Message.ReceiverId
			}

			err = l.svcCtx.UserConversationModel.UpdateNewPrivateMsg(ctx, s, tid, peerId, in.Message.ConversationId, msgModel, incUnread)
			if err != nil {
				return status.Error(codes.Internal, "fail to init target bookmark: "+err.Error())
			}
			updatedUsers[tid] = true
		}

		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to persist message: "+err.Error())
	}

	return &pb.SaveMessageResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
