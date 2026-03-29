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

	var lastErr error
	for i := 0; i < 3; i++ {
		err := r.rdb.SetexCtx(ctx, key, r.serverAddr, int(DefaultExpiry.Seconds()))
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("failed to register route after retries: %v", lastErr)
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
	var lastErr error
	for i := 0; i < 3; i++ {
		_, err := r.rdb.EvalCtx(ctx, script, []string{key}, r.serverAddr)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return lastErr
}

func (r *Router) Find(ctx context.Context, userID int64) (string, error) {
	key := fmt.Sprintf("%s%d", UserRoutePrefix, userID)

	var lastErr error
	for i := 0; i < 2; i++ {
		addr, err := r.rdb.GetCtx(ctx, key)
		if err == nil {
			return addr, nil
		}
		lastErr = err
		time.Sleep(30 * time.Millisecond)
	}
	return "", lastErr
}

func (r *Router) BatchFind(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	if len(userIDs) == 0 {
		return make(map[int64]string), nil
	}

	keys := make([]string, len(userIDs))
	for i, uid := range userIDs {
		keys[i] = fmt.Sprintf("%s%d", UserRoutePrefix, uid)
	}

	var addrs []string
	var lastErr error
	for i := 0; i < 2; i++ {
		vals, err := r.rdb.MgetCtx(ctx, keys...)
		if err == nil {
			addrs = vals
			break
		}
		lastErr = err
		time.Sleep(30 * time.Millisecond)
	}

	if lastErr != nil && len(addrs) == 0 {
		return nil, lastErr
	}

	res := make(map[int64]string)
	for i, addr := range addrs {
		if addr != "" {
			res[userIDs[i]] = addr
		}
	}
	return res, nil
}
