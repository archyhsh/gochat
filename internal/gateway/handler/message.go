package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/archyhsh/gochat/internal/gateway/connection"
	"github.com/archyhsh/gochat/internal/gateway/service"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/logger"
)

type MessageType string

const (
	TypeChat      MessageType = "chat"
	TypeAck       MessageType = "ack"
	TypeRead      MessageType = "read"
	TypeTyping    MessageType = "typing"
	TypeOnline    MessageType = "online"
	TypeOffline   MessageType = "offline"
	TypeHeartbeat MessageType = "heartbeat"
	TypeError     MessageType = "error"
)

type IncomingMessage struct {
	Type    MessageType     `json:"type"`
	Data    json.RawMessage `json:"data"`
	TraceID string          `json:"trace_id,omitempty"`
}

type OutgoingMessage struct {
	Type    MessageType `json:"type"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"trace_id,omitempty"`
}

type ChatMessage struct {
	MsgID          string `json:"msg_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	ReceiverID     int64  `json:"receiver_id,omitempty"`
	GroupID        int64  `json:"group_id,omitempty"`
	MsgType        int    `json:"msg_type"`
	Content        string `json:"content"`
	Timestamp      int64  `json:"timestamp"`
}

// status=1: sent, status=2: delivered, status=3: read
type AckMessage struct {
	MsgID  string `json:"msg_id"`
	Status int    `json:"status"`
}

type ReadMessage struct {
	ConversationID string   `json:"conversation_id"`
	MsgIDs         []string `json:"msg_ids"`
}

type TypingMessage struct {
	ConversationID string `json:"conversation_id"`
	IsTyping       bool   `json:"is_typing"`
}

type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type KafkaMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"trace_id,omitempty"`
}

type MessageHandler struct {
	manager         *connection.Manager
	producer        *kafka.Producer
	relationChecker *service.RelationChecker
	logger          logger.Logger
}

func NewMessageHandler(manager *connection.Manager, producer *kafka.Producer, relationChecker *service.RelationChecker, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		manager:         manager,
		producer:        producer,
		relationChecker: relationChecker,
		logger:          log,
	}
}

func (h *MessageHandler) Handle(conn *connection.Connection, data []byte) error {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		h.sendError(conn, 400, "invalid message format")
		return err
	}
	h.logger.Debug("Received message",
		"connID", conn.ID,
		"userID", conn.UserID,
		"type", msg.Type,
	)
	switch msg.Type {
	case TypeChat:
		return h.handleChat(conn, msg)
	case TypeAck:
		return h.handleAck(conn, msg)
	case TypeRead:
		return h.handleRead(conn, msg)
	case TypeTyping:
		return h.handleTyping(conn, msg)
	case TypeHeartbeat:
		return h.handleHeartbeat(conn, msg)
	default:
		h.sendError(conn, 400, "unknown message type")
		return nil
	}
}

func (h *MessageHandler) handleChat(conn *connection.Connection, msg IncomingMessage) error {
	var chatMsg ChatMessage
	if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
		h.sendError(conn, 400, "invalid chat message")
		return err
	}
	chatMsg.SenderID = conn.UserID
	chatMsg.Timestamp = time.Now().UnixMilli()
	// one-to-one chat
	if chatMsg.ReceiverID > 0 {
		if h.relationChecker != nil {
			if !h.relationChecker.IsFriend(chatMsg.SenderID, chatMsg.ReceiverID) {
				h.sendError(conn, 403, "not friends with this user")
				return nil
			}
			senderBlocked, receiverBlocked := h.relationChecker.GetBlockStatus(chatMsg.SenderID, chatMsg.ReceiverID)
			if senderBlocked || receiverBlocked {
				h.logger.Info("Message blocked due to block status (local only)",
					"senderID", chatMsg.SenderID,
					"receiverID", chatMsg.ReceiverID,
					"senderBlocked", senderBlocked,
					"receiverBlocked", receiverBlocked,
				)
				h.sendAck(conn, chatMsg.MsgID, 1, msg.TraceID)
				return nil
			}
		}
		if h.producer != nil {
			kafkaMsg := KafkaMessage{
				Type:    string(TypeChat),
				Data:    chatMsg,
				TraceID: msg.TraceID,
			}
			data, _ := json.Marshal(kafkaMsg)
			if err := h.producer.Send([]byte(chatMsg.MsgID), data); err != nil {
				h.logger.Error("Failed to send message to Kafka",
					"msgID", chatMsg.MsgID,
					"error", err,
				)
			} else {
				h.logger.Debug("Message sent to Kafka",
					"msgID", chatMsg.MsgID,
					"conversationID", chatMsg.ConversationID,
				)
			}
		}
		outMsg := OutgoingMessage{
			Type:    TypeChat,
			Data:    chatMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(outMsg)
		h.logger.Info("Forwarding message to receiver",
			"senderID", chatMsg.SenderID,
			"receiverID", chatMsg.ReceiverID,
			"msgID", chatMsg.MsgID,
			"isReceiverOnline", h.manager.IsUserOnline(chatMsg.ReceiverID),
		)

		if err := h.manager.SendToUser(chatMsg.ReceiverID, data); err != nil {
			h.logger.Warn("Failed to forward message",
				"receiverID", chatMsg.ReceiverID,
				"error", err,
			)
		}
		h.sendAck(conn, chatMsg.MsgID, 1, msg.TraceID)
	}

	// Group chat message
	if chatMsg.GroupID > 0 {
		if h.relationChecker != nil {
			if !h.relationChecker.IsGroupMember(chatMsg.SenderID, chatMsg.GroupID) {
				h.logger.Warn("User not in group", "userID", chatMsg.SenderID, "groupID", chatMsg.GroupID)
				h.sendError(conn, 403, "not a member of this group")
				return nil
			}
		}
		if h.producer != nil {
			kafkaMsg := KafkaMessage{
				Type:    string(TypeChat),
				Data:    chatMsg,
				TraceID: msg.TraceID,
			}
			data, _ := json.Marshal(kafkaMsg)
			h.producer.Send([]byte(chatMsg.MsgID), data)
		}
		if h.relationChecker != nil {
			members, err := h.relationChecker.GetGroupMembers(chatMsg.GroupID)
			if err != nil {
				h.logger.Error("Failed to get group members", "groupID", chatMsg.GroupID, "error", err)
			} else {
				outMsg := OutgoingMessage{
					Type:    TypeChat,
					Data:    chatMsg,
					TraceID: msg.TraceID,
				}
				data, _ := json.Marshal(outMsg)
				for _, memberID := range members {
					// Skip sender if you want (standard IM usually sends it back or relies on local echo)
					// Here we send it to everyone to ensure consistency
					if memberID == chatMsg.SenderID {
						continue
					}
					h.manager.SendToUser(memberID, data)
				}
			}
		}
		h.sendAck(conn, chatMsg.MsgID, 1, msg.TraceID)
		return nil
	}

	return nil
}

