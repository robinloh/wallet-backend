package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConvertTimezone(t *testing.T) {
	tests := []struct {
		name      string
		inputTime time.Time
		want      time.Time
	}{
		{
			name:      "convert UTC to Asia/Shanghai",
			inputTime: time.Date(2025, 7, 12, 12, 0, 0, 0, time.UTC),
			want:      time.Date(2025, 7, 12, 20, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertTimezone(tt.inputTime)
			assert.Equal(t, tt.want.Hour(), got.Hour())
			assert.Equal(t, tt.want.Minute(), got.Minute())
			assert.Equal(t, tt.want.Second(), got.Second())
			assert.Equal(t, tt.want.Year(), got.Year())
			assert.Equal(t, tt.want.Month(), got.Month())
			assert.Equal(t, tt.want.Day(), got.Day())
			_, wantOffset := tt.want.Zone()
			_, gotOffset := got.Zone()
			assert.Equal(t, wantOffset, gotOffset)
		})
	}
}
