package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserConversationModel = (*customUserConversationModel)(nil)

type (
	// UserConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserConversationModel.
	UserConversationModel interface {
		userConversationModel
		FindUserConversationsByUserIdAndConversationId(userId int64, conversationId string) (*UserConversation, error)
		GetUserConversationsByUserId(userId int64) ([]*UserConversation, error)
	}

	customUserConversationModel struct {
		*defaultUserConversationModel
	}
)

// NewUserConversationModel returns a model for the database table.
func NewUserConversationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserConversationModel {
	return &customUserConversationModel{
		defaultUserConversationModel: newUserConversationModel(conn, c, opts...),
	}
}

func (m *customUserConversationModel) FindUserConversationsByUserIdAndConversationId(userId int64, conversationId string) (*UserConversation, error) {
	query := "SELECT * FROM " + m.table + " WHERE user_id = ? AND conversation_id = ?"
	var resp *UserConversation
	err := m.QueryRowsNoCache(&resp, query, userId, conversationId)
	return resp, err
}

func (m *customUserConversationModel) GetUserConversationsByUserId(userId int64) ([]*UserConversation, error) {
	query := "SELECT * FROM " + m.table + " WHERE user_id = ?"
	var resp []*UserConversation
	err := m.QueryRowsNoCache(&resp, query, userId)
	return resp, err
}
