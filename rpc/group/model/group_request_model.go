package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GroupRequestModel = (*customGroupRequestModel)(nil)

type (
	// GroupRequestModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGroupRequestModel.
	GroupRequestModel interface {
		groupRequestModel
		FindPendingByGroupId(ctx context.Context, groupId int64) ([]*GroupRequest, error)
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

func (m *customGroupRequestModel) FindPendingByGroupId(ctx context.Context, groupId int64) ([]*GroupRequest, error) {
	query := fmt.Sprintf("select %s from %s where `group_id` = ? and `status` = 0 order by created_at desc", groupRequestRows, m.table)
	var resp []*GroupRequest
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, groupId)
	return resp, err
}
