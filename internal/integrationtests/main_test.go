//go:build integration

// Package integrationtests проверяет репозитории против настоящей PostgreSQL,
// поднятой в контейнере. Эти тесты отделены билд-тегом integration, поэтому
// обычный `go test ./...` их не запускает и не требует Docker.
package integrationtests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	startupTimeout = 90 * time.Second
	postgresImage  = "postgres:17-alpine"
)

// testPool — общий пул на весь пакет: контейнер поднимается один раз,
// изоляцию между тестами даёт уникальность данных, а не пересоздание БД.
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, pool, err := setup(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %v\n", err)
		os.Exit(1)
	}

	testPool = pool

	code := m.Run()

	pool.Close()

	if err := testcontainers.TerminateContainer(container); err != nil {
		fmt.Fprintf(os.Stderr, "terminate container: %v\n", err)
	}

	os.Exit(code)
}

func setup(ctx context.Context) (testcontainers.Container, *pgxpool.Pool, error) {
	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase("sentinel_test"),
		tcpostgres.WithUsername("sentinel"),
		tcpostgres.WithPassword("sentinel"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(startupTimeout),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("run postgres container: %w", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return container, nil, fmt.Errorf("connection string: %w", err)
	}

	// Схема накатывается теми же миграциями, что в проде, — иначе тесты
	// проверяли бы структуру, которой в проде нет (фактор X).
	if err := applyMigrations(dsn); err != nil {
		return container, nil, err
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return container, nil, fmt.Errorf("create pool: %w", err)
	}

	return container, pool, nil
}

func applyMigrations(dsn string) error {
	path, err := filepath.Abs(filepath.Join("..", "infra", "db", "postgres", "migrations"))
	if err != nil {
		return fmt.Errorf("resolve migrations path: %w", err)
	}

	migrator, err := migrate.New("file://"+filepath.ToSlash(path), dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	defer func() {
		sourceErr, dbErr := migrator.Close()
		if sourceErr != nil {
			fmt.Fprintf(os.Stderr, "close migrate source: %v\n", sourceErr)
		}

		if dbErr != nil {
			fmt.Fprintf(os.Stderr, "close migrate db: %v\n", dbErr)
		}
	}()

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
