package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		SearchUsersByName(ctx context.Context, name string) ([]*User, error)
		SearchUsersByIds(ctx context.Context, ids []int64) ([]*User, error)
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

func (m *customUserModel) SearchUsersByName(ctx context.Context, name string) ([]*User, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `nickname` LIKE ? AND `status` = 1 LIMIT 50", userRows, m.table)
	var resp []*User
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, name+"%")
	return resp, err
}

func (m *customUserModel) SearchUsersByIds(ctx context.Context, ids []int64) ([]*User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE `id` IN (%s) AND `status` = 1",
		userRows, m.table, strings.Join(placeholders, ","))

	var resp []*User
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, args...)
	return resp, err
}
