package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

const transferOp = "Transfer"

func (a *accountsHandler) Transfer(ctx *fiber.Ctx) error {
	req, err := a.validateTransferRequest(ctx)
	if err != nil || req == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	reqHeader, err := a.validateTransferHeader(ctx)
	if err != nil || reqHeader == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	redisKey := fmt.Sprintf("%s_%s", reqHeader.IdempotencyKey, transferOp)

	redisConn := a.redis.RedisPool.Get()
	defer func(redisConn redis.Conn) {
		err := redisConn.Close()
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] Error closing redis connection for redisKey '%s'. error: %s", transferOp, redisKey, err.Error()))
		}
	}(redisConn)

	ok, err := a.redis.Acquire(redisConn, redisKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error acquiring lock for idempotency key '%s' : %v", transferOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	shouldRelease := true

	defer func() {
		err := a.redis.Release(redisConn, redisKey, shouldRelease)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] error releasing lock for idempotency key '%s' : %v", transferOp, redisKey, err))
		}
	}()

	if !ok {
		shouldRelease = false
		results, err := a.redis.HandleMultipleRequests(ctx.UserContext(), redisKey, 5*time.Second)
		if err != nil || results == nil {
			a.logger.Error(fmt.Sprintf("[%s] error handling multiple requests '%s' : %v", transferOp, redisKey, err))
			return utils.NewError(ctx, fiber.StatusInternalServerError)
		}
		a.logger.Info(fmt.Sprintf("[%s] multiple requests detected for '%s' : Results : %+v", transferOp, redisKey, results))
		return utils.NewSuccess(ctx, results)
	}

	results, err := a.handleTransfer(ctx.UserContext(), req, reqHeader)
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

func (a *accountsHandler) handleTransfer(ctx context.Context, req *models.Transfer, reqHeader *models.TransferRequestHeader) (*models.TransferResponse, error) {
	res, err := a.handleGetTransactions(ctx, reqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error finding existing transaction '%s' : %+v", transferOp, reqHeader.IdempotencyKey, err))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if len(res) > 0 {
		a.logger.Info(fmt.Sprintf("[%s] There was existing transaction '%s' : %+v", transferOp, reqHeader.IdempotencyKey, res))
		return &models.TransferResponse{
			From:          res[0].SenderID,
			To:            res[0].ReceiverID,
			Amount:        res[0].Amount,
			Status:        res[0].Status,
			TransactionID: res[0].TransactionID,
		}, err
	}

	tx, err := a.postgresDB.Db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error starting transaction : %+v", transferOp, err.Error()))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	var withdrawalDone int64

	withdrawalErr := tx.QueryRow(
		ctx,
		database.WITHDRAW_QUERY,
		req.From,
		req.Amount,
		reqHeader.IdempotencyKey,
		database.TxnTypeSender,
		req.To,
	).Scan(&withdrawalDone)

	if withdrawalErr != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error withdrawing from account '%s' : %+v", transferOp, req.From, err))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if withdrawalDone == 0 {
		a.logger.Error(fmt.Sprintf("[%s] Withdrawing '%f' was not done from account '%s'", transferOp, req.Amount, req.From))

		err = tx.Commit(ctx)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] Error committing database transaction '%s' : %+v", transferOp, reqHeader.IdempotencyKey, err))
			_ = tx.Rollback(ctx)
			return &models.TransferResponse{
				From:          req.From,
				To:            req.To,
				Amount:        req.Amount,
				Status:        utils.FAILED,
				TransactionID: reqHeader.IdempotencyKey,
			}, err
		}

		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	var depositDone int64

	depositErr := tx.QueryRow(
		ctx,
		database.DEPOSIT_QUERY,
		req.To,
		req.Amount,
		reqHeader.IdempotencyKey,
		database.TxnTypeReceiver,
		req.From,
	).Scan(&depositDone)

	err = tx.Commit(ctx)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error committing database transaction '%s' : %+v", transferOp, reqHeader.IdempotencyKey, err))
		_ = tx.Rollback(ctx)
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if depositErr != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error depositing into account '%s' : %+v", transferOp, req.To, err))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if depositDone == 0 {
		a.logger.Error(fmt.Sprintf("[%s] Depositing '%f' was not done into account '%s'", transferOp, req.Amount, req.To))
		return &models.TransferResponse{
			From:          req.From,
			To:            req.To,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	return &models.TransferResponse{
		From:          req.From,
		To:            req.To,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: reqHeader.IdempotencyKey,
	}, nil
}

func (a *accountsHandler) validateTransferRequest(ctx *fiber.Ctx) (*models.Transfer, error) {
	req := new(models.TransferRequest)
	if err := ctx.BodyParser(req); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body : %v", transferOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate((*req).From)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request FROM account ID '%s' is invalid", transferOp, (*req).From))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err = uuid.Validate((*req).To)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request TO account ID '%s' is invalid", transferOp, (*req).To))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len((*req).Amount) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is not specified", transferOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	amount, err := strconv.ParseFloat((*req).Amount, 64)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is invalid", transferOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if amount <= 0 {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is not greater than zero", transferOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return &models.Transfer{
		From:   req.From,
		To:     req.To,
		Amount: amount,
	}, nil

}

func (a *accountsHandler) validateTransferHeader(ctx *fiber.Ctx) (*models.TransferRequestHeader, error) {
	depositReqHeader := new(models.TransferRequestHeader)

	if err := ctx.ReqHeaderParser(depositReqHeader); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body header : %v", transferOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len(depositReqHeader.IdempotencyKey) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not supplied", transferOp, depositReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate(depositReqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not valid", transferOp, depositReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return depositReqHeader, nil
}
