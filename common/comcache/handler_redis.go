package comcache

import (
	"context"
	"encoding/json"
	"errors"
	"shopble/common/comrunner"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisHandler struct {
	sync.RWMutex

	client  *redis.Client
	encoder IEncoder
}

func init() {
	RegisterCacheHandler("redis", func(c *Config) ICacheManager {
		return newRedisHandler(c)
	})
}

var (
	ErrNotFound = errors.New("cache: data not found")
	ErrSetValue = errors.New("cache: cannot set to value")
)

func newRedisHandler(conf *Config) (_ *RedisHandler) {
	conn := redis.NewClient(&redis.Options{
		Addr: conf.Redis.Address,
		DB:   conf.Redis.Db,
	})
	comrunner.RegisterRootCloser(func() {
		_ = conn.Close()
	})
	return &RedisHandler{
		client:  conn,
		encoder: JsonEncoder{},
	}
}

func (rc *RedisHandler) Get(ctx context.Context, key string, result any) error {
	cmd := rc.client.Get(ctx, key)
	if cmd.Err() == redis.Nil {
		return ErrNotFound
	}
	bytes, err := cmd.Bytes()
	if err != nil {
		return err
	}
	return rc.encoder.Decode(bytes, result)
}

func (rc *RedisHandler) Exists(ctx context.Context, key string) (bool, error) {
	cmd := rc.client.Exists(ctx, key)
	result, err := cmd.Result()
	return result > 0, err
}

func (rc *RedisHandler) Set(
	ctx context.Context,
	key string, value any, timeout time.Duration,
) error {
	return rc.SetEx(ctx, key, value, Options{Timeout: timeout})
}

func (rc *RedisHandler) SetEx(
	ctx context.Context,
	key string, value any, options Options,
) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return rc.client.Set(ctx, key, valueBytes, options.Timeout).Err()
}

func (rc *RedisHandler) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}
