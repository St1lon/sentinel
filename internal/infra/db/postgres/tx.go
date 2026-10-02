package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier — минимальный набор операций, нужный репозиториям.
// Его реализуют и пул, и активная транзакция, поэтому репозиторий не знает,
// работает он внутри транзакции или вне неё.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type txKey struct{}

// TxManager выполняет функцию в одной транзакции, прокладывая её через context.
type TxManager struct {
	pool *pgxpool.Pool
}

// NewTxManager создаёт менеджер транзакций над пулом.
func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// WithTx запускает fn в транзакции: при ошибке или панике — откат, иначе коммит.
// Вложенный вызов переиспользует уже открытую транзакцию из контекста.
func (m *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false

	defer func() {
		if committed {
			return
		}
		// Контекст может быть уже отменён, поэтому откат идёт на context.WithoutCancel.
		_ = tx.Rollback(context.WithoutCancel(ctx))
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	committed = true

	return nil
}

// GetQuerier возвращает транзакцию из контекста, если она есть, иначе пул.
func GetQuerier(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}

	return pool
}
