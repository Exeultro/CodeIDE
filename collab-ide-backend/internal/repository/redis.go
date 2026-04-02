package repository

import (
	"collab-ide-backend/internal/config"
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	Client *redis.Client
}

func NewRedis(cfg *config.Config) (*RedisRepo, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisHost,
		Password: cfg.RedisPwd,
		DB:       0,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisRepo{Client: client}, nil
}
