package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationModel = (*customConversationModel)(nil)

type (
	// ConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationModel.
	ConversationModel interface {
		conversationModel
		UpdateSeq(ctx context.Context, session sqlx.Session, conversationId string, convType int32, targetId int64, lastMsg *MessageTemplate) (int64, error)
	}

	customConversationModel struct {
		*defaultConversationModel
	}
)

// NewConversationModel returns a model for the database table.
func NewConversationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) ConversationModel {
	return &customConversationModel{
		defaultConversationModel: newConversationModel(conn, c, opts...),
	}
}

func (m *customConversationModel) UpdateSeq(ctx context.Context, session sqlx.Session, conversationId string, convType int32, targetId int64, lastMsg *MessageTemplate) (int64, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			conversation_id, type, target_id, 
			last_msg_id, last_msg_time, last_msg_content, 
			last_msg_type, last_sender_id, latest_seq
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, LAST_INSERT_ID(1))
		ON DUPLICATE KEY UPDATE 
			latest_seq = LAST_INSERT_ID(latest_seq + 1),
			last_msg_id = VALUES(last_msg_id),
			last_msg_time = VALUES(last_msg_time),
			last_msg_content = VALUES(last_msg_content),
			last_msg_type = VALUES(last_msg_type),
			last_sender_id = VALUES(last_sender_id)
	`, m.table)

	_, err := session.ExecCtx(ctx, query,
		conversationId, convType, targetId,
		lastMsg.MsgId, lastMsg.CreatedAt, lastMsg.Content,
		lastMsg.MsgType, lastMsg.SenderId,
	)
	if err != nil {
		return 0, err
	}
	var newSeq int64
	err = session.QueryRowCtx(ctx, &newSeq, "SELECT LAST_INSERT_ID()")
	if err != nil {
		return 0, err
	}
	return newSeq, nil
}
