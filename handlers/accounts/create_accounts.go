package accounts

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/commons"
	"github.com/robinloh/wallet-backend/models"
)

func (a *accountsHandler) CreateAccounts(ctx *fiber.Ctx) error {
	accounts := new([]*models.Account)

	if err := ctx.BodyParser(accounts); err != nil {
		a.logger.Error(fmt.Sprintf("[CreateAccounts] error parsing request body : %v", err))
		return commons.NewError(ctx, fiber.StatusInternalServerError)

	}

	err := a.handleCreateAccounts(accounts)
	if err != nil {
		return commons.NewError(ctx, fiber.StatusInternalServerError)
	}

	return commons.NewSuccess(
		ctx,
		fiber.Map{
			"accounts": accounts,
		},
	)
}

func (a *accountsHandler) handleCreateAccounts(accounts *[]*models.Account) error {
	entries := [][]any{}
	columns := []string{"accountid", "balance"}
	tableName := "accounts"

	for _, account := range *accounts {
		entries = append(entries, []any{account.AccountID, 0.00})
	}

	_, err := a.postgresDB.Db.CopyFrom(
		context.Background(),
		pgx.Identifier{tableName},
		columns,
		pgx.CopyFromRows(entries),
	)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[handleCreateAccounts] error copying rows: %v", err))
		return err
	}

	a.logger.Info(fmt.Sprintf("[handleCreateAccounts] successful created accounts: %#v", entries))
	return nil
}
