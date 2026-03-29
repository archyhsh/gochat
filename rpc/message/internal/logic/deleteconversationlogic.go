package logic

import (
	"context"
	"strconv"

	"github.com/archyhsh/gochat/rpc/message/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConversationLogic {
	return &DeleteConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteConversation implements the logical deletion (hiding) of a conversation for a specific user.
// Use Cases:
// 1. User swipes left to delete a chat in the conversation list (UI hiding).
// 2. Clear chat history preview but preserve the actual message data in the global pool.
// 3. Multi-device synchronization: Hiding the chat on one device will sync the 'is_deleted' state to others via versioning.
func (l *DeleteConversationLogic) DeleteConversation(in *pb.DeleteConversationRequest) (*pb.DeleteConversationResponse, error) {
	// Extract identity from gRPC metadata to ensure security (prevents user ID spoofing)
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

	// Hide the conversation by setting is_deleted = 1 and updating the version
	// Note: Redis cache for this user-conversation pair will be invalidated automatically by the Model method.
	err = l.svcCtx.UserConversationModel.Hide(l.ctx, userId, in.ConversationId)
	if err != nil {
		l.Errorf("DeleteConversation (Hide) failed: %v", err)
		return nil, err
	}

	return &pb.DeleteConversationResponse{
		Base: &pb.BaseResponse{Code: 200, Message: "Success"},
	}, nil
}
