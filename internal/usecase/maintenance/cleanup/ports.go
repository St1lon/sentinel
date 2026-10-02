package cleanupchecks

import (
	"context"
	"time"
)

// CheckRepo — порт пакетного удаления устаревших проверок.
type CheckRepo interface {
	DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error)
}
