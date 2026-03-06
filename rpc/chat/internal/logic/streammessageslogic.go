package logic

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/archyhsh/gochat/pkg/manager"
	"github.com/archyhsh/gochat/rpc/chat/internal/svc"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/pb"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

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

func (l *StreamMessagesLogic) StreamMessages(stream pb.ChatService_StreamMessagesServer) error {
	md, ok := metadata.FromIncomingContext(l.ctx)
	if !ok {
		return status.Errorf(codes.Unauthenticated, "metadata is not provided")
	}

	userIdStrs := md.Get("user_id")
	if len(userIdStrs) == 0 {
		return status.Errorf(codes.Unauthenticated, "user_id is missing in metadata")
	}

	userId, err := strconv.ParseInt(userIdStrs[0], 10, 64)
	if err != nil {
		return status.Errorf(codes.Unauthenticated, "invalid user_id format")
	}

	platform := "grpc"
	if p := md.Get("platform"); len(p) > 0 {
		platform = p[0]
	}

	connID := uuid.New().String()
	conn := manager.NewGrpcConnection(connID, userId, platform, stream)
	l.svcCtx.Manager.Register(conn)

	if err := l.svcCtx.Router.Register(l.ctx, userId); err != nil {
		l.Errorf("Global router registration failed for user %d: %v", userId, err)
	}

	defer func() {
		l.svcCtx.Manager.Unregister(conn)
		_ = l.svcCtx.Router.Unregister(context.Background(), userId)
		l.Logger.Infof("User %d disconnected (connID: %s)", userId, connID)
	}()

	l.Logger.Infof("User %d connected via gRPC stream (connID: %s, platform: %s)", userId, connID, platform)

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			l.Logger.Errorf("Stream recv error for user %d: %v", userId, err)
			return err
		}

		l.Logger.Debugf("Received message from user %d: Type=%v, TraceId=%s", userId, req.Type, req.TraceId)

		switch req.Type {
		case pb.IncomingMessage_TYPE_CHAT:
			if err := l.handleChat(userId, req); err != nil {
				l.sendError(stream, req.TraceId, err.Error())
				continue
			}
		case pb.IncomingMessage_TYPE_ACK, pb.IncomingMessage_TYPE_READ:
			// Forwarding feedback events to Kafka for async DB update
			if err := l.produceToKafka(userId, req); err != nil {
				l.sendError(stream, req.TraceId, "system busy")
				continue
			}
		case pb.IncomingMessage_TYPE_TYPING:
			l.Logger.Debugf("User %d is typing in conversation %s", userId, req.GetTypingMsg().GetConversationId())
		case pb.IncomingMessage_TYPE_HEARTBEAT:
			_ = stream.Send(&pb.OutgoingMessage{
				Type:    pb.OutgoingMessage_TYPE_HEARTBEAT,
				TraceId: req.TraceId,
			})
			continue
		}

		_ = stream.Send(&pb.OutgoingMessage{
			Type:    pb.OutgoingMessage_TYPE_ACK,
			TraceId: req.TraceId,
			Payload: &pb.OutgoingMessage_AckMsg{
				AckMsg: &pb.AckPayload{
					MsgId:  req.GetChatMsg().GetMsgId(),
					Status: 1,
				},
			},
		})
	}
}

func (l *StreamMessagesLogic) handleChat(userId int64, req *pb.IncomingMessage) error {
	chatMsg := req.GetChatMsg()
	if chatMsg == nil {
		return status.Error(codes.InvalidArgument, "empty chat message")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if chatMsg.GroupId > 0 {
		resp, err := l.svcCtx.GroupRpc.CheckGroupMember(ctx, &groupservice.CheckGroupMemberRequest{
			UserId:  userId,
			GroupId: chatMsg.GroupId,
		})
		if err != nil || !resp.IsMember {
			return status.Error(codes.PermissionDenied, "not a member of this group")
		}
	} else if chatMsg.ReceiverId > 0 {
		resp, err := l.svcCtx.RelationRpc.CheckFriend(ctx, &relationservice.CheckFriendRequest{
			UserId:   userId,
			FriendId: chatMsg.ReceiverId,
		})
		if err != nil || !resp.IsFriend {
			return status.Error(codes.PermissionDenied, "not friends with recipient")
		}
		if resp.IsBlocked {
			return status.Error(codes.PermissionDenied, "blocked by recipient")
		}
	}

	return l.produceToKafka(userId, req)
}

func (l *StreamMessagesLogic) produceToKafka(userId int64, req *pb.IncomingMessage) error {
	if l.svcCtx.Producer == nil {
		return nil
	}

	event := &pb.ChatMessageEvent{
		SenderId:  userId,
		TraceId:   req.TraceId,
		Timestamp: time.Now().UnixMilli(),
	}

	switch req.Type {
	case pb.IncomingMessage_TYPE_CHAT:
		msg := req.GetChatMsg()
		event.MsgId = msg.MsgId
		event.ConversationId = msg.ConversationId
		event.ReceiverId = msg.ReceiverId
		event.GroupId = msg.GroupId
		event.MsgType = msg.MsgType
		event.Content = msg.Content
	case pb.IncomingMessage_TYPE_ACK:
		// immediate return for ACKs, no need to produce to Kafka
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	return l.svcCtx.Producer.Send([]byte(event.MsgId), data)
}

func (l *StreamMessagesLogic) sendError(stream pb.ChatService_StreamMessagesServer, traceId string, msg string) {
	_ = stream.Send(&pb.OutgoingMessage{
		Type:    pb.OutgoingMessage_TYPE_ERROR,
		TraceId: traceId,
		Payload: &pb.OutgoingMessage_ErrorMsg{
			ErrorMsg: &pb.ErrorPayload{
				Code:    400,
				Message: msg,
			},
		},
	})
}
