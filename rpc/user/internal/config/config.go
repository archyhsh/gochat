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
	JWT   struct {
		AccessSecret string
		AccessExpire int64
	}
	Kafka struct {
		Brokers []string
		Topic   string
	}
}
