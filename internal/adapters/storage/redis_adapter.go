// Package storage handles Redis storage adapters.
package storage

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisAdapter(addr string, password string, db int) (*RedisAdapter, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisAdapter{
		client: client,
		ctx:    context.Background(),
	}, nil
}

func (r *RedisAdapter) Get(key string) ([]byte, error) {
	val, err := r.client.Get(r.ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func (r *RedisAdapter) Set(key string, data []byte) error {
	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

func (r *RedisAdapter) Delete(key string) error {
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisAdapter) Close() error {
	return r.client.Close()
}
