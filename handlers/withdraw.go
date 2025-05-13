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

const withdrawOp = "Withdraw"

func (a *accountsHandler) Withdraw(ctx *fiber.Ctx) error {
	req, err := a.validateWithdrawRequest(ctx)
	if err != nil || req == nil {
		return err
	}

	reqHeader, err := a.validateWithdrawHeader(ctx)
	if err != nil || reqHeader == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	redisKey := fmt.Sprintf("%s_%s", reqHeader.IdempotencyKey, withdrawOp)

	redisConn := a.redis.RedisPool.Get()
	defer func(redisConn redis.Conn) {
		err := redisConn.Close()
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] Error closing redis connection for redisKey '%s'. error: %s", withdrawOp, redisKey, err.Error()))
		}
	}(redisConn)

	ok, err := a.redis.Acquire(redisConn, redisKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error acquiring lock for idempotency key '%s' : %v", withdrawOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	shouldRelease := true

	defer func() {
		err := a.redis.Release(redisConn, redisKey, shouldRelease)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] error releasing lock for idempotency key '%s' : %v", withdrawOp, redisKey, err))
		}
	}()

	if !ok {
		shouldRelease = false
		results, err := a.redis.HandleMultipleRequests(ctx.UserContext(), redisKey, 5*time.Second)
		if err != nil || results == nil {
			a.logger.Error(fmt.Sprintf("[%s] error handling multiple requests '%s' : %v", withdrawOp, redisKey, err))
			return utils.NewError(ctx, fiber.StatusInternalServerError)
		}
		a.logger.Info(fmt.Sprintf("[%s] multiple requests detected for '%s' : Results : %+v", withdrawOp, redisKey, results))
		return utils.NewSuccess(ctx, results)
	}

	results, err := a.handleWithdraw(ctx.UserContext(), req, reqHeader)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	successResp := fiber.Map{
		"accounts": results,
	}

	err = a.redis.Publish(redisConn, redisKey, successResp)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Unable to publish results for idempotency key '%s' : %v", withdrawOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	} else {
		a.logger.Debug(fmt.Sprintf("[%s] Successfully published results '%+v' for idempotency key '%s'", withdrawOp, results, redisKey))
	}

	return utils.NewSuccess(ctx, successResp)
}

func (a *accountsHandler) handleWithdraw(ctx context.Context, req *models.Withdraw, reqHeader *models.WithdrawRequestHeader) (interface{}, error) {
	var done int64

	err := a.postgresDB.Db.QueryRow(
		ctx,
		database.WITHDRAW_QUERY,
		req.ID,
		req.Amount,
		reqHeader.IdempotencyKey,
		database.TxnTypeWithdraw,
		"",
	).Scan(&done)

	if err != nil {
		a.logger.Error(fmt.Sprintf("[Withdraw] Error withdrawing from account '%s' : %+v", req.ID, err))
		return &models.WithdrawResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	if done == 0 {
		a.logger.Error(fmt.Sprintf("[Withdraw] Withdrawal '%f' was not done from account '%s'", req.Amount, req.ID))
		return &models.WithdrawResponse{
			AccountID:     req.ID,
			Amount:        req.Amount,
			Status:        utils.FAILED,
			TransactionID: reqHeader.IdempotencyKey,
		}, err
	}

	return &models.WithdrawResponse{
		AccountID:     req.ID,
		Amount:        req.Amount,
		Status:        utils.COMPLETED,
		TransactionID: reqHeader.IdempotencyKey,
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

func (a *accountsHandler) validateWithdrawHeader(ctx *fiber.Ctx) (*models.WithdrawRequestHeader, error) {
	withdrawReqHeader := new(models.WithdrawRequestHeader)

	if err := ctx.ReqHeaderParser(withdrawReqHeader); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body header : %v", withdrawOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len(withdrawReqHeader.IdempotencyKey) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not supplied", withdrawOp, withdrawReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate(withdrawReqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not valid", withdrawOp, withdrawReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return withdrawReqHeader, nil
}
