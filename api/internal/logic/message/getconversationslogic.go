// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

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

func (l *GetConversationsLogic) GetConversations() (resp *types.ConversationsResponse, err error) {
	userId, ok := l.ctx.Value("user_id").(int64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not login")
	}
	// send userId to backend RPC via gRPC Metadata
	md := metadata.Pairs("user_id", strconv.FormatInt(userId, 10))
	ctx := metadata.NewOutgoingContext(l.ctx, md)

	rpcResp, err := l.svcCtx.MessageRpc.GetConversations(ctx, &pb.GetConversationsRequest{
		Limit: 50,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "fail to call messageRPC func GetConversations: "+err.Error())
	}

	groupResp, err := l.svcCtx.GroupRpc.GetGroupList(ctx, &pb.GetGroupListRequest{})
	if err != nil {
		l.Errorf("failed to fetch groups for user %d: %v", userId, err)
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
	if groupResp != nil {
		for _, g := range groupResp.Groups {
			convId := fmt.Sprintf("group_%d", g.GroupId)
			if !existingConvIds[convId] {
				conversations = append(conversations, types.Conversation{
					ConversationId:  convId,
					PeerId:          g.GroupId,
					UnreadCount:     0,
					LastMessage:     "Joined group",
					LastMessageTime: time.Now().UnixMilli(),
				})
			}
		}
	}

	return &types.ConversationsResponse{
		Conversations: conversations,
	}, nil
}
