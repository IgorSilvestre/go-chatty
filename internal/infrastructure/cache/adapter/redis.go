package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	redis "github.com/redis/go-redis/v9"

	"go-chatty/internal/infrastructure/cache/port"
)

// RedisCache is an adapter that satisfies the port.Cache interface using Redis.
// It wraps a go-redis v9 Client.
type RedisCache struct {
	client *redis.Client
}

// NewRedisAdapter constructs a RedisCache using the REDIS_URL environment variable.
func NewRedisAdapter() (*RedisCache, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return nil, errors.New("redis: REDIS_URL environment variable is not set")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}
	c := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return &RedisCache{client: c}, nil
}

// Ensure interface compliance at compile time
var _ port.Cache = (*RedisCache)(nil)

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	res, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", port.ErrMiss
	}
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return r.client.Del(ctx, keys...).Result()
}

func (r *RedisCache) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
