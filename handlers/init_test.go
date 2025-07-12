package handlers

import (
	"log/slog"
	"testing"

	"github.com/robinloh/wallet-backend/database"
	"github.com/robinloh/wallet-backend/redis"
	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	logger := slog.Default()
	postgres := &database.Postgres{}
	redisCache := &redis.Redis{}

	type args struct {
		logger     *slog.Logger
		postgresDB *database.Postgres
		cache      *redis.Redis
	}
	tests := []struct {
		name string
		args args
		want APIs
	}{
		{
			name: "successful initialization",
			args: args{
				logger:     logger,
				postgresDB: postgres,
				cache:      redisCache,
			},
			want: &accountsHandler{
				logger:     logger,
				postgresDB: postgres,
				redis:      redisCache,
			},
		},
		{
			name: "nil logger",
			args: args{
				logger:     nil,
				postgresDB: postgres,
				cache:      redisCache,
			},
			want: &accountsHandler{
				postgresDB: postgres,
				redis:      redisCache,
			},
		},
		{
			name: "nil postgres",
			args: args{
				logger:     logger,
				postgresDB: nil,
				cache:      redisCache,
			},
			want: &accountsHandler{
				logger: logger,
				redis:  redisCache,
			},
		},
		{
			name: "nil cache",
			args: args{
				logger:     logger,
				postgresDB: postgres,
				cache:      nil,
			},
			want: &accountsHandler{
				logger:     logger,
				postgresDB: postgres,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, Initialize(tt.args.logger, tt.args.postgresDB, tt.args.cache), "Initialize(%v, %v, %v)", tt.args.logger, tt.args.postgresDB, tt.args.cache)
		})
	}
}
