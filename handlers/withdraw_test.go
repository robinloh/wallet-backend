package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	redis2 "github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/redis"
	"github.com/robinloh/wallet-backend/utils"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_accountsHandler_Withdraw(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	type args struct {
		ctx *fiber.Ctx
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid request",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 100.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: false,
		},
		{
			name: "valid request, but insufficient balance to withdraw",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 0.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid Withdraw request - invalid JSON",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, _ := redis.SetupTestRedis(t)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id":`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid Withdraw header - missing idempotency key",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, _ := redis.SetupTestRedis(t)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid Withdraw header - invalid idempotency key",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, _ := redis.SetupTestRedis(t)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "not-a-uuid")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "redis acquire error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					// Close the Redis connection to simulate connection error
					_ = r.Redis.RedisPool.Close()
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174001")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "handle multiple requests - successful concurrent request",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 0.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					go func() {
						redisConn := r.Redis.RedisPool.Get()
						defer func(redisConn redis2.Conn) {
							_ = redisConn.Close()
						}(redisConn)

						redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174000", withdrawOp)
						_, _ = r.Redis.Acquire(redisConn, redisKey)

						// Publish the result after a short delay
						time.Sleep(2500 * time.Millisecond)
						resp := fiber.Map{
							"accounts": &models.WithdrawResponse{
								AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
								Amount:        2.34,
								Status:        utils.COMPLETED,
								TransactionID: "123e4567-e89b-12d3-a456-426614174000",
							},
						}
						_ = r.Redis.Publish(redisConn, redisKey, resp)
						_ = r.Redis.Release(redisConn, redisKey, true)
					}()

					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: false,
		},
		{
			name: "handle multiple requests - timeout waiting for response",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					// Simulate a request that has acquired the lock but never publishes
					redisConn := r.Redis.RedisPool.Get()
					redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174002", withdrawOp)
					_, _ = r.Redis.Acquire(redisConn, redisKey)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174002")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "handleWithdraw returns error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					db.CloseDbConnection(context.Background(), slog.Default()) // Simulate DB error
					return db
				}(),
				redis: func() *redis.Redis {
					r, _ := redis.SetupTestRedis(t)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174010")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &accountsHandler{
				logger:     tt.fields.logger,
				postgresDB: tt.fields.postgresDB,
				redis:      tt.fields.redis,
			}
			err := a.Withdraw(tt.args.ctx)
			if tt.wantErr {
				assert.NotEqual(t, fiber.StatusOK, tt.args.ctx.Response().StatusCode(), "Expected error response")
			} else {
				assert.Equal(t, fiber.StatusOK, tt.args.ctx.Response().StatusCode(), "Expected success response")
				assert.NoError(t, err)
			}
		})
	}
}

func Test_accountsHandler_validateWithdrawRequest(t *testing.T) {
	type args struct {
		ctx *fiber.Ctx
	}
	tests := []struct {
		name    string
		args    args
		want    *models.Withdraw
		wantErr bool
	}{
		{
			name: "valid request",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960","amount": "20.65"}`))
					return ctx
				}(),
			},
			want:    &models.Withdraw{ID: "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960", Amount: 20.65},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count":`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing amount field",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing id field",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"amount": "20"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "negative amount",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960","amount": "-10.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "zero amount",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960","amount": "0.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid UUID for id",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "invalid-uuid","amount": "20.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty id field",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "","amount": "20.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty amount field",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960","amount": ""}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "non-numeric amount",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"id": "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960","amount": "abc"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	a := &accountsHandler{
		logger: slog.Default(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := a.validateWithdrawRequest(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_accountsHandler_validateWithdrawHeader(t *testing.T) {
	type args struct {
		ctx *fiber.Ctx
	}

	tests := []struct {
		name    string
		args    args
		want    *models.WithdrawRequestHeader
		wantErr bool
	}{
		{
			name: "valid header",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960")
					return ctx
				}(),
			},
			want:    &models.WithdrawRequestHeader{IdempotencyKey: "88fb6b1b-4493-11f0-91f8-ba1e0bcd9960"},
			wantErr: false,
		},
		{
			name: "missing idempotency key",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid idempotency key",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "invalid-uuid")
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty idempotency key",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "")
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
	}

	a := &accountsHandler{
		logger: slog.Default(),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := a.validateWithdrawHeader(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equalf(t, tt.want, got, "validateCreateAccountsHeader(%v)", tt.args.ctx)
		})
	}
}

func Test_accountsHandler_handleWithdraw(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	type args struct {
		ctx       context.Context
		req       *models.Withdraw
		reqHeader *models.WithdrawRequestHeader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.WithdrawResponse
		wantErr bool
	}{
		{
			name: "valid Withdraw",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 1000.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.COMPLETED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
			wantErr: false,
		},
		{
			name: "invalid withdraw - insufficient balance",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 0.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
			wantErr: true,
		},
		{
			name: "transaction already exists",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 1000.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.WITHDRAW_QUERY,
						"d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
						"20.65",
						"46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
						database.TxnTypeWithdraw,
						"",
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.COMPLETED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
			wantErr: false,
		},
		{
			name: "database error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					db.CloseDbConnection(context.Background(), slog.Default()) // Close the connection to simulate a database error
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			wantErr: true,
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
		},
		{
			name: "error Withdrawing - no accounts table",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 0.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						"DROP TABLE IF EXISTS accounts CASCADE",
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
			wantErr: true,
		},
		{
			name: "no Withdraw done",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "c9f27a37-4492-11f0-8f4f-bac11744a8ec", // Different account ID
							"balance": 0.00,
						},
					).Scan()
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: &models.Withdraw{
					ID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
					Amount: 20.65,
				},
				reqHeader: &models.WithdrawRequestHeader{
					IdempotencyKey: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
				},
			},
			want: &models.WithdrawResponse{
				AccountID:     "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "46f20f2f-2f4e-11f0-9ece-d6dceef71cac",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &accountsHandler{
				logger:     tt.fields.logger,
				postgresDB: tt.fields.postgresDB,
				redis:      tt.fields.redis,
			}
			got, err := a.handleWithdraw(tt.args.ctx, tt.args.req, tt.args.reqHeader)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want.Amount, got.Amount)
			assert.Equal(t, tt.want.AccountID, got.AccountID)
			assert.Equal(t, tt.want.Status, got.Status)
			assert.NotEmpty(t, got.TransactionID)
		})
	}
}
