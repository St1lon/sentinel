// Package common содержит порты и типы, переиспользуемые несколькими usecase.
package common

import "context"

// TxManager — порт для выполнения нескольких операций репозиториев
// в одной транзакции. Объявлен в common, потому что нужен более чем одному
// usecase; конкретная реализация живёт в infra/db/postgres.
type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
