package service

import (
	"github.com/archyhsh/gochat/internal/message/model"
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type MessageService struct {
	db     *gorm.DB
	logger logger.Logger
}

func NewMessageService(db *gorm.DB, log logger.Logger) *MessageService {
	return &MessageService{
		db:     db,
		logger: log,
	}
}

func (s *MessageService) SaveMessage(msg *model.Message) error {
	result := s.db.Table(msg.TableName()).Create(msg)
	if result.Error != nil {
		s.logger.Error("Failed to save message",
			"msgID", msg.MsgID,
			"error", result.Error,
		)
		return result.Error
	}
	s.logger.Debug("Message saved",
		"msgID", msg.MsgID,
		"conversationID", msg.ConversationID,
		"table", msg.TableName(),
	)
	return nil
}

func (s *MessageService) GetMessageByID(msgID string) (*model.Message, error) {
	var msg model.Message
	result := s.db.Where("msg_id = ?", msgID).First(&msg)
	if result.Error != nil {
		return nil, result.Error
	}
	return &msg, nil
}

func (s *MessageService) GetConversationMessages(conversationID string, limit, offset int) ([]model.Message, error) {
	var messages []model.Message
	result := s.db.Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages)
	if result.Error != nil {
		return nil, result.Error
	}
	return messages, nil
}

func (s *MessageService) UpdateMessageStatus(msgID string, status int) error {
	result := s.db.Model(&model.Message{}).
		Where("msg_id = ?", msgID).
		Update("status", status)
	return result.Error
}

func (s *MessageService) MarkAsRead(msgID string, userID int64) error {
	read := &model.MessageRead{
		MsgID:  msgID,
		UserID: userID,
	}
	result := s.db.Create(read)
	if result.Error != nil {
		s.logger.Debug("Mark message as read",
			"msgID", msgID,
			"userID", userID,
		)
	}
	return nil
}
