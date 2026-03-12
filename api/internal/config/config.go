package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	UserRpc     zrpc.RpcClientConf
	MessageRpc  zrpc.RpcClientConf
	RelationRpc zrpc.RpcClientConf
	GroupRpc    zrpc.RpcClientConf
	JWT         struct {
		JwtSecret   string
		ExpireHours int
	}
	Kafka struct {
		Brokers []string
		Topic   string
	}
	Redis cache.CacheConf
	Etcd  struct {
		Hosts []string
		Key   string
	}
}
