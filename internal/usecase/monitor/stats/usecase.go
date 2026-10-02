// Package monitorstats реализует расчёт статистики по монитору за период.
package monitorstats

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// Result — агрегат за период и свёрнутые корзины для графика.
type Result struct {
	Monitor *domain.Monitor
	Stats   *domain.MonitorStats
	Buckets []*domain.Bucket
}

// Usecase — статистика по монитору.
type Usecase struct {
	monitors MonitorRepo
	checks   CheckRepo
}

// NewUsecase собирает usecase статистики.
func NewUsecase(monitors MonitorRepo, checks CheckRepo) *Usecase {
	return &Usecase{monitors: monitors, checks: checks}
}

// Execute возвращает агрегат и корзины по монитору пользователя.
// Сырые проверки наружу не отдаются: это агрегирующий запрос в БД, поэтому
// при переезде таблицы checks в ClickHouse меняется только реализация порта.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitor, err := uc.monitors.GetByIDForUser(ctx, req.MonitorID, req.UserID)
	if err != nil {
		return nil, err
	}

	stats, err := uc.checks.Stats(ctx, monitor.ID, req.From, req.To)
	if err != nil {
		return nil, err
	}

	buckets, err := uc.checks.Buckets(ctx, monitor.ID, req.From, req.To, req.bucketSize)
	if err != nil {
		return nil, err
	}

	return &Result{Monitor: monitor, Stats: stats, Buckets: buckets}, nil
}
