package accounts

import (
	"log/slog"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

type APIs interface {
	CreateAccounts(*fiber.Ctx) error
}

type accountsHandler struct {
	logger     *slog.Logger
	postgresDB *gorm.DB
}

func Initialize(logger *slog.Logger, postgresDB *gorm.DB) APIs {
	accountsHandler := &accountsHandler{
		logger:     logger,
		postgresDB: postgresDB,
	}
	return accountsHandler
}
