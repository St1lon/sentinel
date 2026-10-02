package createmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
}
