package cleanupchecks

import (
	"context"
	"time"
)

type CheckRepo interface {
	DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error)
}
