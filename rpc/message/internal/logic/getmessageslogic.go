package logic

import (
	"context"
	"strconv"
	"strings"
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

	// 2. Authorization: Verify if user has access to this conversation
	var currTime time.Time
	uc, err := l.svcCtx.UserConversationModel.FindOneByUserIdConversationId(l.ctx, userId, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			// If no local conversation record, perform a strict membership check
			if strings.HasPrefix(in.ConversationId, "group_") {
				// Group Chat: Check real-time membership via Group RPC
				groupIdStr := strings.TrimPrefix(in.ConversationId, "group_")
				groupId, _ := strconv.ParseInt(groupIdStr, 10, 64)
				check, err := l.svcCtx.GroupRpc.CheckGroupMember(l.ctx, &pb.CheckGroupMemberRequest{
					GroupId: groupId,
					UserId:  userId,
				})
				if err != nil || !check.IsMember {
					return nil, status.Error(codes.PermissionDenied, "access denied: not a group member")
				}
			} else if strings.HasPrefix(in.ConversationId, "conv_") {
				// Private Chat: Check if current user is one of the participants in ID-based conv_A_B
				parts := strings.Split(in.ConversationId, "_")
				if len(parts) != 3 {
					return nil, status.Error(codes.InvalidArgument, "invalid conversation id format")
				}
				id1, _ := strconv.ParseInt(parts[1], 10, 64)
				id2, _ := strconv.ParseInt(parts[2], 10, 64)
				if userId != id1 && userId != id2 {
					return nil, status.Error(codes.PermissionDenied, "access denied: not a participant of this private chat")
				}
			} else {
				return nil, status.Error(codes.InvalidArgument, "unknown conversation type")
			}
			// If authorized but record missing (e.g. newly joined or deleted list), start from now
			currTime = time.Now()
		} else {
			return nil, status.Error(codes.Internal, "failed to query user conversation: "+err.Error())
		}
	} else {
		currTime = uc.LastMsgTime
	}

	// 3. Fetch Messages from partitioned tables
	remainingLimit := in.Limit
	isSync := false
	if remainingLimit < 0 {
		isSync = true
		remainingLimit = -remainingLimit
	}

	cursorSeq := int64(in.LastSequence)
	var allMessages []*pb.ChatMessage

	for i := 0; i < 12; i++ {
		tableName := "message_" + currTime.Format("200601")

		var msgs []*model.MessageTemplate
		var err error
		if isSync {
			msgs, err = l.svcCtx.MessageTemplateModel.FindNewerBySeq(l.ctx, tableName, in.ConversationId, cursorSeq, remainingLimit)
		} else {
			msgs, err = l.svcCtx.MessageTemplateModel.FindPageByTable(l.ctx, tableName, in.ConversationId, cursorSeq, remainingLimit)
		}

		if err != nil {
			if isSync {
				currTime = currTime.AddDate(0, 1, 0)
			} else {
				currTime = currTime.AddDate(0, -1, 0)
			}
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
		if isSync {
			currTime = currTime.AddDate(0, 1, 0)
			if currTime.After(time.Now().AddDate(0, 1, 0)) {
				break
			}
		} else {
			currTime = currTime.AddDate(0, -1, 0)
		}
	}

	return &pb.GetMessagesResponse{
		Base:     &pb.BaseResponse{Code: 200, Message: "Success"},
		Messages: allMessages,
	}, nil
}
