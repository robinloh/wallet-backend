package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/handlers"
	"github.com/robinloh/wallet-backend/redis"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	app := fiber.New()

	db := database.ConnectDb(ctx)
	defer db.CloseDbConnection(ctx, logger)

	cache := redis.ConnectRedis()

	handler := handlers.Initialize(logger, db, cache)

	app.Post("v1/accounts", handler.CreateAccounts)
	app.Get("v1/accounts/:id", handler.GetAccountBalance)

	app.Post("v1/deposit", handler.Deposit)

	app.Get("v1/accounts/transactions/:account_id", handler.GetAccountTransactions)

	_ = app.Listen(":8080")
}
