package utils

import (
	"time"
)

const (
	FAILED    = "failed"
	COMPLETED = "completed"
)

func ConvertTimezone(timestamp time.Time) time.Time {
	l, _ := time.LoadLocation("Asia/Shanghai")
	return timestamp.In(l)
}
