package message

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/api/internal/svc"
	"github.com/archyhsh/gochat/api/internal/types"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(req *types.GetConversationsRequest) (resp *types.ConversationsResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.MessageRpc.GetConversations(ctx, &pb.GetConversationsRequest{
		Limit:   50,
		Keyword: req.Keyword,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call messageRPC: "+err.Error())
	}

	var conversations []types.Conversation
	existingConvIds := make(map[string]bool)
	for _, c := range rpcResp.Conversations {
		conversations = append(conversations, types.Conversation{
			ConversationId:  c.ConversationId,
			PeerId:          c.PeerId,
			UnreadCount:     int(c.UnreadCount),
			LastMessage:     c.LastMessage,
			LastMessageTime: c.LastMessageTime,
		})
		existingConvIds[c.ConversationId] = true
	}

	// For empty keyword, we also supplement with group list for newly joined groups
	if req.Keyword == "" {
		groupResp, err := l.svcCtx.GroupRpc.GetGroupList(ctx, &pb.GetGroupListRequest{})
		if err == nil && groupResp != nil {
			for _, g := range groupResp.Groups {
				convId := fmt.Sprintf("group_%d", g.GroupId)
				if !existingConvIds[convId] {
					conversations = append(conversations, types.Conversation{
						ConversationId:  convId,
						PeerId:          g.GroupId,
						UnreadCount:     0,
						LastMessage:     "No messages yet",
						LastMessageTime: time.Now().UnixMilli(),
					})
				}
			}
		}
	}

	return &types.ConversationsResponse{
		Conversations: conversations,
	}, nil
}
