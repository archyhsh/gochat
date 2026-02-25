package service

import (
	"fmt"
	"strconv"
	"strings"
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
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Table(msg.TableName()).Create(msg)
		if result.Error != nil {
			s.logger.Error("Failed to save message", "msgID", msg.MsgID, "error", result.Error)
			return result.Error
		}
		if err := s.updateConversation(tx, msg.SenderID, msg, false); err != nil {
			return err
		}
		if msg.GroupID > 0 {
			var memberIDs []int64
			tx.Table("group_member").Where("group_id = ? AND user_id != ?", msg.GroupID, msg.SenderID).Pluck("user_id", &memberIDs)
			for _, memberID := range memberIDs {
				if err := s.updateConversation(tx, memberID, msg, true); err != nil {
					s.logger.Warn("Failed to update group member conversation", "memberID", memberID, "error", err)
				}
			}
		} else {
			if err := s.updateConversation(tx, msg.ReceiverID, msg, true); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *MessageService) updateConversation(tx *gorm.DB, userID int64, msg *model.Message, incrementUnread bool) error {
	if userID <= 0 {
		return nil
	}

	msgTime := msg.CreatedAt
	if msgTime.IsZero() {
		msgTime = time.Now()
	}
	updateSQL := `UPDATE user_conversation 
				 SET last_msg_id = ?, last_msg_time = ?, 
				 unread_count = IF(?, unread_count + 1, unread_count),
				 is_deleted = 0
				 WHERE user_id = ? AND conversation_id = ?`

	result := tx.Exec(updateSQL, msg.MsgID, msgTime, incrementUnread, userID, msg.ConversationID)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		unreadCount := 0
		if incrementUnread {
			unreadCount = 1
		}
		var peerID int64
		if msg.GroupID > 0 {
			peerID = msg.GroupID
		} else {
			if msg.SenderID == userID {
				peerID = msg.ReceiverID
			} else {
				peerID = msg.SenderID
			}
		}

		insertSQL := `INSERT INTO user_conversation 
					 (user_id, conversation_id, peer_id, last_msg_id, last_msg_time, unread_count, is_deleted) 
					 VALUES (?, ?, ?, ?, ?, ?, 0)`
		return tx.Exec(insertSQL, userID, msg.ConversationID, peerID, msg.MsgID, msgTime, unreadCount).Error
	}

	return nil
}

func (s *MessageService) ClearUnread(userID int64, conversationID string) error {
	return s.db.Table("user_conversation").
		Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Update("unread_count", 0).Error
}

func (s *MessageService) GetMessageByID(msgID string) (*model.Message, error) {
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
		var msg model.Message
		result := s.db.Table(table).Where("msg_id = ?", msgID).First(&msg)
		if result.Error == nil {
			return &msg, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *MessageService) GetConversationMessages(conversationID string, limit, offset int) ([]model.Message, error) {
	var messages []model.Message
	now := time.Now()
	tables := []string{
		model.MessageTableName(now),
		model.MessageTableName(now.AddDate(0, -1, 0)),
		model.MessageTableName(now.AddDate(0, -2, 0)),
	}

	currentOffset := offset
	for _, table := range tables {
		if !s.tableExists(table) {
			continue
		}
		var count int64
		s.db.Table(table).Where("conversation_id = ?", conversationID).Count(&count)
		if count == 0 {
			continue
		}
		if int64(currentOffset) >= count {
			currentOffset -= int(count)
			continue
		}

		var tableMessages []model.Message
		result := s.db.Table(table).
			Where("conversation_id = ?", conversationID).
			Order("created_at DESC").
			Limit(limit - len(messages)).
			Offset(currentOffset).
			Find(&tableMessages)

		if result.Error == nil {
			messages = append(messages, tableMessages...)
			currentOffset = 0
		}

		if len(messages) >= limit {
			break
		}
	}

	s.logger.Debug("Retrieved messages", "conversationID", conversationID, "count", len(messages))
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

	sql := `SELECT conversation_id, peer_id, unread_count, last_msg_id, last_msg_time as last_message_time
			FROM user_conversation
			WHERE user_id = ? AND is_deleted = 0
			ORDER BY last_msg_time DESC
			LIMIT ?`

	err := s.db.Raw(sql, userID, limit).Scan(&conversations).Error

	if err != nil {
		s.logger.Error("Failed to get user conversations", "userID", userID, "error", err)
		return nil, err
	}

	for i := range conversations {
		if conversations[i].LastMsgID != "" {
			msg, err := s.GetMessageByID(conversations[i].LastMsgID)
			if err == nil {
				conversations[i].LastMessage = msg.Content
				conversations[i].LastMsgType = msg.MsgType
				conversations[i].LastSenderID = msg.SenderID
			}
		}
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

func (s *MessageService) CheckPermission(userID int64, conversationID string) (bool, error) {
	// Private chat
	if strings.HasPrefix(conversationID, "conv_") {
		parts := strings.Split(conversationID, "_")
		if len(parts) != 3 {
			return false, nil
		}
		id1, _ := strconv.ParseInt(parts[1], 10, 64)
		id2, _ := strconv.ParseInt(parts[2], 10, 64)
		return userID == id1 || userID == id2, nil
	}

	// Group chat
	if strings.HasPrefix(conversationID, "group_") {
		groupID, err := strconv.ParseInt(strings.TrimPrefix(conversationID, "group_"), 10, 64)
		if err != nil {
			return false, nil
		}
		var count int64
		err = s.db.Table("group_member").
			Where("group_id = ? AND user_id = ?", groupID, userID).
			Count(&count).Error
		return count > 0, err
	}

	return false, nil
}
