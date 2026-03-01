package logic

import (
	"context"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetConversationsLogic) GetConversations(in *pb.GetConversationsRequest) (*pb.GetConversationsResponse, error) {
	// get conversations for the user and return them
	// todo: get userId from context
	userId := int64(1)
	userConversations, err := l.svcCtx.UserConversationModel.GetUserConversationsByUserId(userId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user conversations")
	}
	var conversations []*pb.ConversationInfo
	for _, uc := range userConversations {
		// skip deleted conversations
		if uc.IsDeleted == 1 {
			continue
		}
		conversations = append(conversations, &pb.ConversationInfo{
			ConversationId:  uc.ConversationId,
			PeerId:          uc.PeerId,
			UnreadCount:     int32(uc.UnreadCount),
			LastMsgId:       uc.LastMsgId,
			LastMessage:     uc.LastMsgContent,
			LastMsgType:     int32(uc.LastMsgType),
			LastSenderId:    uc.LastSenderId,
			LastMessageTime: uc.LastMsgTime.Unix(),
			IsTop:           (uc.IsTop == 1),
			IsMuted:         (uc.IsMuted == 1),
		})
	}
	return &pb.GetConversationsResponse{
		Base:          &pb.BaseResponse{Code: 200, Message: "success"},
		Conversations: conversations,
	}, nil
}
