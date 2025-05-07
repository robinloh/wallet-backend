package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

func (a *accountsHandler) Withdraw(ctx *fiber.Ctx) error {
	req, err := a.validateWithdrawRequest(ctx)
	if err != nil || req == nil {
		return err
	}

	txnID, err := utils.GenerateTxnID()
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Withdraw] Failed to generate transaction ID for account '%s' : %+v", req.ID, err.Error()))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	results, err := a.handleWithdraw(ctx.UserContext(), req, txnID)
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

func (a *accountsHandler) handleWithdraw(ctx context.Context, req *models.Withdraw, txnID string) (interface{}, error) {
	var done int64

	err := a.postgresDB.Db.QueryRow(
		ctx,
		database.WITHDRAW_QUERY,
		req.ID,
		req.Amount,
		txnID,
		database.TxnTypeWithdraw,
	).Scan(&done)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[Withdraw] Error withdrawing from account '%s' : %+v", req.ID, err))
		return &models.WithdrawResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	if done == 0 {
		a.logger.Error(fmt.Sprintf("[Withdraw] Withdrawal '%f' was not done from account '%s'", req.Amount, req.ID))
		return &models.WithdrawResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	return &models.WithdrawResponse{
		AccountID:     req.ID,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: txnID,
	}, err
}

func (a *accountsHandler) validateWithdrawRequest(ctx *fiber.Ctx) (*models.Withdraw, error) {
	req := new(models.WithdrawRequest)

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

	return &models.Withdraw{
		ID:     req.ID,
		Amount: amount,
	}, nil
}
