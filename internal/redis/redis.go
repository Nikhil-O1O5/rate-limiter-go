package redis

import (
	"context"
	"fmt"

	"github.com/Nikhil-O1O5/rate-limiter-go/internal/config"
	"github.com/redis/go-redis/v9"
)

func New(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}
