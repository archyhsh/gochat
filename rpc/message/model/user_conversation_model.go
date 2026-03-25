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
		SearchUserConversationsByUserId(ctx context.Context, userId int64, keyword string) ([]*UserConversationWithSeq, error)
		UpdateNewPrivateMsg(ctx context.Context, session sqlx.Session, userId int64, peerId int64, peerName string, peerAvatar string, conversationId string, lastMsg *MessageTemplate, incUnread bool) error
		UpdateReadSequence(ctx context.Context, userId int64, conversationId string, seq int64) error
		UpdateVersion(ctx context.Context, userId int64, conversationId string, version int64) error
		Restore(ctx context.Context, userId int64, conversationId string) error
		Hide(ctx context.Context, userId int64, conversationId string) error
		GetUsersByPeerId(ctx context.Context, peerId int64) ([]int64, error)
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

func (m *customUserConversationModel) SearchUserConversationsByUserId(ctx context.Context, userId int64, keyword string) ([]*UserConversationWithSeq, error) {
	// For search, we include deleted ones.
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
		WHERE uc.user_id = ? AND uc.peer_name LIKE ?
		ORDER BY uc.is_top DESC, c.last_msg_time DESC
	`, m.table)
	var resp []*UserConversationWithSeq
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userId, "%"+keyword+"%")
	return resp, err
}

func (m *customUserConversationModel) UpdateNewPrivateMsg(ctx context.Context, session sqlx.Session, userId int64, peerId int64, peerName string, peerAvatar string, conversationId string, lastMsg *MessageTemplate, incUnread bool) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			user_id, conversation_id, peer_id, peer_name, peer_avatar, 
			last_msg_id, last_msg_time, last_msg_content, 
			last_msg_type, last_sender_id, unread_count, is_deleted, read_sequence, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, 0)
		ON DUPLICATE KEY UPDATE 
			peer_name = VALUES(peer_name),
			peer_avatar = VALUES(peer_avatar),
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
		userId, conversationId, peerId, peerName, peerAvatar,
		lastMsg.MsgId, lastMsg.CreatedAt, lastMsg.Content,
		lastMsg.MsgType, lastMsg.SenderId, unreadVal,
	)
	if err != nil {
		return err
	}

	// Manual cache invalidation since we used direct SQL Exec
	cacheKey := fmt.Sprintf("cache:userConversation:userId:conversationId:%d:%s", userId, conversationId)
	_ = m.DelCacheCtx(ctx, cacheKey)

	return nil
}

func (m *customUserConversationModel) UpdateReadSequence(ctx context.Context, userId int64, conversationId string, seq int64) error {
	query := fmt.Sprintf("UPDATE %s SET read_sequence = ?, unread_count = 0 WHERE user_id = ? AND conversation_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, seq, userId, conversationId)
	if err == nil {
		cacheKey := fmt.Sprintf("cache:userConversation:userId:conversationId:%d:%s", userId, conversationId)
		_ = m.DelCacheCtx(ctx, cacheKey)
	}
	return err
}

func (m *customUserConversationModel) UpdateVersion(ctx context.Context, userId int64, conversationId string, version int64) error {
	query := fmt.Sprintf("UPDATE %s SET version = ? WHERE user_id = ? AND conversation_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, version, userId, conversationId)
	if err == nil {
		cacheKey := fmt.Sprintf("cache:userConversation:userId:conversationId:%d:%s", userId, conversationId)
		_ = m.DelCacheCtx(ctx, cacheKey)
	}
	return err
}

func (m *customUserConversationModel) Restore(ctx context.Context, userId int64, conversationId string) error {
	query := fmt.Sprintf("UPDATE %s SET is_deleted = 0, version = ? WHERE user_id = ? AND conversation_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, time.Now().UnixNano(), userId, conversationId)
	if err == nil {
		cacheKey := fmt.Sprintf("cache:userConversation:userId:conversationId:%d:%s", userId, conversationId)
		_ = m.DelCacheCtx(ctx, cacheKey)
	}
	return err
}

func (m *customUserConversationModel) Hide(ctx context.Context, userId int64, conversationId string) error {
	query := fmt.Sprintf("UPDATE %s SET is_deleted = 1, version = ? WHERE user_id = ? AND conversation_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, time.Now().UnixNano(), userId, conversationId)
	if err == nil {
		cacheKey := fmt.Sprintf("cache:userConversation:userId:conversationId:%d:%s", userId, conversationId)
		_ = m.DelCacheCtx(ctx, cacheKey)
	}
	return err
}

func (m *customUserConversationModel) GetUsersByPeerId(ctx context.Context, peerId int64) ([]int64, error) {
	query := fmt.Sprintf("SELECT user_id FROM %s WHERE peer_id = ? AND is_deleted = 0", m.table)
	var resp []int64
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, peerId)
	return resp, err
}
