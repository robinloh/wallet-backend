package commons

import (
	"github.com/gofiber/fiber/v2"
)

func NewError(ctx *fiber.Ctx, errStatus int, errMsgs ...int) error {
	return ctx.
		Status(errStatus).
		JSON(fiber.Map{
			"success": false,
			"error":   errStatus,
			"message": errMsgs,
		})
}

func NewSuccess(ctx *fiber.Ctx, details fiber.Map) error {
	details["success"] = true
	return ctx.
		Status(fiber.StatusOK).
		JSON(details)
}
