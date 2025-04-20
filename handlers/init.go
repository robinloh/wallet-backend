package handlers

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/robinloh/wallet-backend/database"
)

type APIs interface {
	CreateAccounts(*fiber.Ctx) error
	GetAccountBalance(*fiber.Ctx) error
}

type accountsHandler struct {
	logger     *slog.Logger
	postgresDB *database.Postgres
	redis      redis.Conn
}

func Initialize(logger *slog.Logger, postgresDB *database.Postgres, cache redis.Conn) APIs {
	accountsHandler := &accountsHandler{
		logger:     logger,
		postgresDB: postgresDB,
		redis:      cache,
	}
	return accountsHandler
}
