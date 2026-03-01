package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MessageReadModel = (*customMessageReadModel)(nil)

type (
	// MessageReadModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMessageReadModel.
	MessageReadModel interface {
		messageReadModel
	}

	customMessageReadModel struct {
		*defaultMessageReadModel
	}
)

// NewMessageReadModel returns a model for the database table.
func NewMessageReadModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MessageReadModel {
	return &customMessageReadModel{
		defaultMessageReadModel: newMessageReadModel(conn, c, opts...),
	}
}
