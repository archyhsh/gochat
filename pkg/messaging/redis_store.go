package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

const DefaultDLQKey = "queue:kafka:failed"

type RedisFailureStore struct {
	rdb *redis.Redis
	key string
}

type FailedMessage struct {
	Topic      string `json:"topic"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	Error      string `json:"error"`
	FailedAt   int64  `json:"failed_at"`
	RetryCount int    `json:"retry_count"`
}

func NewRedisFailureStore(rdb *redis.Redis, key string) *RedisFailureStore {
	if key == "" {
		key = DefaultDLQKey
	}
	return &RedisFailureStore{
		rdb: rdb,
		key: key,
	}
}

func (s *RedisFailureStore) Save(ctx context.Context, topic string, key, value []byte, err error) error {
	msg := FailedMessage{
		Topic:    topic,
		Key:      string(key),
		Value:    string(value),
		Error:    err.Error(),
		FailedAt: time.Now().Unix(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = s.rdb.LpushCtx(ctx, s.key, string(data))
	return err
}
