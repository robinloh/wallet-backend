package main

import (
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/handlers/accounts"
)

func main() {
	app := fiber.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db := database.ConnectDb()
	defer db.CloseDbConnection()

	accountsHandler := accounts.Initialize(logger, db)

	app.Post("v1/accounts", accountsHandler.CreateAccounts)
	app.Get("v1/accounts/:id", accountsHandler.GetAccountBalance)

	_ = app.Listen(":8080")
}
