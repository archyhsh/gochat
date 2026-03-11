// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"fmt"
	"sync"

	"github.com/archyhsh/gochat/api/internal/config"
	"github.com/archyhsh/gochat/api/internal/middleware"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/router"
	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/messageservice"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"

	"github.com/IBM/sarama"
)

type ServiceContext struct {
	Config         config.Config
	AuthMiddleware rest.Middleware
	JwtManager     *auth.JWTManager
	UserRpc        userservice.UserService
	GroupRpc       groupservice.GroupService
	MessageRpc     messageservice.MessageService
	RelationRpc    relationservice.RelationService
	KafkaProducer  sarama.SyncProducer
	Router         *router.Router
	Conns          sync.Map
}

func NewServiceContext(c config.Config) *ServiceContext {
	jwtManager := auth.NewJWTManager(c.JWT.JwtSecret, c.JWT.ExpireHours)
	_ = snowflake.Init(1)
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 5
	producerConfig.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(c.Kafka.Brokers, producerConfig)
	if err != nil {
		panic("Failed to initialize Kafka producer: " + err.Error())
	}
	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: c.Redis[0].Host,
		Type: c.Redis[0].Type,
		Pass: c.Redis[0].Pass,
	})

	serverAddr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	rt := router.NewRouter(rdb, serverAddr)

	return &ServiceContext{
		Config:         c,
		AuthMiddleware: middleware.NewAuthMiddleware(jwtManager).Handle,
		JwtManager:     jwtManager,
		UserRpc:        userservice.NewUserService(zrpc.MustNewClient(c.UserRpc)),
		GroupRpc:       groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc)),
		MessageRpc:     messageservice.NewMessageService(zrpc.MustNewClient(c.MessageRpc)),
		RelationRpc:    relationservice.NewRelationService(zrpc.MustNewClient(c.RelationRpc)),
		KafkaProducer:  producer,
		Router:         rt,
	}
}
