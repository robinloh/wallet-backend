package redis

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/testcontainers/testcontainers-go"
	redisTest "github.com/testcontainers/testcontainers-go/modules/redis"
)

type Redis struct {
	RedisPool *redis.Pool
	Logger    *slog.Logger
}

var (
	redisInstance *Redis
	redisOnce     sync.Once
)

func ConnectRedis(logger *slog.Logger) *Redis {
	redisOnce.Do(func() {
		redisPort := os.Getenv("REDIS_PORT")
		redisInstance = &Redis{
			RedisPool: &redis.Pool{
				Dial: func() (redis.Conn, error) {
					conn, err := redis.Dial("tcp", fmt.Sprintf("redis:%s", redisPort))
					if err != nil {
						return nil, err
					}
					return conn, err
				},
			},
			Logger: logger,
		}
	})

	return redisInstance
}

type TestRedis struct {
	Redis     *Redis
	container *redisTest.RedisContainer
}

func SetupTestRedis(t *testing.T) (*TestRedis, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	container, err := redisTest.Run(
		ctx,
		"redis:latest",
	)

	testcontainers.CleanupContainer(t, container)

	if err != nil {
		return nil, fmt.Errorf("failed to run redis container: %w", err)
	}

	p, err := container.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("unable to get mapped port : %v", err.Error())
	}

	_ = os.Setenv("REDIS_PORT", p.Port())

	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", fmt.Sprintf(":%s", p.Port()))
			if err != nil {
				return nil, err
			}
			return conn, err
		},
	}

	redisInstance = &Redis{
		RedisPool: redisPool,
		Logger:    slog.Default(),
	}

	return &TestRedis{
		Redis:     redisInstance,
		container: container,
	}, nil
}
