package handlers

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/redis"
)

type APIs interface {
	CreateAccounts(*fiber.Ctx) error
	GetAccountBalance(*fiber.Ctx) error
}

type accountsHandler struct {
	logger     *slog.Logger
	postgresDB *database.Postgres
	redis      *redis.Redis
}

func Initialize(logger *slog.Logger, postgresDB *database.Postgres, cache *redis.Redis) APIs {
	accountsHandler := &accountsHandler{
		logger:     logger,
		postgresDB: postgresDB,
		redis:      cache,
	}
	return accountsHandler
}
