package svc

import (
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/messaging"
	"github.com/archyhsh/gochat/rpc/group/internal/config"
	"github.com/archyhsh/gochat/rpc/group/model"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config            config.Config
	SqlConn           sqlx.SqlConn
	GroupModel        model.GroupModel
	GroupMemberModel  model.GroupMemberModel
	GroupRequestModel model.GroupRequestModel
	Producer          *messaging.ReliableProducer
	UserRpc           userservice.UserService
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	rawProducer, _ := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic)
	rdb := redis.MustNewRedis(c.Cache[0].RedisConf)
	failureStore := messaging.NewRedisFailureStore(rdb, "")
	producer := messaging.NewReliableProducer(rawProducer, failureStore, c.Kafka.Topic)

	return &ServiceContext{
		Config:            c,
		SqlConn:           sqlConn,
		GroupModel:        model.NewGroupModel(sqlConn, c.Cache),
		GroupMemberModel:  model.NewGroupMemberModel(sqlConn, c.Cache),
		GroupRequestModel: model.NewGroupRequestModel(sqlConn, c.Cache),
		Producer:          producer,
		UserRpc:           userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
	}
}
