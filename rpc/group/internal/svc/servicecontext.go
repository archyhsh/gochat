package svc

import (
	"github.com/archyhsh/gochat/rpc/group/internal/config"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config            config.Config
	SqlConn           sqlx.SqlConn
	GroupModel        model.GroupModel
	GroupMemberModel  model.GroupMemberModel
	GroupRequestModel model.GroupRequestModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	return &ServiceContext{
		Config:            c,
		SqlConn:           sqlConn,
		GroupModel:        model.NewGroupModel(sqlConn, c.Cache),
		GroupMemberModel:  model.NewGroupMemberModel(sqlConn, c.Cache),
		GroupRequestModel: model.NewGroupRequestModel(sqlConn, c.Cache),
	}
}
