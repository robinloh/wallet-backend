package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

func (a *accountsHandler) GetAccountTransactions(ctx *fiber.Ctx) error {
	req, err := a.validateGetAccountTransactionsRequest(ctx)
	if err != nil || req == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	transactions, err := a.handleGetAccountTransactions(ctx.UserContext(), req)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	if len(transactions) == 0 {
		return utils.NewError(ctx, fiber.StatusNotFound)
	}

	return utils.NewSuccess(
		ctx,
		fiber.Map{
			"transactions": transactions,
		},
	)
}

func (a *accountsHandler) handleGetAccountTransactions(ctx context.Context, req *models.AccountTransactionsRequest) ([]models.AccountTransactionsResponse, error) {
	results, err := a.postgresDB.Db.Query(
		ctx,
		database.GET_ACCOUNT_TRANSACTIONS_QUERY,
		pgx.NamedArgs{
			"account_id": req.AccountID,
		},
	)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[handleGetAccountTransactions] unable to query account transactions: %+v", err))
		return nil, fmt.Errorf("unable to query account transactions : %v", err.Error())
	}

	defer results.Close()

	resp := make([]models.AccountTransactionsResponse, 0)

	for results.Next() {
		account := models.AccountTransactionsResponse{}
		err = results.Scan(&account.AccountID, &account.TransactionID, &account.Amount, &account.TxnType, &account.SenderID, &account.ReceiverID, &account.Timestamp, &account.Status)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[handleGetAccountTransactions] unable to parse account transactions: %+v", err))
			return nil, fmt.Errorf("unable to parse account transactions : %v", err.Error())
		}
		account.Timestamp = utils.ConvertTimezone(account.Timestamp)
		resp = append(resp, account)
	}

	return resp, err
}

func (a *accountsHandler) validateGetAccountTransactionsRequest(ctx *fiber.Ctx) (*models.AccountTransactionsRequest, error) {
	accountId := ctx.Params("account_id")
	if err := uuid.Validate(accountId); err != nil {
		a.logger.Error(fmt.Sprintf("[GetAccountBalance] Invalid account ID '%s'", accountId))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}
	return &models.AccountTransactionsRequest{
		AccountID: accountId,
	}, nil
}
