package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	redis2 "github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/redis"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func Test_accountsHandler_CreateAccounts(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	type args struct {
		ctx *fiber.Ctx
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantDoubleRequests bool
		wantErr            bool
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
					return database.ConnectDb(context.Background())
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
					ctx.Request().SetBody([]byte(`{"count": 2}`))
					return ctx
				}(),
			},
			wantErr: false,
		},
		{
			name: "invalid JSON in request body",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					return &database.Postgres{}
				}(),
				redis: func() *redis.Redis {
					return &redis.Redis{}
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count":.`)) // malformed JSON
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "invalid request header - idemotency key missing",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					return &database.Postgres{}
				}(),
				redis: func() *redis.Redis {
					return &redis.Redis{}
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count": 1}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "database connection failure",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					db.CloseDbConnection(context.Background(), slog.Default()) // simulate closed database
					return db
				}(),
				redis: func() *redis.Redis {
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count": 2}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "redis lock acquisition failure",
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
					return &redis.Redis{
						RedisPool: &redis2.Pool{
							Dial: func() (redis2.Conn, error) {
								conn, err := redis2.Dial("tcp", fmt.Sprintf(":%s", "0")) // Invalid port to simulate failure
								if err != nil {
									return nil, err
								}
								return conn, err
							},
						},
					}
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count": 2}`))
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

						redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174000", createAccountsOp)
						_, _ = r.Redis.Acquire(redisConn, redisKey)

						// Publish the result after a short delay
						time.Sleep(2500 * time.Millisecond)
						resp := fiber.Map{
							"accounts": &models.AccountResponse{
								ID:      "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
								Balance: "0",
							},
							"success": true,
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
					ctx.Request().SetBody([]byte(`{"count": 1}`))
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
					redisKey := fmt.Sprintf("%s_%s", "123e4567-e89b-12d3-a456-426614174002", createAccountsOp)
					_, _ = r.Redis.Acquire(redisConn, redisKey)
					return r.Redis
				}(),
			},
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174002")
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count": 2}`))
					return ctx
				}(),
			},
			wantErr: true,
		},
		{
			name: "database error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					// Simulate database error by closing the connection
					err = db.Db.Close(context.Background())
					if err != nil {
						t.Fatalf("failed to close database connection : %+v", err)
					}
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
					ctx.Request().SetBody([]byte(`{"count": 2}`))
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
			_ = a.CreateAccounts(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
		})
	}
}

func Test_accountsHandler_validateCreateAccountsRequest(t *testing.T) {
	type args struct {
		ctx *fiber.Ctx
	}

	tests := []struct {
		name    string
		args    args
		want    *models.AccountRequest
		wantErr bool
	}{
		{
			name: "valid request",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count":2}`))
					return ctx
				}(),
			},
			want:    &models.AccountRequest{Count: 2},
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
			name: "count less than 1",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{"count":-1}`))
					return ctx
				}(),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing count field",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.SetContentType("application/json")
					ctx.Request().SetBody([]byte(`{}`))
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
			got, _ := a.validateCreateAccountsRequest(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_accountsHandler_validateCreateAccountsHeader(t *testing.T) {
	type args struct {
		ctx *fiber.Ctx
	}
	tests := []struct {
		name    string
		args    args
		want    *models.AccountRequestHeader
		wantErr bool
	}{
		{
			name: "valid header",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					ctx.Request().Header.Set("Idempotency-Key", "123e4567-e89b-12d3-a456-426614174000") // Valid UUID
					return ctx
				}(),
			},
			want: &models.AccountRequestHeader{
				IdempotencyKey: "123e4567-e89b-12d3-a456-426614174000",
			},
			wantErr: false,
		},
		{
			name: "missing idempotency key",
			args: args{
				ctx: func() *fiber.Ctx {
					ctx := fiber.New().AcquireCtx(&fasthttp.RequestCtx{})
					// No Idempotency-Key header set
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
					ctx.Request().Header.Set("Idempotency-Key", "invalid-uuid") // Invalid UUID
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
					ctx.Request().Header.Set("Idempotency-Key", "") // Empty key
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
			got, _ := a.validateCreateAccountsHeader(tt.args.ctx)
			assert.Equal(t, tt.wantErr, tt.args.ctx.Response().StatusCode() != fiber.StatusOK)
			assert.Equalf(t, tt.want, got, "validateCreateAccountsHeader(%v)", tt.args.ctx)
		})
	}
}

func Test_accountsHandler_handleCreateAccounts(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	type args struct {
		ctx    context.Context
		accReq *models.AccountRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []pgx.NamedArgs
		wantErr bool
	}{
		{
			name: "valid account creation",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, _ = database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					return database.ConnectDb(context.Background())
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				accReq: &models.AccountRequest{
					Count: 2,
				},
			},
			want: []pgx.NamedArgs{
				{"id": "d6f85c9a-44c1-11f0-b7d1-5234096af7c1", "balance": 0.00},
				{"id": "d6f8606e-44c1-11f0-b7d1-5234096af7c1", "balance": 0.00},
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
					db.CloseDbConnection(context.Background(), slog.Default())
					return db
				}(),
				redis: func() *redis.Redis {
					_, _ = redis.SetupTestRedis(t)
					return redis.ConnectRedis(slog.Default())
				}(),
			},
			args: args{
				ctx: context.Background(),
				accReq: &models.AccountRequest{
					Count: 2,
				},
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
			got, err := a.handleCreateAccounts(tt.args.ctx, tt.args.accReq)
			assert.Equal(t, tt.wantErr, err != nil)
			for _, arg := range got {
				assert.NoError(t, uuid.Validate(arg["id"].(string)))
				assert.Equal(t, arg["balance"], float64(0))
			}
		})
	}
}
