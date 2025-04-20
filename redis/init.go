package redis

import (
	"fmt"
	"sync"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	Redis *redis.Conn
}

var (
	redisInstance *Redis
	redisOnce     sync.Once
)

func ConnectRedis() *Redis {
	redisOnce.Do(func() {
		conn, err := redis.Dial("tcp", "redis:6379")
		if err != nil {
			panic(fmt.Sprintf("redis connect err : %s", err))
		}
		redisInstance = &Redis{
			Redis: &conn,
		}
	})

	return redisInstance
}
