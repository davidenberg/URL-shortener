package caching

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) (*RedisStore, error) {
	opts := redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}
	client := redis.NewClient(&opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}
	ret := new(RedisStore)
	ret.client = client
	return ret, nil
}

func (r *RedisStore) Save(ctx context.Context, shortURL, ogURL string, ttl time.Duration) error {
	return r.client.Set(ctx, shortURL, ogURL, ttl).Err()
}

func (r *RedisStore) Get(ctx context.Context, shortURL string) (string, error) {
	return r.client.Get(ctx, shortURL).Result()
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}
