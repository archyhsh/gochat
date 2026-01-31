package service

import (
	"fmt"
	"time"

	"github.com/archyhsh/gochat/internal/message/model"
	"github.com/archyhsh/gochat/pkg/logger"
	"gorm.io/gorm"
)

type MessageService struct {
	db     *gorm.DB
	logger logger.Logger
}

func NewMessageService(db *gorm.DB, log logger.Logger) *MessageService {
	s := &MessageService{
		db:     db,
		logger: log,
	}
	s.ensureTableExists()
	return s
}

func (s *MessageService) ensureTableExists() {
	tableName := model.MessageTableName(time.Now())
	if s.tableExists(tableName) {
		s.logger.Debug("Message table already exists", "table", tableName)
		return
	}
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			msg_id VARCHAR(64) NOT NULL,
			conversation_id VARCHAR(64) NOT NULL,
			sender_id BIGINT NOT NULL,
			receiver_id BIGINT DEFAULT 0,
			group_id BIGINT DEFAULT 0,
			msg_type INT DEFAULT 1,
			content TEXT,
			status INT DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uk_msg_id (msg_id),
			KEY idx_conversation_id (conversation_id),
			KEY idx_sender_id (sender_id),
			KEY idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, tableName)
	if err := s.db.Exec(createSQL).Error; err != nil {
		s.logger.Error("Failed to create message table", "table", tableName, "error", err)
	} else {
		s.logger.Info("Message table created", "table", tableName)
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
	now := time.Now()
	tables := []string{
		model.MessageTableName(now),
		model.MessageTableName(now.AddDate(0, -1, 0)),
		model.MessageTableName(now.AddDate(0, -2, 0)),
	}
	for _, table := range tables {
		if !s.tableExists(table) {
			continue
		}
		var tableMessages []model.Message
		result := s.db.Table(table).
			Where("conversation_id = ?", conversationID).
			Order("created_at DESC").
			Limit(limit - len(messages)).
			Offset(offset).
			Find(&tableMessages)
		if result.Error == nil {
			messages = append(messages, tableMessages...)
		}
		if len(messages) >= limit {
			break
		}
	}
	return messages, nil
}

func (s *MessageService) GetConversationMessagesSimple(conversationID string, limit, offset int) ([]model.Message, error) {
	var messages []model.Message
	table := model.MessageTableName(time.Now())
	if !s.tableExists(table) {
		return messages, nil
	}
	result := s.db.Table(table).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages)

	if result.Error != nil {
		return nil, result.Error
	}
	return messages, nil
}

func (s *MessageService) GetUserConversations(userID int64, limit int) ([]model.ConversationInfo, error) {
	var conversations []model.ConversationInfo
	table := model.MessageTableName(time.Now())

	if !s.tableExists(table) {
		return conversations, nil
	}
	sql := fmt.Sprintf(`
		SELECT 
			m.conversation_id,
			m.content as last_message,
			m.msg_type as last_msg_type,
			m.sender_id as last_sender_id,
			m.created_at as last_message_time,
			CASE 
				WHEN m.sender_id = ? THEN m.receiver_id 
				ELSE m.sender_id 
			END as peer_id
		FROM %s m
		INNER JOIN (
			SELECT conversation_id, MAX(created_at) as max_time
			FROM %s
			WHERE sender_id = ? OR receiver_id = ?
			GROUP BY conversation_id
		) latest ON m.conversation_id = latest.conversation_id AND m.created_at = latest.max_time
		ORDER BY m.created_at DESC
		LIMIT ?
	`, table, table)
	result := s.db.Raw(sql, userID, userID, userID, limit).Scan(&conversations)
	if result.Error != nil {
		s.logger.Error("Failed to get user conversations",
			"userID", userID,
			"error", result.Error,
		)
		return nil, result.Error
	}
	return conversations, nil
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

func (s *MessageService) tableExists(tableName string) bool {
	var count int64
	s.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?", tableName).Scan(&count)
	return count > 0
}
