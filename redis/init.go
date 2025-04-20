package redis

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
)

type Redis struct {
	redis redis.Conn
}

func ConnectRedis() redis.Conn {
	conn, err := redis.Dial("tcp", "redis:6379")
	if err != nil {
		panic(fmt.Sprintf("redis connect err : %s", err))
	}

	return conn
}
