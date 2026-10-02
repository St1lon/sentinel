package createmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// MonitorRepo — порт хранилища мониторов: созданию нужна только запись.
type MonitorRepo interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
}
