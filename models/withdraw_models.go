package models

type WithdrawRequest struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

type Withdraw struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
}

type WithdrawResponse struct {
	AccountID     string  `json:"account_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	TransactionID string  `json:"transaction_id"`
}
