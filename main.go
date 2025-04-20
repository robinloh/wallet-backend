package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/handlers"
	"github.com/robinloh/wallet-backend/redis"
)

func main() {
	app := fiber.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db := database.ConnectDb()
	defer db.CloseDbConnection()

	cache := redis.ConnectRedis()

	handler := handlers.Initialize(logger, db, cache)

	app.Post("v1/accounts", handler.CreateAccounts)
	app.Get("v1/accounts/:id", handler.GetAccountBalance)

	_ = app.Listen(":8080")
}
