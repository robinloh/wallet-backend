package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
)

func (r *Redis) Acquire(conn redis.Conn, key string) (bool, error) {
	return redis.Bool(conn.Do("SETNX", key, "lock"))
}

func (r *Redis) Release(conn redis.Conn, key string, shouldRelease bool) error {
	var err error
	if shouldRelease {
		_, err = conn.Do("DEL", key)
	}
	return err
}

func (r *Redis) Publish(conn redis.Conn, key string, results fiber.Map) error {
	byteResults, err := json.Marshal(results)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", key, byteResults)
	return err
}

func (r *Redis) HandleMultipleRequests(ctx context.Context, redisKey string, timeout time.Duration) (fiber.Map, error) {
	type resCh struct {
		res []byte
		ch  string
		err error
	}

	conn := r.RedisPool.Get()
	defer func(conn redis.Conn) {
		err := conn.Close()
		if err != nil {
			r.Logger.Error(fmt.Sprintf("Error closing connection for redis pool (for redisKey : %s)", redisKey))
		}
	}(conn)

	psc := redis.PubSubConn{Conn: conn}
	ch := make(chan resCh, 1)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		if err := psc.Subscribe(redisKey); err != nil {
			return
		}
		r.Logger.Info(fmt.Sprintf("Subscribed to key '%s'. Waiting for reply", redisKey))
		for {
			switch n := psc.ReceiveContext(ctxWithTimeout).(type) {
			case error:
				ch <- resCh{
					err: n,
				}
				return
			case redis.Message:
				r.Logger.Info(fmt.Sprintf("Received message from key '%s' : [%s] %+v", redisKey, n.Channel, string(n.Data)))
				ch <- resCh{
					res: n.Data,
					ch:  n.Channel,
				}
				return
			default:
				r.Logger.Info(fmt.Sprintf("Received : %+v", n))
			}
		}
	}()

	select {

	case <-ctxWithTimeout.Done():
		err := fmt.Errorf("timeout while waiting for reply for key : %s", redisKey)
		r.Logger.Info(err.Error())
		return nil, err

	case res := <-ch:
		if err := psc.Unsubscribe(redisKey); err != nil {
			r.Logger.Info(fmt.Sprintf("Error unsubscribing from key '%s' : %+v", redisKey, err))
			return nil, err
		}

		if res.err != nil {
			r.Logger.Info(fmt.Sprintf("Error while receiving context from key '%s' : %+v", redisKey, res.err))
			return nil, res.err
		}

		var results fiber.Map
		err := json.Unmarshal(res.res, &results)
		if err != nil {
			r.Logger.Info(fmt.Sprintf("Error while unmarshalling response from key '%s' : %+v", redisKey, err))
		}

		return results, nil
	}
}
