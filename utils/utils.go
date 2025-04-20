package utils

import (
	"time"

	"github.com/google/uuid"
)

const (
	SENDER = "sender"
)

const (
	FAILED    = "failed"
	COMPLETED = "completed"
)

func GenerateTxnID() (string, error) {
	txnID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return txnID.String(), err
}

func ConvertTimezone(timestamp time.Time) time.Time {
	l, _ := time.LoadLocation("Asia/Shanghai")
	return timestamp.In(l)
}
