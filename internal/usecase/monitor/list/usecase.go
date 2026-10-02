// Package listmonitors реализует постраничный список мониторов пользователя.
package listmonitors

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// Result — страница мониторов с метаданными пагинации.
type Result struct {
	Monitors []*domain.Monitor
	Total    int
	Limit    int
	Offset   int
}

// Usecase — список мониторов пользователя.
type Usecase struct {
	monitors MonitorRepo
}

// NewUsecase собирает usecase списка мониторов.
func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

// Execute возвращает страницу мониторов пользователя.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	monitors, total, err := uc.monitors.ListByUser(ctx, req.UserID, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	return &Result{Monitors: monitors, Total: total, Limit: req.Limit, Offset: req.Offset}, nil
}
