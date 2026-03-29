package logic

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	// 1. Authentication: Extract user_id from metadata
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

	// 2. Authorization: Check if user is participant
	uc, _ := l.svcCtx.UserConversationModel.FindOneByUserIdConversationId(l.ctx, userId, msg.ConversationId)
	if uc == nil {
		// Fallback check for newly joined or deleted list
		if strings.HasPrefix(msg.ConversationId, "group_") {
			check, err := l.svcCtx.GroupRpc.CheckGroupMember(l.ctx, &pb.CheckGroupMemberRequest{
				GroupId: msg.GroupId,
				UserId:  userId,
			})
			if err != nil || !check.IsMember {
				return nil, status.Error(codes.PermissionDenied, "access denied: not a member of this conversation")
			}
		} else if strings.HasPrefix(msg.ConversationId, "conv_") {
			parts := strings.Split(msg.ConversationId, "_")
			if len(parts) == 3 {
				id1, _ := strconv.ParseInt(parts[1], 10, 64)
				id2, _ := strconv.ParseInt(parts[2], 10, 64)
				if userId != id1 && userId != id2 {
					return nil, status.Error(codes.PermissionDenied, "access denied: not a participant")
				}
			} else {
				return nil, status.Error(codes.PermissionDenied, "access denied")
			}
		} else {
			return nil, status.Error(codes.PermissionDenied, "access denied")
		}
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
