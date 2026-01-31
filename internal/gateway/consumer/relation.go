package consumer

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
	"github.com/archyhsh/gochat/internal/gateway/connection"
	"github.com/archyhsh/gochat/internal/gateway/handler"
	"github.com/archyhsh/gochat/internal/relation/model"
	"github.com/archyhsh/gochat/pkg/logger"
)

type RelationEventConsumer struct {
	manager *connection.Manager
	logger  logger.Logger
}

func NewRelationEventConsumer(manager *connection.Manager, log logger.Logger) *RelationEventConsumer {
	return &RelationEventConsumer{
		manager: manager,
		logger:  log,
	}
}

func (c *RelationEventConsumer) Handle(ctx context.Context, message *sarama.ConsumerMessage) error {
	var event model.RelationEvent
	if err := json.Unmarshal(message.Value, &event); err != nil {
		c.logger.Error("Failed to unmarshal relation event", "error", err)
		return err
	}

	c.logger.Info("Received relation event",
		"type", event.Type,
		"userID", event.UserID,
		"peerID", event.PeerID,
		"action", event.Action,
	)

	switch event.Type {
	case "block":
		return c.handleBlockEvent(&event)
	case "friend":
		return c.handleFriendEvent(&event)
	default:
		c.logger.Warn("Unknown relation event type", "type", event.Type)
	}

	return nil
}

func (c *RelationEventConsumer) handleBlockEvent(event *model.RelationEvent) error {
	msg := handler.OutgoingMessage{
		Type: "relation_changed",
		Data: map[string]interface{}{
			"type":    "block",
			"user_id": event.UserID,
			"action":  event.Action,
			"peer_id": event.PeerID,
		},
	}
	data, _ := json.Marshal(msg)

	if err := c.manager.SendToUser(event.PeerID, data); err != nil {
		c.logger.Warn("Failed to send relation event to user", "userID", event.PeerID, "error", err)
	}

	c.logger.Info("Relation event sent to user", "userID", event.PeerID, "action", event.Action)
	return nil
}

func (c *RelationEventConsumer) handleFriendEvent(event *model.RelationEvent) error {
	users := []int64{event.UserID, event.PeerID}
	msg := handler.OutgoingMessage{
		Type: "relation_changed",
		Data: map[string]interface{}{
			"type":    "friend",
			"user_id": event.UserID,
			"peer_id": event.PeerID,
			"action":  event.Action,
		},
	}
	data, _ := json.Marshal(msg)
	for _, uid := range users {
		if err := c.manager.SendToUser(uid, data); err != nil {
			c.logger.Warn("Failed to send relation event to user", "userID", uid, "error", err)
		}
	}
	c.logger.Info("Relation event sent to users", "users", users, "action", event.Action)
	return nil
}
