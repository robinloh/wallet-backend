package models

type Account struct {
	AccountID string `json:"account_id" gorm:"primaryKey"`
}

type AccountResponse struct {
	Account Account `json:"account"`
	Balance string  `json:"balance" gorm:"default:0.00"`
}
