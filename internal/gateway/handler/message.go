package handler

import (
	"encoding/json"

	"github.com/yourusername/gochat/internal/gateway/connection"
	"github.com/yourusername/gochat/pkg/logger"
)

// MessageType 消息类型
type MessageType string

const (
	TypeChat      MessageType = "chat"      // 聊天消息
	TypeAck       MessageType = "ack"       // 消息确认
	TypeRead      MessageType = "read"      // 已读回执
	TypeTyping    MessageType = "typing"    // 正在输入
	TypeOnline    MessageType = "online"    // 上线通知
	TypeOffline   MessageType = "offline"   // 下线通知
	TypeHeartbeat MessageType = "heartbeat" // 心跳
	TypeError     MessageType = "error"     // 错误
)

// IncomingMessage 接收的消息
type IncomingMessage struct {
	Type    MessageType     `json:"type"`
	Data    json.RawMessage `json:"data"`
	TraceID string          `json:"trace_id,omitempty"`
}

// OutgoingMessage 发送的消息
type OutgoingMessage struct {
	Type    MessageType `json:"type"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"trace_id,omitempty"`
}

// ChatMessage 聊天消息
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

// AckMessage 确认消息
type AckMessage struct {
	MsgID  string `json:"msg_id"`
	Status int    `json:"status"` // 1: 已发送, 2: 已送达, 3: 已读
}

// ReadMessage 已读消息
type ReadMessage struct {
	ConversationID string   `json:"conversation_id"`
	MsgIDs         []string `json:"msg_ids"`
}

// TypingMessage 正在输入
type TypingMessage struct {
	ConversationID string `json:"conversation_id"`
	IsTyping       bool   `json:"is_typing"`
}

// ErrorMessage 错误消息
type ErrorMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MessageHandler 消息处理器
type MessageHandler struct {
	manager *connection.Manager
	logger  logger.Logger
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(manager *connection.Manager, log logger.Logger) *MessageHandler {
	return &MessageHandler{
		manager: manager,
		logger:  log,
	}
}

// Handle 处理消息
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

// handleChat 处理聊天消息
func (h *MessageHandler) handleChat(conn *connection.Connection, msg IncomingMessage) error {
	var chatMsg ChatMessage
	if err := json.Unmarshal(msg.Data, &chatMsg); err != nil {
		h.sendError(conn, 400, "invalid chat message")
		return err
	}

	// 设置发送者
	chatMsg.SenderID = conn.UserID

	// TODO: 发送到 Kafka 进行异步处理
	// TODO: 持久化到数据库

	// 如果是私聊，直接发送给接收者
	if chatMsg.ReceiverID > 0 {
		outMsg := OutgoingMessage{
			Type:    TypeChat,
			Data:    chatMsg,
			TraceID: msg.TraceID,
		}
		data, _ := json.Marshal(outMsg)
		h.manager.SendToUser(chatMsg.ReceiverID, data)

		// 发送 ACK 给发送者
		h.sendAck(conn, chatMsg.MsgID, 1, msg.TraceID)
	}

	// TODO: 如果是群聊，发送给群内所有成员

	return nil
}

// handleAck 处理确认消息
func (h *MessageHandler) handleAck(conn *connection.Connection, msg IncomingMessage) error {
	var ackMsg AckMessage
	if err := json.Unmarshal(msg.Data, &ackMsg); err != nil {
		return err
	}

	// TODO: 更新消息状态

	return nil
}

// handleRead 处理已读回执
func (h *MessageHandler) handleRead(conn *connection.Connection, msg IncomingMessage) error {
	var readMsg ReadMessage
	if err := json.Unmarshal(msg.Data, &readMsg); err != nil {
		return err
	}

	// TODO: 更新消息已读状态
	// TODO: 通知发送者消息已读

	return nil
}

// handleTyping 处理正在输入
func (h *MessageHandler) handleTyping(conn *connection.Connection, msg IncomingMessage) error {
	var typingMsg TypingMessage
	if err := json.Unmarshal(msg.Data, &typingMsg); err != nil {
		return err
	}

	// TODO: 通知对方正在输入

	return nil
}

// handleHeartbeat 处理心跳
func (h *MessageHandler) handleHeartbeat(conn *connection.Connection, msg IncomingMessage) error {
	// 回复心跳
	outMsg := OutgoingMessage{
		Type:    TypeHeartbeat,
		Data:    map[string]string{"status": "pong"},
		TraceID: msg.TraceID,
	}
	data, _ := json.Marshal(outMsg)
	return conn.SendMessage(json.RawMessage(data))
}

// sendError 发送错误消息
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

// sendAck 发送确认消息
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
