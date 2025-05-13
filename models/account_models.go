package models

type AccountRequestHeader struct {
	IdempotencyKey string `reqHeader:"Idempotency-Key"`
}

type AccountRequest struct {
	Count int `json:"count"`
}

type AccountResponse struct {
	ID      string `json:"id"`
	Balance string `json:"balance"`
}

type GetAccountBalanceRequest struct {
	Id string `json:"id"`
}
