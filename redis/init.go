package redis

import (
	"log/slog"
	"sync"

	"github.com/gomodule/redigo/redis"
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
		redisInstance = &Redis{
			RedisPool: &redis.Pool{
				Dial: func() (redis.Conn, error) {
					conn, err := redis.Dial("tcp", "redis:6379")
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
