package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/redis"
	"github.com/stretchr/testify/assert"
)

func TestGetAccountTransactions(t *testing.T) {
	type fields struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		redis      *redis.Redis
	}
	tests := []struct {
		name      string
		fields    fields
		accountID string
		wantErr   bool
	}{
		{
			name: "Valid Account ID",
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
							"balance": 1.23,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.DEPOSIT_QUERY,
						"d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
						1.23,
						uuid.New().String(),
						database.TxnTypeDeposit,
						"",
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
			accountID: "d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
		},
		{
			name: "Account ID not found",
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
							"balance": 1.23,
						},
					).Scan()
					_ = db.Db.QueryRow(
						context.Background(),
						database.DEPOSIT_QUERY,
						"d6f85c9a-44c1-11f0-b7d1-5234096af7c1",
						1.23,
						uuid.New().String(),
						database.TxnTypeDeposit,
						"",
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
			accountID: uuid.New().String(), // Using a new UUID that does not exist in the database
		},
		{
			name: "Simulate database connection error",
			fields: fields{
				logger: slog.Default(),
				postgresDB: func() *database.Postgres {
					_, err := database.SetupTestDatabase(t, "test_user", "test_password", "test_wallet_backend_db", "localhost")
					if err != nil {
						t.Fatalf("failed to setup test database : %+v", err)
					}
					db := database.ConnectDb(context.Background())
					db.CloseDbConnection(context.Background(), slog.Default()) // Close the database connection to simulate an error
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
			accountID: uuid.New().String(), // Using a new UUID that does not exist in the database
			wantErr:   true,
		},
		{
			name: "Invalid Account ID format",
			fields: fields{
				logger: slog.Default(),
			},
			accountID: "invalid-uuid",
			wantErr:   true,
		},
		{
			name: "Empty Account ID",
			fields: fields{
				logger: slog.Default(),
			},
			accountID: "",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := &accountsHandler{
				logger:     test.fields.logger,
				postgresDB: test.fields.postgresDB,
				redis:      test.fields.redis,
			}
			app := fiber.New()
			app.Get("v1/accounts/transactions/:account_id", a.GetAccountTransactions)

			param := fmt.Sprintf("https://localohost:8080/v1/accounts/transactions/%s", test.accountID)
			req := httptest.NewRequest("GET", param, nil)

			resp, err := app.Test(req, -1)
			if test.wantErr {
				assert.NotEqual(t, fiber.StatusOK, resp.StatusCode, "Expected error response")
			} else {
				assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected success response")
				assert.NoError(t, err)
			}
		})
	}
}
