// Package cleanupchecks удаляет устаревшие результаты проверок.
// Запускается одноразовым процессом cmd/cleanup (фактор XII «Admin processes»).
package cleanupchecks

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// Request — глубина хранения и размер пачки удаления.
type Request struct {
	RetentionDays int
	BatchSize     int
	Now           time.Time
}

func (req *Request) validate() error {
	if req.RetentionDays <= 0 || req.BatchSize <= 0 {
		return domain.ErrInvalidPaging
	}

	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	return nil
}

// Result — сколько проверок удалено и до какой даты.
type Result struct {
	Deleted int64
	Before  time.Time
}

// Usecase — очистка устаревших проверок.
type Usecase struct {
	checks CheckRepo
}

// NewUsecase собирает usecase очистки.
func NewUsecase(checks CheckRepo) *Usecase {
	return &Usecase{checks: checks}
}

// Execute удаляет проверки старше retention пачками, пока они не закончатся.
// Пачками — чтобы не держать долгую блокировку на большой таблице.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	before := req.Now.AddDate(0, 0, -req.RetentionDays)
	result := &Result{Before: before}

	for {
		deleted, err := uc.checks.DeleteOlderThan(ctx, before, req.BatchSize)
		if err != nil {
			return nil, err
		}

		result.Deleted += deleted

		if deleted < int64(req.BatchSize) {
			return result, nil
		}

		if err := ctx.Err(); err != nil {
			return result, ctx.Err()
		}
	}
}
