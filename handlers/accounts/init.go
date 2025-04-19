package accounts

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/robinloh/wallet-backend/database"
)

type APIs interface {
	CreateAccounts(*fiber.Ctx) error
}

type accountsHandler struct {
	logger     *slog.Logger
	postgresDB *database.Postgres
}

func Initialize(logger *slog.Logger, postgresDB *database.Postgres) APIs {
	accountsHandler := &accountsHandler{
		logger:     logger,
		postgresDB: postgresDB,
	}
	return accountsHandler
}
