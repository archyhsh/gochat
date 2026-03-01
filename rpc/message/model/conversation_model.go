package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationModel = (*customConversationModel)(nil)

type (
	// ConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationModel.
	ConversationModel interface {
		conversationModel
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
