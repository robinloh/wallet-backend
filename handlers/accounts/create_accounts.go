package accounts

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"github.com/robinloh/wallet-backend/commons"
	"github.com/robinloh/wallet-backend/models"
)

func (a *accountsHandler) CreateAccounts(ctx *fiber.Ctx) error {
	accounts := new([]*models.Account)

	if err := ctx.BodyParser(accounts); err != nil {
		log.Errorf("[CreateAccounts] error parsing request body : %v", err)
		return commons.NewError(ctx, fiber.StatusInternalServerError)

	}

	for _, account := range *accounts {
		if err := uuid.Validate(account.AccountID); err != nil {
			log.Errorf("[CreateAccounts] error validating account ID : %v", err)
			return commons.NewError(ctx, fiber.StatusBadRequest)
		}
	}

	a.postgresDB.Create(accounts)

	return commons.NewSuccess(
		ctx,
		fiber.Map{
			"accounts": accounts,
		},
	)
}
