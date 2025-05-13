package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

const createAccountsOp = "CreateAccounts"

func (a *accountsHandler) CreateAccounts(ctx *fiber.Ctx) error {
	req, err := a.validateCreateAccountsRequest(ctx)
	if err != nil || req == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	reqHeader, err := a.validateCreateAccountsHeader(ctx)
	if err != nil || reqHeader == nil {
		return utils.NewError(ctx, fiber.StatusBadRequest)
	}

	redisKey := fmt.Sprintf("%s_%s", reqHeader.IdempotencyKey, createAccountsOp)

	redisConn := a.redis.RedisPool.Get()
	defer func(redisConn redis.Conn) {
		err := redisConn.Close()
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] Error closing redis connection for redisKey '%s'. error: %s", createAccountsOp, redisKey, err.Error()))
		}
	}(redisConn)

	ok, err := a.redis.Acquire(redisConn, redisKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error acquiring lock for idempotency key '%s' : %v", createAccountsOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	shouldRelease := true

	defer func() {
		err := a.redis.Release(redisConn, redisKey, shouldRelease)
		if err != nil {
			a.logger.Error(fmt.Sprintf("[%s] error releasing lock for idempotency key '%s' : %v", createAccountsOp, redisKey, err))
		}
	}()

	if !ok {
		shouldRelease = false
		results, err := a.redis.HandleMultipleRequests(ctx.UserContext(), redisKey, 5*time.Second)
		if err != nil || results == nil {
			a.logger.Error(fmt.Sprintf("[%s] error handling multiple requests '%s' : %v", createAccountsOp, redisKey, err))
			return utils.NewError(ctx, fiber.StatusInternalServerError)
		}
		a.logger.Info(fmt.Sprintf("[%s] multiple requests detected for '%s' : Results : %+v", createAccountsOp, redisKey, results))
		return utils.NewSuccess(ctx, results)
	}

	results, err := a.handleCreateAccounts(ctx.UserContext(), req)
	if err != nil {
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	}

	successResp := fiber.Map{
		"accounts": results,
	}

	err = a.redis.Publish(redisConn, redisKey, successResp)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] Unable to publish results for idempotency key '%s' : %v", createAccountsOp, redisKey, err))
		return utils.NewError(ctx, fiber.StatusInternalServerError)
	} else {
		a.logger.Debug(fmt.Sprintf("[%s] Successfully published results '%+v' for idempotency key '%s'", createAccountsOp, results, redisKey))
	}

	return utils.NewSuccess(ctx, successResp)
}

func (a *accountsHandler) validateCreateAccountsRequest(ctx *fiber.Ctx) (*models.AccountRequest, error) {
	accReq := new(models.AccountRequest)

	if err := ctx.BodyParser(accReq); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body : %v", createAccountsOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if (*accReq).Count < 1 {
		a.logger.Error(fmt.Sprintf("[%s] request input count '%d' cannot be less than 1", createAccountsOp, (*accReq).Count))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return accReq, nil
}

func (a *accountsHandler) validateCreateAccountsHeader(ctx *fiber.Ctx) (*models.AccountRequestHeader, error) {
	accReqHeader := new(models.AccountRequestHeader)

	if err := ctx.ReqHeaderParser(accReqHeader); err != nil {
		a.logger.Error(fmt.Sprintf("[%s] error parsing request body header : %v", createAccountsOp, err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if len(accReqHeader.IdempotencyKey) == 0 {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not supplied", createAccountsOp, accReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	err := uuid.Validate(accReqHeader.IdempotencyKey)
	if err != nil {
		a.logger.Error(fmt.Sprintf("[%s] request header IdempotencyKey '%s' is not valid", createAccountsOp, accReqHeader.IdempotencyKey))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	return accReqHeader, nil
}

func (a *accountsHandler) handleCreateAccounts(ctx context.Context, accReq *models.AccountRequest) ([]pgx.NamedArgs, error) {
	batch := &pgx.Batch{}
	argsList := make([]pgx.NamedArgs, 0, accReq.Count)

	for i := 0; i < accReq.Count; i++ {
		accountID, _ := uuid.NewUUID()
		args := pgx.NamedArgs{
			"id":      accountID.String(),
			"balance": 0.00,
		}
		argsList = append(argsList, args)
		batch.Queue(database.INSERT_ACCOUNTS_QUERY, args)
	}

	results := a.postgresDB.Db.SendBatch(ctx, batch)
	defer func(results pgx.BatchResults) {
		err := results.Close()
		if err != nil {
			a.logger.Error(fmt.Sprintf("[handleCreateAccounts] error closing batch: %v", err))
		}
	}(results)

	for i := 0; i < accReq.Count; i++ {
		_, err := results.Exec()
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
				a.logger.Error(fmt.Sprintf("[handleCreateAccounts] account '%s' already exists : %v", argsList[i]["accountid"], err))
				continue
			}
			return argsList, fmt.Errorf("unable to insert row for account '%s' : %v", argsList[i]["accountid"], err)
		}
	}

	a.logger.Info(fmt.Sprintf("[handleCreateAccounts] successfully created . %+v", argsList))

	return argsList, results.Close()
}
