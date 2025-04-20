package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
)

type Postgres struct {
	Db *pgx.Conn
}

var (
	pgInstance *Postgres
	pgOnce     sync.Once
)

func ConnectDb(ctx context.Context) *Postgres {
	dataSource := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)

	pgOnce.Do(func() {
		db, err := pgx.Connect(ctx, dataSource)
		if err != nil {
			panic("unable to connect to database : " + err.Error())
		}
		pgxdecimal.Register(db.TypeMap())
		pgInstance = &Postgres{
			Db: db,
		}
	})
	return pgInstance
}

func (p *Postgres) CloseDbConnection(ctx context.Context, logger *slog.Logger) {
	err := p.Db.Close(ctx)
	if err != nil {
		logger.Error("unable to close database connection : " + err.Error())
		os.Exit(1)
	}
}
