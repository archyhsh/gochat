package router

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	UserRoutePrefix = "route:user:"
	DefaultExpiry   = 30 * time.Minute // Reduced TTL for better accuracy with heartbeats
)

type Router struct {
	rdb        *redis.Redis
	serverAddr string
}

func NewRouter(rdb *redis.Redis, serverAddr string) *Router {
	return &Router{
		rdb:        rdb,
		serverAddr: serverAddr,
	}
}

func (r *Router) Register(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)
	// Store the internal gRPC/HTTP address of THIS gateway node
	return r.rdb.SetexCtx(ctx, key, r.serverAddr, int(DefaultExpiry.Seconds()))
}

func (r *Router) Unregister(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	_, err := r.rdb.EvalCtx(ctx, script, []string{key}, r.serverAddr)
	return err
}

func (r *Router) Find(ctx context.Context, userID int64) (string, error) {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)
	addr, err := r.rdb.GetCtx(ctx, key)
	if err != nil {
		return "", err
	}
	return addr, nil
}
