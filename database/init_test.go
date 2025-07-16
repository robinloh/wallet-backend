package database

import (
	"context"
	"log/slog"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestConnectDb(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		password string
		dbName   string
		host     string
	}
	tests := []struct {
		name          string
		args          args
		expectedDB    bool
		expectedErr   bool
		expectedPanic bool
	}{
		{
			name:       "Success connect to DB",
			expectedDB: true,
			args: args{
				ctx:      context.Background(),
				username: "test_user",
				password: "test_password",
				dbName:   "test_wallet_backend_db",
				host:     "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedPanic {
				assert.Panics(t, func() {
					_ = ConnectDb(tt.args.ctx)
				})
				return
			}

			td, setupErr := SetupTestDatabase(t, tt.args.username, tt.args.password, tt.args.dbName, tt.args.host)
			assert.Equal(t, tt.expectedDB, td != nil && td.Db != nil)
			assert.Equal(t, tt.expectedErr, setupErr != nil)

			actualDB1 := ConnectDb(tt.args.ctx)
			actualDB2 := ConnectDb(tt.args.ctx) // should return the same instance

			assert.Equal(t, actualDB1.Db, actualDB2.Db)

			actualDB1.CloseDbConnection(tt.args.ctx, slog.Default())
			err := td.container.Terminate(tt.args.ctx)
			assert.NoError(t, err)
		})
	}
}

func TestPostgres_CloseDbConnection(t *testing.T) {
	type fields struct {
		Db *Postgres
	}
	type args struct {
		ctx    context.Context
		logger *slog.Logger
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected bool
	}{
		{
			name:     "Success closing of DB connection",
			expected: true,
			args: args{
				ctx:    context.Background(),
				logger: slog.Default(),
			},
			fields: fields{
				Db: func() *Postgres {
					mock, err := pgxmock.NewConn()
					if err != nil {
						t.Fatalf("Failed to create pgxmock: %v", err)
					}
					mock.ExpectClose()
					return &Postgres{
						Db: mock,
					}
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Postgres{
				Db: tt.fields.Db.Db,
			}
			p.CloseDbConnection(tt.args.ctx, tt.args.logger)
		})
	}
}
