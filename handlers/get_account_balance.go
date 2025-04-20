package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

func (a *accountsHandler) GetAccountBalance(ctx *fiber.Ctx) error {
	err := a.validateGetAccountBalanceRequest(ctx)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	accounts, err := a.handleGetAccountBalance(ctx)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	if len(accounts) == 0 {
		return utils.NewError(ctx, fiber.StatusNotFound)
	}

	return utils.NewSuccess(
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

	resp := make([]models.AccountResponse, 0)

	for results.Next() {
		account := models.AccountResponse{}
		var bal pgtype.Float8
		err = results.Scan(&account.ID, &bal)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[handleGetAccountBalance] unable to parse account balance: %+v", err))
			return nil, fmt.Errorf("unable to parse account balance : %v", err.Error())
		}
		account.Balance = fmt.Sprintf("%+v", bal)
		resp = append(resp, account)
	}

	return resp, err
}

func (a *accountsHandler) validateGetAccountBalanceRequest(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if err := uuid.Validate(id); err != nil {
		a.logger.Error(fmt.Sprintf("[GetAccountBalance] Invalid account ID '%s'", id))
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}
	return nil
}
