package model

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserConversationModel = (*customUserConversationModel)(nil)

type (
	// UserConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserConversationModel.
	UserConversationModel interface {
		userConversationModel
		FindOneByUserIdConversationId(ctx context.Context, userId int64, conversationId string) (*UserConversation, error)
		GetUserConversationsByUserId(ctx context.Context, userId int64) ([]*UserConversationWithSeq, error)
		UpdateNewPrivateMsg(ctx context.Context, session sqlx.Session, userId int64, peerId int64, conversationId string, lastMsg *MessageTemplate, incUnread bool) error
		UpdateReadSequence(ctx context.Context, userId int64, conversationId string, seq int64) error
	}

	customUserConversationModel struct {
		*defaultUserConversationModel
	}

	// UserConversationWithSeq includes shared fields from the global conversation table
	UserConversationWithSeq struct {
		UserConversation
		GlobalLastMsgId      string    `db:"global_last_msg_id"`
		GlobalLastMsgTime    time.Time `db:"global_last_msg_time"`
		GlobalLastMsgContent string    `db:"global_last_msg_content"`
		GlobalLastMsgType    int64     `db:"global_last_msg_type"`
		GlobalLastSenderId   int64     `db:"global_last_sender_id"`
		LatestSeq            int64     `db:"latest_seq"`
	}
)

// NewUserConversationModel returns a model for the database table.
func NewUserConversationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserConversationModel {
	return &customUserConversationModel{
		defaultUserConversationModel: newUserConversationModel(conn, c, opts...),
	}
}

func (m *customUserConversationModel) GetUserConversationsByUserId(ctx context.Context, userId int64) ([]*UserConversationWithSeq, error) {
	query := fmt.Sprintf(`
		SELECT 
			uc.*, 
			c.last_msg_id as global_last_msg_id,
			c.last_msg_time as global_last_msg_time,
			c.last_msg_content as global_last_msg_content,
			c.last_msg_type as global_last_msg_type,
			c.last_sender_id as global_last_sender_id,
			c.latest_seq 
		FROM %s uc 
		INNER JOIN conversation c ON uc.conversation_id = c.conversation_id 
		WHERE uc.user_id = ? AND uc.is_deleted = 0
		ORDER BY uc.is_top DESC, c.last_msg_time DESC
	`, m.table)
	var resp []*UserConversationWithSeq
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userId)
	return resp, err
}

func (m *customUserConversationModel) UpdateNewPrivateMsg(ctx context.Context, session sqlx.Session, userId int64, peerId int64, conversationId string, lastMsg *MessageTemplate, incUnread bool) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			user_id, conversation_id, peer_id, 
			last_msg_id, last_msg_time, last_msg_content, 
			last_msg_type, last_sender_id, unread_count, is_deleted, read_sequence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0)
		ON DUPLICATE KEY UPDATE 
			last_msg_id = VALUES(last_msg_id),
			last_msg_time = VALUES(last_msg_time),
			last_msg_content = VALUES(last_msg_content),
			last_msg_type = VALUES(last_msg_type),
			last_sender_id = VALUES(last_sender_id),
			unread_count = unread_count + VALUES(unread_count),
			is_deleted = 0
	`, m.table)

	unreadVal := 0
	if incUnread {
		unreadVal = 1
	}

	_, err := session.ExecCtx(ctx, query,
		userId, conversationId, peerId,
		lastMsg.MsgId, lastMsg.CreatedAt, lastMsg.Content,
		lastMsg.MsgType, lastMsg.SenderId, unreadVal,
	)
	return err
}

func (m *customUserConversationModel) UpdateReadSequence(ctx context.Context, userId int64, conversationId string, seq int64) error {
	query := fmt.Sprintf("UPDATE %s SET read_sequence = ?, unread_count = 0 WHERE user_id = ? AND conversation_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, seq, userId, conversationId)
	return err
}
