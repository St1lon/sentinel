package listchecks

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт чтения монитора с проверкой владельца.
type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}

// CheckRepo — порт чтения проверок монитора.
type CheckRepo interface {
	ListByMonitor(
		ctx context.Context, monitorID string, from, to time.Time, limit int,
	) ([]*domain.Check, error)
}
