package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

const depositOp = "Deposit"

func (a *accountsHandler) Deposit(ctx *fiber.Ctx) error {
	req, err := a.validateDepositRequest(ctx)
	if err != nil || req == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	reqHeader, err := a.validateDepositHeader(ctx)
	if err != nil || reqHeader == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	redisKey := fmt.Sprintf("%s_%s", reqHeader.IdempotencyKey, depositOp)

	redisConn := a.redis.RedisPool.Get()
	defer func(redisConn redis.Conn) {
		err := redisConn.Close()
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] Error closing redis connection for redisKey '%s'. error: %s", depositOp, redisKey, err.Error()))
		}
	}(redisConn)

	ok, err := a.redis.Acquire(redisConn, redisKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error acquiring lock for idempotency key '%s' : %v", depositOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	shouldRelease := true

	defer func() {
		err := a.redis.Release(redisConn, redisKey, shouldRelease)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] error releasing lock for idempotency key '%s' : %v", depositOp, redisKey, err))
		}
	}()

	if !ok {
		shouldRelease = false
		results, err := a.redis.HandleMultipleRequests(ctx.UserContext(), redisKey, 5*time.Second)
		if err != nil || results == nil {
			a.logger.Error(fmt.Sprintf("[%s] error handling multiple requests '%s' : %v", depositOp, redisKey, err))
			return utils.NewError(ctx, fiber.StatusInternalServerError)
		}
		a.logger.Info(fmt.Sprintf("[%s] multiple requests detected for '%s' : Results : %+v", depositOp, redisKey, results))
		return utils.NewSuccess(ctx, results)
	}

	results, err := a.handleDeposit(ctx.UserContext(), req, reqHeader)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	successResp := fiber.Map{
		"accounts": results,
	}

	err = a.redis.Publish(redisConn, redisKey, successResp)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Unable to publish results for idempotency key '%s' : %v", depositOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	} else {
		a.logger.Debug(fmt.Sprintf("[%s] Successfully published results '%+v' for idempotency key '%s'", depositOp, results, redisKey))
	}

	return utils.NewSuccess(ctx, successResp)
}

func (a *accountsHandler) handleDeposit(ctx context.Context, req *models.Deposit, reqHeader *models.DepositRequestHeader) (interface{}, error) {
	res, err := a.handleGetTransactions(ctx, reqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error finding existing transaction '%s' : %+v", depositOp, reqHeader.IdempotencyKey, err))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if len(res) > 0 {
		a.logger.Info(fmt.Sprintf("[%s] There was existing transaction '%s' : %+v", depositOp, reqHeader.IdempotencyKey, res))
		return &models.DepositResponse{
			AccountID:     res[0].AccountID,
			Amount:        res[0].Amount,
			Status:        res[0].Status,
			TransactionID: res[0].TransactionID,
		}, err
	}

	var done int64

	err = a.postgresDB.Db.QueryRow(
		ctx,
		database.DEPOSIT_QUERY,
		req.ID,
		req.Amount,
		reqHeader.IdempotencyKey,
		database.TxnTypeDeposit,
		"",
	).Scan(&done)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Error depositing from account '%s' : %+v", depositOp, req.ID, err))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if done == 0 {
		a.logger.Error(fmt.Sprintf("[%s] Depositing '%f' was not done from account '%s'", depositOp, req.Amount, req.ID))
		return &models.DepositResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	return &models.DepositResponse{
		AccountID:     req.ID,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: reqHeader.IdempotencyKey,
	}, nil
}

func (a *accountsHandler) validateDepositRequest(ctx *fiber.Ctx) (*models.Deposit, error) {
	req := new(models.DepositRequest)

	if err := ctx.BodyParser(req); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body : %v", depositOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate((*req).ID)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request input account ID '%s' is invalid", depositOp, (*req).ID))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len((*req).Amount) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is not specified", depositOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	amount, err := strconv.ParseFloat((*req).Amount, 64)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is invalid", depositOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if amount <= 0 {
		a.logger.Error(fmt.Sprintf("[%s] request input amount '%s' is not greater than zero", depositOp, (*req).Amount))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return &models.Deposit{
		ID:     (*req).ID,
		Amount: amount,
	}, nil
}

func (a *accountsHandler) validateDepositHeader(ctx *fiber.Ctx) (*models.DepositRequestHeader, error) {
	depositReqHeader := new(models.DepositRequestHeader)

	if err := ctx.ReqHeaderParser(depositReqHeader); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body header : %v", depositOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len(depositReqHeader.IdempotencyKey) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not supplied", depositOp, depositReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate(depositReqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not valid", depositOp, depositReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return depositReqHeader, nil
}
