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

func (a *accountsHandler) Transfer(ctx *fiber.Ctx) error {
	req, err := a.validateTransferRequest(ctx)
	if err != nil || req == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	txnID, err := utils.GenerateTxnID()
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] Failed to generate transaction ID for account '%s' : %+v", req.From, err.Error()))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	results, err := a.handleTransfer(ctx.UserContext(), req, txnID)
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

func (a *accountsHandler) handleTransfer(ctx context.Context, req *models.Transfer, txnID string) (*models.TransferResponse, error) {
	tx, err := a.postgresDB.Db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		a.logger.Error("[Transfer] Error starting transaction :" + err.Error())
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	var withdrawalDone int64

	withdrawalErr := tx.QueryRow(
		ctx,
		database.WITHDRAW_QUERY,
		req.From,
		req.Amount,
		txnID,
		database.TxnTypeSender,
		req.To,
	).Scan(&withdrawalDone)

	if withdrawalErr != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] Error withdrawing from account '%s' : %+v", req.From, err))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	if withdrawalDone == 0 {
		a.logger.Error(fmt.Sprintf("[Transfer] Withdrawing '%f' was not done from account '%s'", req.Amount, req.From))

		err = tx.Commit(ctx)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[Transfer] Error committing database transaction '%s' : %+v", txnID, err))
			_ = tx.Rollback(ctx)
			return &models.TransferResponse{
				From:          req.From,
				To:            req.To,
				Amount:        req.Amount,
				Status:        utils.FAILED,
				TransactionID: txnID,
			}, err
		}

		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	var depositDone int64

	depositErr := tx.QueryRow(
		ctx,
		database.DEPOSIT_QUERY,
		req.To,
		req.Amount,
		txnID,
		database.TxnTypeReceiver,
		req.From,
	).Scan(&depositDone)

	err = tx.Commit(ctx)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] Error committing database transaction '%s' : %+v", txnID, err))
		_ = tx.Rollback(ctx)
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	if depositErr != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] Error depositing into account '%s' : %+v", req.To, err))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	if depositDone == 0 {
		a.logger.Error(fmt.Sprintf("[Transfer] Depositing '%f' was not done into account '%s'", req.Amount, req.To))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: txnID,
		}, err
	}

	return &models.TransferResponse{
		From:          req.From,
		To:            req.To,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: txnID,
	}, nil
}

func (a *accountsHandler) validateTransferRequest(ctx *fiber.Ctx) (*models.Transfer, error) {
	req := new(models.TransferRequest)
	if err := ctx.BodyParser(req); err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] error parsing request body : %v", err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate((*req).From)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] request FROM account ID '%s' is invalid", (*req).From))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err = uuid.Validate((*req).To)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] request TO account ID '%s' is invalid", (*req).To))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len((*req).Amount) == 0 {
		a.logger.Error(fmt.Sprintf("[Transfer] request input amount '%s' is not specified", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	amount, err := strconv.ParseFloat((*req).Amount, 64)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[Transfer] request input amount '%s' is invalid", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if amount <= 0 {
		a.logger.Error(fmt.Sprintf("[Transfer] request input amount '%s' is not greater than zero", (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return &models.Transfer{
		From:   req.From,
		To:     req.To,
		Amount: amount,
	}, nil

}
