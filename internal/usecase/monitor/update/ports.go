package updatemonitor

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
	Update(ctx context.Context, monitor *domain.Monitor) error
}

type IncidentRepo interface {
	ResolveOpen(ctx context.Context, monitorID string, resolvedAt time.Time) error
}
