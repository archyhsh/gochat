package logic

import (
	"context"
	"io"
	"strconv"

	"github.com/archyhsh/gochat/internal/gateway/connection"
	"github.com/archyhsh/gochat/rpc/chat/internal/svc"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/zeromicro/go-zero/core/logx"
)

type StreamMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStreamMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamMessagesLogic {
	return &StreamMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// StreamMessages handles bidirectional gRPC streaming for real-time chat.
// It manages the connection lifecycle by registering it with the connection.Manager.
func (l *StreamMessagesLogic) StreamMessages(stream pb.ChatService_StreamMessagesServer) error {
	// 1. Authenticate user from gRPC metadata
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	// Extract user_id from metadata (passed from gateway/interceptor)
	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return status.Errorf(codes.Unauthenticated, "user_id is missing in metadata")
	}

	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid user_id format")
	}

	// Extract platform info
	platform := "grpc"
	if p := md.Get("platform"); len(p) > 0 {
		platform = p[0]
	}

	// 2. Wrap the gRPC stream into a connection.Connection interface
	connID := uuid.New().String()
	conn := connection.NewGrpcConnection(connID, userId, platform, stream, l.svcCtx.Manager)

	// 3. Register the connection with the global manager
	l.svcCtx.Manager.Register(conn)
	// Ensure cleanup on disconnect
	defer func() {
		l.svcCtx.Manager.Unregister(conn)
		l.Logger.Infof("User %d disconnected, connID: %s", userId, connID)
	}()

	l.Logger.Infof("User %d connected via gRPC stream, connID: %s, platform: %s", userId, connID, platform)

	// 4. Enter the receive loop to process messages from the client
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			l.Logger.Errorf("Stream error for user %d: %v", userId, err)
			return err
		}

		// Handle the incoming proto message
		// In a real migration, we would route this to a Kafka producer or directly
		// process it. For now, we log the activity.
		l.Logger.Debugf("Received message from user %d: Type=%v, TraceId=%s", userId, req.Type, req.TraceId)

		// TODO: Implement bridging logic between Proto messages and existing Manager handlers
		// For example, converting req to JSON and calling l.svcCtx.Manager.HandleMessage(conn, data)
	}
}
