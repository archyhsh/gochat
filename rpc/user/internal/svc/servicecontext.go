package svc

import (
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/rpc/user/internal/config"
	"github.com/archyhsh/gochat/rpc/user/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	UserModel       model.UserModel
	UserDeviceModel model.UserDeviceModel
	JwtManager      *auth.JWTManager
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	return &ServiceContext{
		Config:          c,
		UserModel:       model.NewUserModel(sqlConn, c.Cache),
		UserDeviceModel: model.NewUserDeviceModel(sqlConn, c.Cache),
		JwtManager:      auth.NewJWTManager(c.Auth.AccessSecret, int(c.Auth.AccessExpire)/3600),
	}
}
