package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ FriendApplyModel = (*customFriendApplyModel)(nil)

type (
	// FriendApplyModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFriendApplyModel.
	FriendApplyModel interface {
		friendApplyModel
		FindPendingApplyByFromAndTo(ctx context.Context, fromId, toId int64) (*FriendApply, error)
		FindApplyListByToUserId(ctx context.Context, toUserId int64) ([]*FriendApply, error)
	}

	customFriendApplyModel struct {
		*defaultFriendApplyModel
	}
)

// NewFriendApplyModel returns a model for the database table.
func NewFriendApplyModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) FriendApplyModel {
	return &customFriendApplyModel{
		defaultFriendApplyModel: newFriendApplyModel(conn, c, opts...),
	}
}

func (m *customFriendApplyModel) FindPendingApplyByFromAndTo(ctx context.Context, fromId, toId int64) (*FriendApply, error) {
	query := fmt.Sprintf("select %s from %s where from_user_id = ? and to_user_id = ? and status = 0 limit 1", friendApplyRows, m.table)
	var resp FriendApply
	err := m.QueryRowNoCacheCtx(ctx, &resp, query, fromId, toId)
	switch err {
	case nil:
		return &resp, nil
	case sqlx.ErrNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (m *customFriendApplyModel) FindApplyListByToUserId(ctx context.Context, toUserId int64) ([]*FriendApply, error) {
	query := fmt.Sprintf("select %s from %s where to_user_id = ? and status = 0", friendApplyRows, m.table)
	var resp []*FriendApply
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, toUserId)
	return resp, err
}
