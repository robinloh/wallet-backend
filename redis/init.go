package redis

import "github.com/gomodule/redigo/redis"

func ConnectRedis() *redis.Pool {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
	}

	return pool
}
