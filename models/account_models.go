package models

import "gorm.io/gorm"

type Account struct {
	gorm.Model
	AccountID string `json:"account_id" gorm:"primaryKey"`
}

type AccountResponse struct {
	Account Account `json:"account"`
	Balance string  `json:"balance" gorm:"default:0.00"`
}
