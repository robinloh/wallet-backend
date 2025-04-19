package database

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Db *pgxpool.Pool
}

var (
	pgInstance *Postgres
	pgOnce     sync.Once
)

func ConnectDb() *Postgres {
	dataSource := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)

	pgOnce.Do(func() {
		db, err := pgxpool.New(context.Background(), dataSource)
		if err != nil {
			panic("unable to connect to database : " + err.Error())
		}
		pgInstance = &Postgres{
			Db: db,
		}
	})
	return pgInstance
}

func (p *Postgres) CloseDbConnection() {
	p.Db.Close()
}
