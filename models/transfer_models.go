package models

type TransferRequestHeader struct {
	IdempotencyKey string `reqHeader:"Idempotency-Key"`
}

type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
}

type Transfer struct {
	From   string  `json:"from"`
	To     string  `json:"to"`
	Amount float64 `json:"amount"`
}

type TransferResponse struct {
	From          string  `json:"from"`
	To            string  `json:"to"`
	Amount        float64 `json:"amount"`
	TransactionID string  `json:"transaction_id"`
	Status        string  `json:"status"`
}
