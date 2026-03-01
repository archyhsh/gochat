package config

import (
	"github.com/archyhsh/gochat/pkg/kafka"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DB struct {
		DataSource string
	}
	Cache    cache.CacheConf
	Producer *kafka.Producer
}
