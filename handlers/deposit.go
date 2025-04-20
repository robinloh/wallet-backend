package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

func (a *accountsHandler) Deposit(ctx *fiber.Ctx) error {
	req, err := a.validateDepositRequest(ctx)
	if err != nil {
		return err
	}

	txnID, err := utils.GenerateTxnID()
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] Failed to generate transaction ID for account '%s' : %+v", req.ID, err.Error()))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	results, err := a.handleDeposit(req, txnID)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	return utils.NewSuccess(
		ctx,
		fiber.Map{
			"accounts": results,
		},
	)
}

func (a *accountsHandler) handleDeposit(req *models.Deposit, txnID string) (interface{}, error) {
	ctx := context.Background()

	tx, err := a.postgresDB.Db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		a.logger.Error("[Deposit] Error starting transaction :" + err.Error())
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	defer func(tx pgx.Tx, ctx context.Context) {
		_ = tx.Rollback(ctx)
	}(tx, ctx)

	_, err = tx.Exec(
		ctx,
		database.DEPOSIT_QUERY,
		req.ID,
		req.Amount,
	)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] Error depositing into account '%s' : %+v", req.ID, err))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	_, err = tx.Exec(
		ctx,
		database.DEPOSIT_INSERT_TRANSACTION_QUERY,
		pgx.NamedArgs{
			"id":              txnID,
			"account_id":      req.ID,
			"amount":          req.Amount,
			"sendreceiveflag": utils.SENDER,
			"sender_id":       req.ID,
			"receiver_id":     req.ID,
			"status":          utils.COMPLETED,
		},
	)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] Error updating transaction '%s' for account '%s' : %+v", txnID, req.ID, err))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	if err = tx.Commit(ctx); err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] Error committing transaction into account '%s' : %+v", req.ID, err))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	return &models.DepositResponse{
		AccountID:     req.ID,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: txnID,
	}, nil
}

func (a *accountsHandler) validateDepositRequest(ctx *fiber.Ctx) (*models.Deposit, error) {
	req := new(models.DepositRequest)

	if err := ctx.BodyParser(req); err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] error parsing request body : %v", err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate((*req).ID)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] request input account ID '%s' is invalid", (*req).ID))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len((*req).Amount) == 0 {
		a.logger.Error(fmt.Sprintf("[Deposit] request input amount '%s' is not specified", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	amount, err := strconv.ParseFloat((*req).Amount, 64)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Deposit] request input amount '%s' is invalid", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if amount <= 0 {
		a.logger.Error(fmt.Sprintf("[Deposit] request input amount '%s' is not greater than zero", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return &models.Deposit{
		ID:     (*req).ID,
		Amount: amount,
	}, nil
}
