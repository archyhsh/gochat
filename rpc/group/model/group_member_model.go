package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ GroupMemberModel = (*customGroupMemberModel)(nil)

type (
	// GroupMemberModel is an interface to be customized, add more methods here,
	// and implement the added methods in customGroupMemberModel.
	GroupMemberModel interface {
		groupMemberModel
		FindMembersByGroupId(ctx context.Context, groupId int64) ([]*GroupMember, error)
		FindMemberByGroupIdAndUserId(ctx context.Context, groupId int64, userId int64) (*GroupMember, error)
		UpdateNickname(ctx context.Context, groupId int64, userId int64, nickname string) error
	}

	customGroupMemberModel struct {
		*defaultGroupMemberModel
	}
)

// NewGroupMemberModel returns a model for the database table.
func NewGroupMemberModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) GroupMemberModel {
	return &customGroupMemberModel{
		defaultGroupMemberModel: newGroupMemberModel(conn, c, opts...),
	}
}

func (m *customGroupMemberModel) FindMembersByGroupId(ctx context.Context, groupId int64) ([]*GroupMember, error) {
	var members []*GroupMember
	err := m.QueryRowsNoCacheCtx(ctx, &members, "select * from `group_member` where group_id = ?", groupId)
	return members, err
}

func (m *customGroupMemberModel) FindMemberByGroupIdAndUserId(ctx context.Context, groupId int64, userId int64) (*GroupMember, error) {
	var member GroupMember
	err := m.QueryRowNoCacheCtx(ctx, &member, "select * from `group_member` where group_id = ? and user_id = ?", groupId, userId)
	return &member, err
}

func (m *customGroupMemberModel) UpdateNickname(ctx context.Context, groupId int64, userId int64, nickname string) error {
	query := "update `group_member` set nickname = ? where group_id = ? and user_id = ?"
	_, err := m.ExecNoCacheCtx(ctx, query, nickname, groupId, userId)
	return err
}
