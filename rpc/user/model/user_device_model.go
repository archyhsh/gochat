package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserDeviceModel = (*customUserDeviceModel)(nil)

type (
	// UserDeviceModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserDeviceModel.
	UserDeviceModel interface {
		userDeviceModel
	}

	customUserDeviceModel struct {
		*defaultUserDeviceModel
	}
)

// NewUserDeviceModel returns a model for the database table.
func NewUserDeviceModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserDeviceModel {
	return &customUserDeviceModel{
		defaultUserDeviceModel: newUserDeviceModel(conn, c, opts...),
	}
}
