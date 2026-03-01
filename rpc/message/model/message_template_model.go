package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MessageTemplateModel = (*customMessageTemplateModel)(nil)

type (
	// MessageTemplateModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMessageTemplateModel.
	MessageTemplateModel interface {
		messageTemplateModel
		FindOneByTableAndMessageId(ctx context.Context, table string, messageId string) (*MessageTemplate, error)
		FindPageByTable(ctx context.Context, table string, conversationId string, limit int32, offset int32) ([]*MessageTemplate, error)
		CountByTable(ctx context.Context, table string, conversationId string) (int64, error)
	}

	customMessageTemplateModel struct {
		*defaultMessageTemplateModel
	}
)

// NewMessageTemplateModel returns a model for the database table.
func NewMessageTemplateModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MessageTemplateModel {
	return &customMessageTemplateModel{
		defaultMessageTemplateModel: newMessageTemplateModel(conn, c, opts...),
	}
}

func (m *customMessageTemplateModel) FindOneByTableAndMessageId(ctx context.Context, table string, messageId string) (*MessageTemplate, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE message_id = ?", messageTemplateRows, table)
	var resp *MessageTemplate
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, messageId)
	return resp, err
}

func (m *customMessageTemplateModel) FindPageByTable(ctx context.Context, table string, conversationId string, limit int32, offset int32) ([]*MessageTemplate, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE conversation_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?", messageTemplateRows, table)
	var resp []*MessageTemplate
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, conversationId, limit, offset)
	return resp, err
}

func (m *customMessageTemplateModel) CountByTable(ctx context.Context, table string, conversationId string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE conversation_id = ?", table)
	var count int64
	err := m.QueryRowNoCacheCtx(ctx, &count, query, conversationId)
	return count, err
}
