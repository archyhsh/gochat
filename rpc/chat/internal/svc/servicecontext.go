package svc

import (
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/archyhsh/gochat/pkg/manager"
	"github.com/archyhsh/gochat/pkg/router"
	"github.com/archyhsh/gochat/rpc/chat/internal/config"
	"github.com/archyhsh/gochat/rpc/group/groupservice"
	"github.com/archyhsh/gochat/rpc/relation/relationservice"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config      config.Config
	RelationRpc relationservice.RelationService
	GroupRpc    groupservice.GroupService
	Producer    *kafka.Producer
	Manager     *manager.Manager
	Router      *router.Router
	Redis       *redis.Client
}

func NewServiceContext(c config.Config, m *manager.Manager) *ServiceContext {
	rdb := redis.NewClient(&redis.Options{
		Addr:     c.Redis.Host,
		Password: c.Redis.Pass,
	})

	return &ServiceContext{
		Config:      c,
		RelationRpc: relationservice.NewRelationService(zrpc.MustNewClient(c.RelationRpc)),
		GroupRpc:    groupservice.NewGroupService(zrpc.MustNewClient(c.GroupRpc)),
		Producer:    c.Producer,
		Manager:     m,
		Redis:       rdb,
		Router:      router.NewRouter(rdb, c.ListenOn),
	}
}
