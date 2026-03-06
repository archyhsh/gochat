package svc

import (
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/rpc/relation/internal/config"
	"github.com/archyhsh/gochat/rpc/relation/model"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config           config.Config
	FriendshipModel  model.FriendshipModel
	FriendApplyModel model.FriendApplyModel
	UserRpc          userservice.UserService
	SqlConn          sqlx.SqlConn
	Producer         *kafka.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	p, _ := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic)
	return &ServiceContext{
		Config:           c,
		FriendshipModel:  model.NewFriendshipModel(sqlConn, c.Cache),
		FriendApplyModel: model.NewFriendApplyModel(sqlConn, c.Cache),
		SqlConn:          sqlConn,
		UserRpc:          userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		Producer:         p,
	}
}
