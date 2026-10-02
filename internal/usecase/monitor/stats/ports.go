package monitorstats

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

type CheckRepo interface {
	Stats(ctx context.Context, monitorID string, from, to time.Time) (*domain.MonitorStats, error)
	Buckets(
		ctx context.Context, monitorID string, from, to time.Time, size domain.BucketSize,
	) ([]*domain.Bucket, error)
}
