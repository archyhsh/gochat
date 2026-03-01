package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GroupModel = (*customGroupModel)(nil)

type (
	// GroupModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGroupModel.
	GroupModel interface {
		groupModel
		FindGroupsByUserId(ctx context.Context, userId int64) ([]*Group, error)
		FindValidGroupsByGroupId(ctx context.Context, groupId int64) (*Group, error)
		SearchGroupsByName(ctx context.Context, name string) ([]*Group, error)
	}

	customGroupModel struct {
		*defaultGroupModel
	}
)

// NewGroupModel returns a model for the database table.
func NewGroupModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) GroupModel {
	return &customGroupModel{
		defaultGroupModel: newGroupModel(conn, c, opts...),
	}
}

func (m *customGroupModel) FindGroupsByUserId(ctx context.Context, userId int64) ([]*Group, error) {
	query := fmt.Sprintf("SELECT g.* FROM %s g INNER JOIN `group_member` gm ON g.id = gm.group_id WHERE gm.user_id = ? AND g.status = 1", m.table)
	var resp []*Group
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, userId)
	return resp, err
}

func (m *customGroupModel) FindValidGroupsByGroupId(ctx context.Context, groupId int64) (*Group, error) {
	query := fmt.Sprintf("SELECT g.* FROM %s g WHERE g.id = ? AND g.status = 1", m.table)
	var resp *Group
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, groupId)
	return resp, err
}

func (m *customGroupModel) SearchGroupsByName(ctx context.Context, name string) ([]*Group, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `name` LIKE ? AND `status` = 1 LIMIT 50", groupRows, m.table)
	var resp []*Group
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, name+"%")
	return resp, err
}
