package models

type DepositRequest struct {
	ID     string `json:"id"`
	Amount string `json:"amount"`
}

type Deposit struct {
	ID     string  `json:"id"`
	Amount float64 `json:"amount"`
}

type DepositResponse struct {
	AccountID     string  `json:"account_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	TransactionID string  `json:"transaction_id"`
}
