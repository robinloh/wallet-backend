package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PgxInterface interface {
	Close(context.Context) error
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
}

type Postgres struct {
	Db PgxInterface
}

var (
	pgInstance *Postgres
	pgOnce     sync.Once
)

func ConnectDb(ctx context.Context) *Postgres {
	pgOnce.Do(func() {
		dataSource := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
			os.Getenv("POSTGRES_HOST"),
			os.Getenv("POSTGRES_USER"),
			os.Getenv("POSTGRES_PASSWORD"),
			os.Getenv("POSTGRES_DB"),
			os.Getenv("POSTGRES_PORT"),
		)

		db, err := pgx.Connect(ctx, dataSource)
		if err != nil {
			panic("unable to connect to database : " + err.Error())
		}

		t, err := db.LoadType(ctx, "txntype")
		if err != nil {
			panic("unable to load database table type : " + err.Error())
		}
		db.TypeMap().RegisterType(t)

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

type TestDatabase struct {
	Db        *pgx.Conn
	container *postgres.PostgresContainer
}

func SetupTestDatabase(t *testing.T, username string, password string, dbName string, host string) (*TestDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	_ = os.Setenv("POSTGRES_USER", username)
	_ = os.Setenv("POSTGRES_PASSWORD", password)
	_ = os.Setenv("POSTGRES_DB", dbName)
	_ = os.Setenv("POSTGRES_HOST", host)

	container, err := postgres.Run(
		ctx,
		"postgres:alpine",
		postgres.WithDatabase(os.Getenv("POSTGRES_DB")),
		postgres.WithUsername(os.Getenv("POSTGRES_USER")),
		postgres.WithPassword(os.Getenv("POSTGRES_PASSWORD")),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)

	testcontainers.CleanupContainer(t, container)

	if err != nil {
		return nil, fmt.Errorf("failed to run postgres container: %w", err)
	}

	p, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return nil, fmt.Errorf("unable to get mapped port : %v", err.Error())
	}

	_ = os.Setenv("POSTGRES_PORT", p.Port())

	dataSource := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)

	db, err := pgx.Connect(ctx, dataSource)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database : %v", err.Error())
	}

	goose.SetBaseFS(nil)

	if err := goose.SetDialect("pgx"); err != nil {
		return nil, fmt.Errorf("unable to set dialect : %v", err.Error())
	}

	database := stdlib.OpenDB(*db.Config())

	dirPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("unable to get current working directory : %v", err.Error())
	}

	if err := goose.Up(database, fmt.Sprintf("%s/migrations", filepath.Dir(dirPath))); err != nil {
		return nil, fmt.Errorf("unable to run migrations : %v", err.Error())
	}

	pgInstance = &Postgres{
		Db: db,
	}

	return &TestDatabase{
		Db:        db,
		container: container,
	}, nil
}
