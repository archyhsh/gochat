package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DB struct {
		DataSource string
	}
	Cache cache.CacheConf
	Kafka struct {
		Brokers []string
		Topics  struct {
			Message string
			Group   string
		}
		GroupID string
	}
	UserRpc  zrpc.RpcClientConf
	GroupRpc zrpc.RpcClientConf
}
