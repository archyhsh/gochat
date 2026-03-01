package svc

import (
	"github.com/archyhsh/gochat/rpc/message/internal/config"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config                config.Config
	SqlConn               sqlx.SqlConn
	ConversationModel     model.ConversationModel
	MessageReadModel      model.MessageReadModel
	MessageTemplateModel  model.MessageTemplateModel
	UserConversationModel model.UserConversationModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	return &ServiceContext{
		Config:                c,
		SqlConn:               sqlConn,
		ConversationModel:     model.NewConversationModel(sqlConn, c.Cache),
		MessageReadModel:      model.NewMessageReadModel(sqlConn, c.Cache),
		MessageTemplateModel:  model.NewMessageTemplateModel(sqlConn, c.Cache),
		UserConversationModel: model.NewUserConversationModel(sqlConn, c.Cache),
	}
}
