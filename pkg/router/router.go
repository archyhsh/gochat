package router

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	UserRoutePrefix = "route:user:"
	DefaultExpiry   = 24 * time.Hour
)

type Router struct {
	rdb        *redis.Client
	serverAddr string
}

func NewRouter(rdb *redis.Client, serverAddr string) *Router {
	return &Router{
		rdb:        rdb,
		serverAddr: serverAddr,
	}
}

func (r *Router) Register(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)
	return r.rdb.Set(ctx, key, r.serverAddr, DefaultExpiry).Err()
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
	return r.rdb.Eval(ctx, script, []string{key}, r.serverAddr).Err()
}

func (r *Router) Find(ctx context.Context, userID int64) (string, error) {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)
	addr, err := r.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return addr, err
}
