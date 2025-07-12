package redis

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	redisTest "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestRedis_Acquire(t *testing.T) {
	ctx := context.Background()
	testRedis, err := SetupTestRedis(t)
	require.NoError(t, err)
	defer func(container *redisTest.RedisContainer, ctx context.Context) {
		_ = container.Terminate(ctx)
	}(testRedis.container, ctx)

	conn := testRedis.Redis.RedisPool.Get()
	defer func(conn redis.Conn) {
		_ = conn.Close()
	}(conn)

	r := &Redis{
		RedisPool: testRedis.Redis.RedisPool,
	}

	tests := []struct {
		name        string
		setup       func() error
		key         string
		expected    bool
		expectedErr bool
	}{
		{
			name: "key doesn't exist, should acquire",
			setup: func() error {
				return nil // no setup needed
			},
			key:         "test_key_1",
			expected:    true,
			expectedErr: false,
		},
		{
			name: "key exists, should not acquire",
			setup: func() error {
				_, err := conn.Do("SET", "test_key_2", "lock")
				return err
			},
			key:         "test_key_2",
			expected:    false,
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			require.NoError(t, err)
			acquired, err := r.Acquire(conn, tt.key)
			if tt.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, acquired)
			}
		})
	}
}

func TestRedis_Release(t *testing.T) {
	ctx := context.Background()
	testRedis, err := SetupTestRedis(t)
	require.NoError(t, err)
	defer func(container *redisTest.RedisContainer, ctx context.Context) {
		_ = container.Terminate(ctx)
	}(testRedis.container, ctx)

	conn := testRedis.Redis.RedisPool.Get()
	defer func(conn redis.Conn) {
		_ = conn.Close()
	}(conn)

	r := &Redis{
		RedisPool: testRedis.Redis.RedisPool,
	}

	tests := []struct {
		name            string
		setup           func() error
		key             string
		shouldRelease   bool
		expectedIsError bool
	}{
		{
			name: "key exists, should release successfully",
			setup: func() error {
				_, err := conn.Do("SET", "test_key_1", "value")
				return err
			},
			key:             "test_key_1",
			shouldRelease:   true,
			expectedIsError: false,
		},
		{
			name: "key exists, should not release",
			setup: func() error {
				_, err := conn.Do("SET", "test_key_2", "value")
				return err
			},
			key:             "test_key_2",
			shouldRelease:   false,
			expectedIsError: false,
		},
		{
			name: "key doesn't exist, release requested",
			setup: func() error {
				return nil
			},
			key:             "non_existent_key",
			shouldRelease:   true,
			expectedIsError: false,
		},
		{
			name: "key doesn't exist, no release requested",
			setup: func() error {
				return nil
			},
			key:             "non_existent_key",
			shouldRelease:   false,
			expectedIsError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			require.NoError(t, err)
			err = r.Release(conn, tt.key, tt.shouldRelease)
			if tt.expectedIsError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.shouldRelease {
					exists, err := redis.Bool(conn.Do("EXISTS", tt.key))
					require.NoError(t, err)
					require.False(t, exists)
				}
			}
		})
	}
}

func TestRedis_Publish(t *testing.T) {
	ctx := context.Background()
	testRedis, err := SetupTestRedis(t)
	require.NoError(t, err)
	defer func(container *redisTest.RedisContainer, ctx context.Context) {
		_ = container.Terminate(ctx)
	}(testRedis.container, ctx)

	conn := testRedis.Redis.RedisPool.Get()
	defer func(conn redis.Conn) {
		_ = conn.Close()
	}(conn)

	r := &Redis{
		RedisPool: testRedis.Redis.RedisPool,
	}

	tests := []struct {
		name        string
		setup       func() error
		key         string
		results     fiber.Map
		expectError bool
	}{
		{
			name: "successful publish",
			setup: func() error {
				return nil
			},
			key: "test_channel_1",
			results: fiber.Map{
				"message": "test message",
			},
			expectError: false,
		},
		{
			name: "no subscribers",
			setup: func() error {
				return nil
			},
			key: "test_channel_no_subscribers",
			results: fiber.Map{
				"data": "no subscribers here",
			},
			expectError: false,
		},
		{
			name: "invalid JSON encoding",
			setup: func() error {
				return nil
			},
			key: "test_channel_invalid_json",
			results: fiber.Map{
				"invalid": make(chan int),
			},
			expectError: true,
		},
		{
			name: "parsing error",
			setup: func() error {
				return nil
			},
			key: "test_channel_invalid_json",
			results: fiber.Map{
				"": make(chan int),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setup()
			require.NoError(t, err)

			err = r.Publish(conn, tt.key, tt.results)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedis_HandleMultipleRequests(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		redisPool   *redis.Pool
		tearDown    func(*redis.Pool) error
		redisKey    string
		timeout     time.Duration
		expectedRes fiber.Map
		expectedErr bool
	}{
		{
			name: "timeout error",
			redisPool: func() *redis.Pool {
				r, err := SetupTestRedis(t)
				if err != nil {
					t.Fatalf("Failed to setup test Redis : %+v", err)
				}
				redisConn := r.Redis.RedisPool.Get()
				_, _ = r.Redis.Acquire(redisConn, "channel_success")
				return r.Redis.RedisPool
			}(),
			tearDown: func(redisPool *redis.Pool) error {
				return redisPool.Close()
			},
			redisKey:    "channel_timeout",
			timeout:     100 * time.Millisecond,
			expectedRes: nil,
			expectedErr: true,
		},
		{
			name: "successful response with Received message from key",
			redisPool: func() *redis.Pool {
				r, err := SetupTestRedis(t)
				if err != nil {
					t.Fatalf("Failed to setup test Redis : %+v", err)
				}
				go func() {
					redisConn := r.Redis.RedisPool.Get()
					_, _ = r.Redis.Acquire(redisConn, "channel_success")
					time.Sleep(2 * time.Second)
					_ = r.Redis.Publish(redisConn, "channel_success", fiber.Map{"message": "test message"})
					_ = r.Redis.Release(redisConn, "channel_success", true)
				}()
				return r.Redis.RedisPool
			}(),
			tearDown: func(redisPool *redis.Pool) error {
				return nil
			},
			redisKey:    "channel_success",
			timeout:     5 * time.Second,
			expectedRes: fiber.Map{"message": "test message"},
			expectedErr: false,
		},
		{
			name: "invalid json response",
			redisPool: func() *redis.Pool {
				r, err := SetupTestRedis(t)
				if err != nil {
					t.Fatalf("Failed to setup test Redis : %+v", err)
				}
				go func() {
					redisConn := r.Redis.RedisPool.Get()
					_, _ = r.Redis.Acquire(redisConn, "channel_invalid_json")
					time.Sleep(2 * time.Second)
					_, _ = redisConn.Do("PUBLISH", "channel_invalid_json", "{invalid json")
					_ = r.Redis.Release(redisConn, "channel_invalid_json", true)
				}()
				return r.Redis.RedisPool
			}(),
			tearDown: func(redisPool *redis.Pool) error {
				return redisPool.Close()
			},
			redisKey:    "channel_invalid_json",
			timeout:     5 * time.Second,
			expectedRes: nil,
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Redis{
				RedisPool: tt.redisPool,
				Logger:    slog.Default(),
			}
			res, err := r.HandleMultipleRequests(ctx, tt.redisKey, tt.timeout)
			if tt.expectedErr {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRes, res)
			}

			if tt.tearDown != nil {
				err := tt.tearDown(tt.redisPool)
				require.NoError(t, err)
			}
		})
	}
}
