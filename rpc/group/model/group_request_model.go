package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GroupRequestModel = (*customGroupRequestModel)(nil)

type (
	// GroupRequestModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGroupRequestModel.
	GroupRequestModel interface {
		groupRequestModel
	}

	customGroupRequestModel struct {
		*defaultGroupRequestModel
	}
)

// NewGroupRequestModel returns a model for the database table.
func NewGroupRequestModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) GroupRequestModel {
	return &customGroupRequestModel{
		defaultGroupRequestModel: newGroupRequestModel(conn, c, opts...),
	}
}
