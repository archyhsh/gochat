package svc

import (
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/messaging"
	"github.com/archyhsh/gochat/rpc/user/internal/config"
	"github.com/archyhsh/gochat/rpc/user/model"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config          config.Config
	UserModel       model.UserModel
	UserDeviceModel model.UserDeviceModel
	JwtManager      *auth.JWTManager
	Producer        *messaging.ReliableProducer
}

func NewServiceContext(c config.Config) *ServiceContext {
	sqlConn := sqlx.NewMysql(c.DB.DataSource)
	rawProducer, _ := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic)
	rdb := redis.MustNewRedis(c.Cache[0].RedisConf)
	failureStore := messaging.NewRedisFailureStore(rdb, "")
	producer := messaging.NewReliableProducer(rawProducer, failureStore, c.Kafka.Topic)

	return &ServiceContext{
		Config:          c,
		UserModel:       model.NewUserModel(sqlConn, c.Cache),
		UserDeviceModel: model.NewUserDeviceModel(sqlConn, c.Cache),
		JwtManager:      auth.NewJWTManager(c.JWT.AccessSecret, int(c.JWT.AccessExpire)/3600),
		Producer:        producer,
	}
}
