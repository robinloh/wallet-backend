package models

import "time"

type AccountTransactionsRequest struct {
	AccountID string `json:"account_id"`
}

type AccountTransactionsResponse struct {
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Amount        float64   `json:"amount"`
	TxnType       string    `json:"txntype"`
	SenderID      string    `json:"sender_id"`
	ReceiverID    string    `json:"receiver_id"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
}
