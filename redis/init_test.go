package redis

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectRedis(t *testing.T) {
	type args struct {
		logger *slog.Logger
	}
	tests := []struct {
		name    string
		args    args
		init    func()
		wantErr bool
		wantNil bool
	}{
		{
			name: "Valid logger, successful connection",
			args: args{
				logger: slog.Default(),
			},
			wantNil: false,
		},
		{
			name: "Nil logger, successful connection",
			args: args{
				logger: nil,
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SetupTestRedis(t)
			assert.Equal(t, tt.wantErr, err != nil)
			redis1 := ConnectRedis(tt.args.logger)
			redis2 := ConnectRedis(tt.args.logger)
			assert.Equal(t, redis1, redis2)
		})
	}
}
