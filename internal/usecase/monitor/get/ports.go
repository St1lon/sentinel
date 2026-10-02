package getmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error)
}
