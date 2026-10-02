package listincidents

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

type IncidentRepo interface {
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]*domain.Incident, int, error)
}
