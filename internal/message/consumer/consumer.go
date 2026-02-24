package consumer

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"github.com/archyhsh/gochat/internal/message/model"
	"github.com/archyhsh/gochat/internal/message/service"
	"github.com/archyhsh/gochat/pkg/logger"
)

type MessageConsumer struct {
	msgService *service.MessageService
	logger     logger.Logger
}

func NewMessageConsumer(msgService *service.MessageService, log logger.Logger) *MessageConsumer {
	return &MessageConsumer{
		msgService: msgService,
		logger:     log,
	}
}

func (c *MessageConsumer) Handle(ctx context.Context, message *sarama.ConsumerMessage) error {
	c.logger.Debug("Received Kafka message",
		"topic", message.Topic,
		"partition", message.Partition,
		"offset", message.Offset,
		"key", string(message.Key),
	)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(message.Value, &raw); err != nil {
		return err
	}

	var msgType string
	json.Unmarshal(raw["type"], &msgType)

	switch msgType {
	case "chat":
		var kafkaMsg model.KafkaChatMessage
		json.Unmarshal(message.Value, &kafkaMsg)
		return c.handleChatMessage(ctx, &kafkaMsg)
	case "ack":
		return c.handleAckMessage(ctx, raw["data"])
	case "read":
		return c.handleReadMessage(ctx, raw["data"])
	default:
		c.logger.Warn("Unknown message type", "type", msgType)
	}
	return nil
}

func (c *MessageConsumer) handleChatMessage(ctx context.Context, kafkaMsg *model.KafkaChatMessage) error {
	msg := kafkaMsg.Data.ToMessage()
	if err := c.msgService.SaveMessage(msg); err != nil {
		c.logger.Error("Failed to save chat message",
			"msgID", msg.MsgID,
			"error", err,
		)
		return err
	}
	c.logger.Info("Chat message persisted",
		"msgID", msg.MsgID,
		"conversationID", msg.ConversationID,
	)
	return nil
}

func (c *MessageConsumer) handleAckMessage(ctx context.Context, data json.RawMessage) error {
	var ack model.AckMessage
	if err := json.Unmarshal(data, &ack); err != nil {
		return err
	}
	if err := c.msgService.UpdateMessageStatus(ack.MsgID, ack.Status); err != nil {
		c.logger.Error("Failed to update message status", "msgID", ack.MsgID, "error", err)
		return err
	}
	return nil
}

func (c *MessageConsumer) handleReadMessage(ctx context.Context, data json.RawMessage) error {
	var read model.ReadMessage
	if err := json.Unmarshal(data, &read); err != nil {
		return err
	}
	// For simplicity, we just log it or update a conversation read pointer.
	// In a real system, you might mark all messages in MsgIDs as read.
	for _, msgID := range read.MsgIDs {
		// Note: We don't have userID here unless it's in the ReadMessage
		// For now, we update the status of individual messages if needed
		c.msgService.UpdateMessageStatus(msgID, 3) // 3 = Read
	}
	return nil
}
