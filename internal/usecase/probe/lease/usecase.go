// Package leasemonitors забирает в работу мониторы, которым пора проверяться.
package leasemonitors

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// Request — размер пачки и момент времени, относительно которого ищутся
// просроченные мониторы (передаётся явно, чтобы usecase оставался тестируемым).
type Request struct {
	BatchSize int
	Now       time.Time
}

func (req *Request) validate() error {
	if req.BatchSize <= 0 {
		return domain.ErrInvalidPaging
	}

	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	return nil
}

// Usecase — выдача пачки мониторов воркеру.
type Usecase struct {
	monitors MonitorRepo
}

// NewUsecase собирает usecase выдачи мониторов.
func NewUsecase(monitors MonitorRepo) *Usecase {
	return &Usecase{monitors: monitors}
}

// Execute забирает мониторы в работу.
//
// Атомарность и неповторяемость выдачи обеспечивает репозиторий
// (FOR UPDATE SKIP LOCKED + сдвиг next_check_at одним запросом), поэтому
// несколько экземпляров воркера можно запускать без координатора и без лидера.
func (uc *Usecase) Execute(ctx context.Context, req *Request) ([]*domain.Monitor, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.monitors.LeaseDue(ctx, req.Now, req.BatchSize)
}
