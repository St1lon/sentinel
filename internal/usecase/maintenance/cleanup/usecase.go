package cleanupchecks

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

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

type Result struct {
	Deleted int64
	Before  time.Time
}

type Usecase struct {
	checks CheckRepo
}

func NewUsecase(checks CheckRepo) *Usecase {
	return &Usecase{checks: checks}
}

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
