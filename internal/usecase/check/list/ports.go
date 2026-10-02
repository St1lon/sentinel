package listchecks

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

type CheckRepo interface {
	ListByMonitor(
		ctx context.Context, monitorID string, from, to time.Time, limit int,
	) ([]*domain.Check, error)
}
