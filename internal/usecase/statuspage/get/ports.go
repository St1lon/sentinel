package getstatuspage

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// UserRepo — порт поиска владельца страницы по публичному слагу.
type UserRepo interface {
	GetByStatusPageSlug(ctx context.Context, slug string) (*domain.User, error)
}

// MonitorRepo — порт чтения публичных мониторов пользователя.
type MonitorRepo interface {
	ListPublicByUser(ctx context.Context, userID string) ([]*domain.Monitor, error)
}

// CheckRepo — порт свёрнутых корзин сразу по нескольким мониторам.
type CheckRepo interface {
	BucketsForMonitors(
		ctx context.Context, monitorIDs []string, from, to time.Time, size domain.BucketSize,
	) ([]*domain.Bucket, error)
}

// IncidentRepo — порт ленты инцидентов сразу по нескольким мониторам.
type IncidentRepo interface {
	ListByMonitors(
		ctx context.Context, monitorIDs []string, since time.Time, limit int,
	) ([]*domain.Incident, error)
}
