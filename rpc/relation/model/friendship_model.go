package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ FriendshipModel = (*customFriendshipModel)(nil)

type (
	// FriendshipModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFriendshipModel.
	FriendshipModel interface {
		friendshipModel
		FindNormalFriendListByUserId(ctx context.Context, toUserId int64) ([]*Friendship, error)
		InsertFriendshipByUserIdFriendId(ctx context.Context, userId, friendId int64) error
		DeleteFriendshipByUserIdFriendId(ctx context.Context, userId, friendId int64) error
		UpdateRemarkWithVersion(ctx context.Context, userId, friendId int64, remark string, version int64) error
	}

	customFriendshipModel struct {
		*defaultFriendshipModel
	}
)

// NewFriendshipModel returns a model for the database table.
func NewFriendshipModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) FriendshipModel {
	return &customFriendshipModel{
		defaultFriendshipModel: newFriendshipModel(conn, c, opts...),
	}
}

func (m *customFriendshipModel) FindNormalFriendListByUserId(ctx context.Context, toUserId int64) ([]*Friendship, error) {
	query := fmt.Sprintf("select %s from %s where user_id = ? and status = 0", friendshipRows, m.table)
	var resp []*Friendship
	err := m.QueryRowsNoCacheCtx(ctx, &resp, query, toUserId)
	return resp, err
}

func (m *customFriendshipModel) InsertFriendshipByUserIdFriendId(ctx context.Context, userId, friendId int64) error {
	return m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		query := fmt.Sprintf("insert into %s (%s) values (?, ?, ?, ?, ?)", m.table, friendshipRowsExpectAutoSet)
		_, err := session.ExecCtx(ctx, query, userId, friendId, "", 0, 0)
		if err != nil {
			return status.Error(codes.Internal, "failed to insert friendship")
		}
		_, err = session.ExecCtx(ctx, query, friendId, userId, "", 0, 0)
		if err != nil {
			return status.Error(codes.Internal, "failed to insert friendship")
		}
		return nil
	})
}

func (m *customFriendshipModel) DeleteFriendshipByUserIdFriendId(ctx context.Context, userId, friendId int64) error {
	return m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		query := fmt.Sprintf("delete from %s where (user_id = ? and friend_id = ?) or (user_id = ? and friend_id = ?)", m.table)
		_, err := session.ExecCtx(ctx, query, userId, friendId, friendId, userId)
		return err
	})
}

func (m *customFriendshipModel) UpdateRemarkWithVersion(ctx context.Context, userId, friendId int64, remark string, version int64) error {
	query := fmt.Sprintf("update %s set remark = ?, version = ? where user_id = ? and friend_id = ?", m.table)
	_, err := m.ExecNoCacheCtx(ctx, query, remark, version, userId, friendId)
	return err
}
