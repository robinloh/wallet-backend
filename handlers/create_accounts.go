package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/models"
	"github.com/robinloh/wallet-backend/utils"
)

func (a *accountsHandler) CreateAccounts(ctx *fiber.Ctx) error {
	accReq, err := a.validateRequest(ctx)
	if err != nil {
		return err
	}

	results, err := a.handleCreateAccounts(accReq)
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

func (a *accountsHandler) validateRequest(ctx *fiber.Ctx) (*models.AccountRequest, error) {
	accReq := new(models.AccountRequest)

	if err := ctx.BodyParser(accReq); err != nil {
		a.logger.Error(fmt.Sprintf("[CreateAccounts] error parsing request body : %v", err))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}

	if (*accReq).Count < 1 {
		a.logger.Error(fmt.Sprintf("[CreateAccounts] request input count '%d' cannot be less than 1", (*accReq).Count))
		return nil, utils.NewError(ctx, fiber.StatusBadRequest)
	}
	return accReq, nil
}

func (a *accountsHandler) handleCreateAccounts(accReq *models.AccountRequest) ([]pgx.NamedArgs, error) {
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

	results := a.postgresDB.Db.SendBatch(context.Background(), batch)
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