func (h *MessageHandler) handleAck(conn *connection.Connection, msg IncomingMessage) error {
	var ackMsg AckMessage
	if err := json.Unmarshal(msg.Data, &ackMsg); err != nil {
		return err
	}

	// Send to Kafka for status update in DB
	if h.producer != nil {
		kafkaMsg := KafkaMessage{
			Type:    string(TypeAck),
			Data:    ackMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(kafkaMsg)
		h.producer.Send([]byte(ackMsg.MsgID), data)
	}

	return nil
}

func (h *MessageHandler) handleRead(conn *connection.Connection, msg IncomingMessage) error {
	var readMsg ReadMessage
	if err := json.Unmarshal(msg.Data, &readMsg); err != nil {
		return err
	}
	if h.producer != nil {
		kafkaMsg := KafkaMessage{
			Type:    string(TypeRead),
			Data:    readMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(kafkaMsg)
		h.producer.Send([]byte(readMsg.ConversationID), data)
	}
	peerID := h.getPeerID(readMsg.ConversationID, conn.UserID)
	if peerID > 0 {
		outMsg := OutgoingMessage{
			Type:    TypeRead,
			Data:    readMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(outMsg)
		h.manager.SendToUser(peerID, data)
	}
	return nil
}

func (h *MessageHandler) handleTyping(conn *connection.Connection, msg IncomingMessage) error {
	var typingMsg TypingMessage
	if err := json.Unmarshal(msg.Data, &typingMsg); err != nil {
		return err
	}
	peerID := h.getPeerID(typingMsg.ConversationID, conn.UserID)
	if peerID > 0 {
		outMsg := OutgoingMessage{
			Type:    TypeTyping,
			Data:    typingMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(outMsg)
		h.manager.SendToUser(peerID, data)
	}

	return nil
}

func (h *MessageHandler) getPeerID(convID string, userID int64) int64 {
	if !strings.HasPrefix(convID, "conv_") {
		return 0
	}
	parts := strings.Split(convID, "_")
	if len(parts) != 3 {
		return 0
	}
	id1, _ := strconv.ParseInt(parts[1], 10, 64)
	id2, _ := strconv.ParseInt(parts[2], 10, 64)
	if id1 == userID {
		return id2
	}
	return id1
}

func (h *MessageHandler) handleHeartbeat(conn *connection.Connection, msg IncomingMessage) error {
	outMsg := OutgoingMessage{
		Type:    TypeHeartbeat,
		Data:    map[string]string{"status": "pong"},
		TraceID: msg.TraceID,
	}
	data, _ := json.Marshal(outMsg)
	return conn.SendMessage(json.RawMessage(data))
}

func (h *MessageHandler) sendError(conn *connection.Connection, code int, message string) {
	outMsg := OutgoingMessage{
		Type: TypeError,
		Data: ErrorMessage{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(outMsg)
	conn.SendMessage(json.RawMessage(data))
}

func (h *MessageHandler) sendAck(conn *connection.Connection, msgID string, status int, traceID string) {
	outMsg := OutgoingMessage{
		Type: TypeAck,
		Data: AckMessage{
			MsgID:  msgID,
			Status: status,
		},
		TraceID: traceID,
	}
	data, _ := json.Marshal(outMsg)
	conn.SendMessage(json.RawMessage(data))
}
