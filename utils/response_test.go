package utils

import (
	"encoding/json"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name      string
		errStatus int
		errMsgs   []int
		want      fiber.Map
	}{
		{
			name:      "single error message",
			errStatus: fiber.StatusBadRequest,
			errMsgs:   []int{123},
			want: fiber.Map{
				"success": false,
				"error":   float64(fiber.StatusBadRequest),
				"message": []interface{}{float64(123)},
			},
		},
		{
			name:      "multiple error messages",
			errStatus: fiber.StatusInternalServerError,
			errMsgs:   []int{456, 789},
			want: fiber.Map{
				"success": false,
				"error":   float64(fiber.StatusInternalServerError),
				"message": []interface{}{float64(456), float64(789)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			ctx := app.AcquireCtx(&fasthttp.RequestCtx{})

			err := NewError(ctx, tt.errStatus, tt.errMsgs...)
			assert.NoError(t, err)
			assert.Equal(t, tt.errStatus, ctx.Response().StatusCode())

			var response fiber.Map
			body := ctx.Response().Body()
			err = json.Unmarshal(body, &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, response)

			app.ReleaseCtx(ctx)
		})
	}
}

func TestNewSuccess(t *testing.T) {
	tests := []struct {
		name    string
		details fiber.Map
		want    fiber.Map
	}{
		{
			name: "success with details",
			details: fiber.Map{
				"data": "test data",
			},
			want: fiber.Map{
				"success": true,
				"data":    "test data",
			},
		},
		{
			name:    "success without details",
			details: fiber.Map{},
			want: fiber.Map{
				"success": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			ctx := app.AcquireCtx(&fasthttp.RequestCtx{})

			err := NewSuccess(ctx, tt.details)
			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusOK, ctx.Response().StatusCode())

			var response fiber.Map
			body := ctx.Response().Body()
			err = json.Unmarshal(body, &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, response)

			app.ReleaseCtx(ctx)
		})
	}
}
