package monitorstats

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт чтения монитора с проверкой владельца.
type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

// CheckRepo — порт агрегирующих выборок по проверкам.
type CheckRepo interface {
	Stats(ctx context.Context, monitorID string, from, to time.Time) (*domain.MonitorStats, error)
	Buckets(
		ctx context.Context, monitorID string, from, to time.Time, size domain.BucketSize,
	) ([]*domain.Bucket, error)
}
