package svc

import (
	"github.com/archyhsh/gochat/rpc/message/internal/config"
	"github.com/archyhsh/gochat/rpc/message/model"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                config.Config
	SqlConn               sqlx.SqlConn
	ConversationModel     model.ConversationModel
	MessageReadModel      model.MessageReadModel
	MessageTemplateModel  model.MessageTemplateModel
	UserConversationModel model.UserConversationModel
	UserRpc               userservice.UserService
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
		UserRpc:               userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
	}
}
