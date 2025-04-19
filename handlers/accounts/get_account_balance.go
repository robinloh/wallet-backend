package accounts

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/commons"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
)

func (a *accountsHandler) GetAccountBalance(ctx *fiber.Ctx) error {
	err := a.validateGetAccountBalanceRequest(ctx)
	if err != nil {
		return commons.NewError(ctx, fiber.StatusBadRequest)
	}

	accounts, err := a.handleGetAccountBalance(ctx)
	if err != nil {
		return commons.NewError(ctx, fiber.StatusInternalServerError)
	}

	if len(accounts) == 0 {
		return commons.NewError(ctx, fiber.StatusNotFound)
	}

	return commons.NewSuccess(
		ctx,
		fiber.Map{
			"accounts": accounts,
		},
	)
}

func (a *accountsHandler) handleGetAccountBalance(ctx *fiber.Ctx) ([]models.AccountResponse, error) {
	results, err := a.postgresDB.Db.Query(
		context.Background(),
		database.GET_ACCOUNT_BALANCE_QUERY,
		pgx.NamedArgs{
			"id": ctx.Params("id"),
		},
	)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[handleGetAccountBalance] unable to query account balance: %+v", err))
		return nil, fmt.Errorf("unable to query account balance : %v", err.Error())
	}
	defer results.Close()

	resp, err := pgx.CollectRows(results, pgx.RowToStructByName[models.AccountResponse])
	a.logger.Info(fmt.Sprintf("[handleGetAccountBalance] account response: %+v . error : %+v", resp, err))

	return resp, err
}

func (a *accountsHandler) validateGetAccountBalanceRequest(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := uuid.Validate(id); err != nil {
		a.logger.Error(fmt.Sprintf("[GetAccountBalance] Invalid account ID '%s'", id))
		return commons.NewError(ctx, fiber.StatusBadRequest)
	}
	return nil
}
