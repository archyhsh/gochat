package model

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-sql-driver/mysql"
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
		FindPageByTable(ctx context.Context, table string, conversationId string, lastSeq int64, limit int32) ([]*MessageTemplate, error)
		FindNewerBySeq(ctx context.Context, table string, conversationId string, lastSeq int64, limit int32) ([]*MessageTemplate, error)
		CountByTable(ctx context.Context, table string, conversationId string) (int64, error)
		CheckTableExist(ctx context.Context, table string) error
		InsertToTable(ctx context.Context, session sqlx.Session, table string, data *MessageTemplate) error
	}

	customMessageTemplateModel struct {
		*defaultMessageTemplateModel
		conn       sqlx.SqlConn
		tablecache sync.Map
	}
)

// NewMessageTemplateModel returns a model for the database table.
func NewMessageTemplateModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MessageTemplateModel {
	return &customMessageTemplateModel{
		defaultMessageTemplateModel: newMessageTemplateModel(conn, c, opts...),
		conn:                        conn,
	}
}

func (m *customMessageTemplateModel) FindOneByTableAndMessageId(ctx context.Context, table string, messageId string) (*MessageTemplate, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE msg_id = ?", messageTemplateRows, table)
	var resp MessageTemplate
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, messageId)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *customMessageTemplateModel) FindPageByTable(ctx context.Context, table string, conversationId string, lastSeq int64, limit int32) ([]*MessageTemplate, error) {
	var query string
	var args []interface{}
	if lastSeq > 0 {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE conversation_id = ? AND sequence_id < ? ORDER BY sequence_id DESC LIMIT ?", messageTemplateRows, table)
		args = append(args, conversationId, lastSeq, limit)
	} else {
		query = fmt.Sprintf("SELECT %s FROM %s WHERE conversation_id = ? ORDER BY sequence_id DESC LIMIT ?", messageTemplateRows, table)
		args = append(args, conversationId, limit)
	}

	var resp []*MessageTemplate
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	return resp, err
}

func (m *customMessageTemplateModel) FindNewerBySeq(ctx context.Context, table string, conversationId string, lastSeq int64, limit int32) ([]*MessageTemplate, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE conversation_id = ? AND sequence_id > ? ORDER BY sequence_id ASC LIMIT ?", messageTemplateRows, table)
	var resp []*MessageTemplate
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, conversationId, lastSeq, limit)
	return resp, err
}

func (m *customMessageTemplateModel) CountByTable(ctx context.Context, table string, conversationId string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE conversation_id = ?", table)
	var count int64
	err := m.QueryRowNoCacheCtx(ctx, &count, query, conversationId)
	return count, err
}

func (m *customMessageTemplateModel) CheckTableExist(ctx context.Context, table string) error {
	if _, ok := m.tablecache.Load(table); ok {
		return nil
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` LIKE `message_template` ", table)
	_, err := m.conn.ExecCtx(ctx, query)
	if err != nil {
		return err
	}
	m.tablecache.Store(table, true)
	return nil
}

func (m *customMessageTemplateModel) InsertToTable(ctx context.Context, session sqlx.Session, table string, data *MessageTemplate) error {
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", table, messageTemplateRowsExpectAutoSet)
	_, err := session.ExecCtx(ctx, query, data.MsgId, data.ConversationId, data.SenderId, data.ReceiverId, data.GroupId, data.SequenceId, data.MsgType, data.Content, data.Status)
	if err != nil {
		// 检查是否为唯一键冲突错误 (MySQL Error 1062) 我们希望支持幂等处理重复信息
		if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
			return nil // 幂等处理：视为成功，以便提交 Kafka Offset
		}
		return err
	}
	return nil
}
