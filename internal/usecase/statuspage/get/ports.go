package getstatuspage

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type UserRepo interface {
	GetByStatusPageSlug(ctx context.Context, slug string) (*domain.User, error)
}

type MonitorRepo interface {
	ListPublicByUser(ctx context.Context, userID string) ([]*domain.Monitor, error)
}

type CheckRepo interface {
	BucketsForMonitors(
		ctx context.Context, monitorIDs []string, from, to time.Time, size domain.BucketSize,
	) ([]*domain.Bucket, error)
}

type IncidentRepo interface {
	ListByMonitors(
		ctx context.Context, monitorIDs []string, since time.Time, limit int,
	) ([]*domain.Incident, error)
}
