package model

import (
	"context"
	"database/sql"
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
		UpdateSeq(ctx context.Context, session sqlx.Session, conversationId string, lastMsg *MessageTemplate) (int64, error)
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

func (m *customConversationModel) UpdateSeq(ctx context.Context, session sqlx.Session, conversationId string, lastMsg *MessageTemplate) (int64, error) {
	query := fmt.Sprintf("UPDATE %s SET `latest_seq` = LAST_INSERT_ID(`latest_seq` + 1), `last_msg_id` = ?, `last_msg_time` = ?, `last_msg_content` = ?, `last_msg_type` = ?, `last_sender_id` = ? WHERE `conversation_id` = ?", m.table)
	res, err := session.ExecCtx(ctx, query,
		lastMsg.MsgId,
		lastMsg.CreatedAt,
		lastMsg.Content,
		lastMsg.MsgType,
		lastMsg.SenderId,
		conversationId,
	)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected == 0 {
		return 0, sql.ErrNoRows
	}

	var newSeq int64
	err = session.QueryRowCtx(ctx, &newSeq, "SELECT LAST_INSERT_ID()")
	if err != nil {
		return 0, err
	}

	return newSeq, nil
}
