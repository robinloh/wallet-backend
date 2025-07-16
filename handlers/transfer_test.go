package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	redis2 "github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/redis"
	"github.com/robinloh/wallet-backend/utils"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_accountsHandler_Transfer(t *testing.T) {
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
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c2",
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "50"}`))
					return ctx
				}(),
			},
			wantErr: false,
		},
		{
			name: "invalid transfer request - invalid JSON",
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
			name: "invalid transfer header - missing idempotency key",
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid deposit header - invalid idempotency key",
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
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
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c2",
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

						redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174000", transferOp)
						_, _ = r.Redis.Acquire(redisConn, redisKey)

						// Publish the result after a short delay
						time.Sleep(2500 * time.Millisecond)
						resp := fiber.Map{
							"accounts": &models.TransferResponse{
								From:          "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
								To:            "d6f85c9a-44c1-11f0-b7d1-5234096af7c2",
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
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
					db := database.ConnectDb(context.Background())
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "d6f85c9a-44c1-11f0-b7d1-5234096af7c2",
							"balance": 0.00,
						},
					).Scan()
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					r, err := redis.SetupTestRedis(t)
					if err != nil {
						t.Fatalf("Failed to setup test Redis : %+v", err)
					}
					// Simulate a request that has acquired the lock but never publishes
					redisConn := r.Redis.RedisPool.Get()
					redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174002", transferOp)
					_, _ = r.Redis.Acquire(redisConn, redisKey)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174002")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "handleTransfer returns error",
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
					ctx.Request().SetBody([]byte(`{"from": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "to": "d6f85c9a-44c1-11f0-b7d1-5234096af7c2", "amount": "2.34"}`))
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
			err := a.Transfer(tt.args.ctx)
			if tt.wantErr {
				assert.NotEqual(t, fiber.StatusOK, tt.args.ctx.Response().StatusCode(), "Expected error response")
			} else {
				assert.Equal(t, fiber.StatusOK, tt.args.ctx.Response().StatusCode(), "Expected success response")
				assert.NoError(t, err)
			}
		})
	}
}

func Test_accountsHandler_validateTransferRequest(t *testing.T) {
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
		want    *models.Transfer
		wantErr bool
	}{
		{
			name: "returns error for non Content-Type JSON format",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"123e4567-e89b-12d3-a456-426614174001","amount":"invalid-amount"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for invalid from account ID format",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"invalid-account","to":"123e4567-e89b-12d3-a456-426614174001","amount":"100.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for invalid to account ID format",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"invalid-account","amount":"100.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for invalid transfer amount format",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"123e4567-e89b-12d3-a456-426614174001","amount":"invalid-amount"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for missing transfer amount",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"123e4567-e89b-12d3-a456-426614174001"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for negative transfer amount",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"123e4567-e89b-12d3-a456-426614174001","amount":"-100.00"}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns valid transfer object for correct input",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Content-Type", "application/json")
					ctx.Request().SetBody([]byte(`{"from":"123e4567-e89b-12d3-a456-426614174000","to":"123e4567-e89b-12d3-a456-426614174001","amount":"100.00"}`))
					return ctx
				}(),
			},
			want: &models.Transfer{
				From:   "123e4567-e89b-12d3-a456-426614174000",
				To:     "123e4567-e89b-12d3-a456-426614174001",
				Amount: 100.00,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &accountsHandler{
				logger:     tt.fields.logger,
				postgresDB: tt.fields.postgresDB,
				redis:      tt.fields.redis,
			}
			got, _ := a.validateTransferRequest(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_accountsHandler_validateTransferHeader(t *testing.T) {
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
		want    *models.TransferRequestHeader
		wantErr bool
	}{
		{
			name: "returns error for missing idempotency key in header",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Del("Idempotency-Key")
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns error for invalid idempotency key format in header",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "invalid-key")
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "returns valid header object for correct idempotency key",
			fields: fields{
				logger:     slog.Default(),
				postgresDB: &database.Postgres{},
				redis:      &redis.Redis{},
			},
			args: args{
				ctx: func() *fiber.Ctx {
					app := fiber.New()
					ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					return ctx
				}(),
			},
			want: &models.TransferRequestHeader{
				IdempotencyKey: "123e4567-e89b-12d3-a456-426614174000",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &accountsHandler{
				logger:     tt.fields.logger,
				postgresDB: tt.fields.postgresDB,
				redis:      tt.fields.redis,
			}
			got, _ := a.validateTransferHeader(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equalf(t, tt.want, got, "validateCreateAccountsHeader(%v)", tt.args.ctx)
		})
	}
}

func Test_accountsHandler_handleTransfer(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	type args struct {
		ctx       context.Context
		req       *models.Transfer
		reqHeader *models.TransferRequestHeader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *models.TransferResponse
		wantErr bool
	}{
		{
			name: "successful transfer",
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
							"id":      "123e4567-e89b-12d3-a456-426614174000",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "123e4567-e89b-12d3-a456-426614174001",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.COMPLETED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: false,
		},
		{
			name: "simulate database connection error - cannot retrieve account transactions",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					db.CloseDbConnection(context.Background(), slog.Default())
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 20.65,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174004",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174004",
			},
			wantErr: true,
		},
		{
			name: "transaction already exists",
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
							"id":      "123e4567-e89b-12d3-a456-426614174000",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "123e4567-e89b-12d3-a456-426614174001",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.WITHDRAW_QUERY,
						"123e4567-e89b-12d3-a456-426614174000",
						"20.65",
						"123e4567-e89b-12d3-a456-426614174004",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.DEPOSIT_QUERY,
						"123e4567-e89b-12d3-a456-426614174001",
						"20.65",
						"123e4567-e89b-12d3-a456-426614174004",
						database.TxnTypeReceiver,
						"123e4567-e89b-12d3-a456-426614174000",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 20.65,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174004",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        20.65,
				Status:        utils.COMPLETED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174004",
			},
			wantErr: false,
		},
		{
			name: "withdraw account not available",
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
							"id":      "123e4567-e89b-12d3-a456-426614174001",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 20.65,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174004",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174004",
			},
			wantErr: false,
		},
		{
			name: "deposit account not available",
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
							"id":      "123e4567-e89b-12d3-a456-426614174000",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 20.65,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174004",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        20.65,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174004",
			},
			wantErr: false,
		},
		{
			name: "insufficient withdraw funds",
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
							"id":      "123e4567-e89b-12d3-a456-426614174000",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "123e4567-e89b-12d3-a456-426614174001",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 50000.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174003",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        50000.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174003",
			},
			wantErr: false,
		},
		{
			name: "zero transfer amount",
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
							"id":      "123e4567-e89b-12d3-a456-426614174000",
							"balance": 100.00,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.INSERT_ACCOUNTS_QUERY,
						pgx.NamedArgs{
							"id":      "123e4567-e89b-12d3-a456-426614174001",
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 0.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174004",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        0.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174004",
			},
			wantErr: false,
		},
		{
			name: "Error starting transaction",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin().WillReturnError(fmt.Errorf("simulated begin transaction error"))
					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: true,
		},
		{
			name: "withdraw transaction error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin()
					mock.ExpectQuery(regexp.QuoteMeta(database.WITHDRAW_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174000",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).WillReturnError(fmt.Errorf("simulated withdraw error"))
					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: true,
		},
		{
			name: "deposit transaction error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin()
					mock.ExpectQuery(regexp.QuoteMeta(database.WITHDRAW_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174000",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).WillReturnRows(pgxmock.NewRows([]string{"withdrawal_done"}).AddRow(int64(1)))
					mock.ExpectCommit()
					mock.ExpectQuery(regexp.QuoteMeta(database.DEPOSIT_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174001",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeReceiver,
						"123e4567-e89b-12d3-a456-426614174000",
					).WillReturnError(fmt.Errorf("simulated deposit error"))
					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: true,
		},
		{
			name: "withdraw not done - Commit error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin()
					mock.ExpectQuery(regexp.QuoteMeta(database.WITHDRAW_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174000",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).WillReturnRows(pgxmock.NewRows([]string{"withdrawal_done"}).AddRow(int64(0)))
					mock.ExpectCommit().WillReturnError(fmt.Errorf("error committing transaction"))
					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: true,
		},
		{
			name: "withdraw transaction error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin()
					mock.ExpectQuery(regexp.QuoteMeta(database.WITHDRAW_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174000",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).WillReturnError(fmt.Errorf("simulated withdraw error"))
					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
			},
			wantErr: true,
		},
		{
			name: "deposit not done - Commit error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectQuery(database.GET_TRANSACTIONS_QUERY).WithArgs(pgx.NamedArgs{
						"id": "123e4567-e89b-12d3-a456-426614174002",
					}).WillReturnRows(pgxmock.NewRows([]string{"id", "from_account_id", "to_account_id", "amount", "status"}))
					mock.ExpectBegin()
					mock.ExpectQuery(regexp.QuoteMeta(database.WITHDRAW_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174000",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeSender,
						"123e4567-e89b-12d3-a456-426614174001",
					).WillReturnRows(pgxmock.NewRows([]string{"withdrawal_done"}).AddRow(int64(1)))
					mock.ExpectQuery(regexp.QuoteMeta(database.DEPOSIT_QUERY)).WithArgs(
						"123e4567-e89b-12d3-a456-426614174001",
						100.0,
						"123e4567-e89b-12d3-a456-426614174002",
						database.TxnTypeReceiver,
						"123e4567-e89b-12d3-a456-426614174000",
					).WillReturnRows(pgxmock.NewRows([]string{"deposit_done"}).AddRow(int64(1)))
					mock.ExpectCommit().WillReturnError(fmt.Errorf("commit error"))

					return &database.Postgres{
						Db: mock,
					}
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
				ctx: context.Background(),
				req: &models.Transfer{
					From:   "123e4567-e89b-12d3-a456-426614174000",
					To:     "123e4567-e89b-12d3-a456-426614174001",
					Amount: 100.0,
				},
				reqHeader: &models.TransferRequestHeader{
					IdempotencyKey: "123e4567-e89b-12d3-a456-426614174002",
				},
			},
			want: &models.TransferResponse{
				From:          "123e4567-e89b-12d3-a456-426614174000",
				To:            "123e4567-e89b-12d3-a456-426614174001",
				Amount:        100.0,
				Status:        utils.FAILED,
				TransactionID: "123e4567-e89b-12d3-a456-426614174002",
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
			got, err := a.handleTransfer(tt.args.ctx, tt.args.req, tt.args.reqHeader)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want.Amount, got.Amount)
			assert.Equal(t, tt.want.From, got.From)
			assert.Equal(t, tt.want.To, got.To)
			assert.Equal(t, tt.want.Status, got.Status)
			assert.NotEmpty(t, got.TransactionID)
		})
	}
}
