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
	var kafkaMsg model.KafkaChatMessage
	if err := json.Unmarshal(message.Value, &kafkaMsg); err != nil {
		c.logger.Error("Failed to unmarshal Kafka message",
			"error", err,
			"value", string(message.Value),
		)
		return err
	}
	switch kafkaMsg.Type {
	case "chat":
		return c.handleChatMessage(ctx, &kafkaMsg)
	default:
		c.logger.Warn("Unknown message type",
			"type", kafkaMsg.Type,
		)
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
		"senderID", msg.SenderID,
		"traceID", kafkaMsg.TraceID,
	)
	return nil
}
