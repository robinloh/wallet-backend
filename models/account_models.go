package models

type AccountRequest struct {
	Count int `json:"count"`
}

type AccountResponse struct {
	ID      string `json:"id"`
	Balance string `json:"balance"`
}
