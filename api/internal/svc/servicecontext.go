// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"fmt"
	"sync"

	"github.com/archyhsh/gochat/api/internal/config"
	"github.com/archyhsh/gochat/api/internal/middleware"
	"github.com/archyhsh/gochat/pkg/auth"
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/messaging"
	"github.com/archyhsh/gochat/pkg/router"
	"github.com/archyhsh/gochat/pkg/snowflake"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/message/messageservice"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/archyhsh/gochat/rpc/user/userservice"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config         config.Config
	AuthMiddleware rest.Middleware
	JwtManager     *auth.JWTManager
	UserRpc        userservice.UserService
	GroupRpc       groupservice.GroupService
	MessageRpc     messageservice.MessageService
	RelationRpc    relationservice.RelationService
	KafkaProducer  *messaging.ReliableProducer
	Router         *router.Router
	Conns          sync.Map
}

func NewServiceContext(c config.Config) *ServiceContext {
	jwtManager := auth.NewJWTManager(c.JWT.JwtSecret, c.JWT.ExpireHours)
	_ = snowflake.Init(1)

	rawProducer, err := kafka.NewProducer(c.Kafka.Brokers, c.Kafka.Topic)
	if err != nil {
		panic("Failed to initialize Kafka producer: " + err.Error())
	}

	// Better Redis compatibility: supports cluster/sentinel if configured in YAML
	rdb := redis.MustNewRedis(c.Redis[0].RedisConf)

	failureStore := messaging.NewRedisFailureStore(rdb, "")
	producer := messaging.NewReliableProducer(rawProducer, failureStore, c.Kafka.Topic)

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
