package models

type AccountRequest struct {
	Count int `json:"count"`
}

type AccountResponse struct {
	AccountID string `json:"account_id"`
	Balance   string `json:"balance"`
}
