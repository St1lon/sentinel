package createmonitor

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type MonitorRepo interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
}

// TargetGuard — политика допустимых целей: схема, учётные данные в URL,
// приватные и служебные сети. Реализуется в infra/prober.
type TargetGuard interface {
	CheckTarget(target string) error
}
