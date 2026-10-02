package getstatuspage

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

const (
	defaultWindowDays = 90
	maxIncidents      = 50
)

type Request struct {
	Slug string
	Now  time.Time
}

func (req *Request) validate() error {
	if !validate.UUID(req.Slug) {
		return domain.ErrStatusPageNotFound
	}

	if req.Now.IsZero() {
		req.Now = time.Now().UTC()
	}

	return nil
}

type Result struct {
	From      time.Time
	To        time.Time
	Monitors  []*domain.Monitor
	Buckets   []*domain.Bucket
	Incidents []*domain.Incident
}

type Usecase struct {
	users     UserRepo
	monitors  MonitorRepo
	checks    CheckRepo
	incidents IncidentRepo
}

func NewUsecase(users UserRepo, monitors MonitorRepo, checks CheckRepo, incidents IncidentRepo) *Usecase {
	return &Usecase{users: users, monitors: monitors, checks: checks, incidents: incidents}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	user, err := uc.users.GetByStatusPageSlug(ctx, req.Slug)
	if err != nil {
		return nil, err
	}

	monitors, err := uc.monitors.ListPublicByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	to := req.Now.UTC()
	from := to.AddDate(0, 0, -defaultWindowDays)

	result := &Result{
		From:      from,
		To:        to,
		Monitors:  monitors,
		Buckets:   []*domain.Bucket{},
		Incidents: []*domain.Incident{},
	}

	if len(monitors) == 0 {
		return result, nil
	}

	ids := make([]string, 0, len(monitors))
	for _, monitor := range monitors {
		ids = append(ids, monitor.ID)
	}

	result.Buckets, err = uc.checks.BucketsForMonitors(ctx, ids, from, to, domain.BucketSizeDay)
	if err != nil {
		return nil, err
	}

	result.Incidents, err = uc.incidents.ListByMonitors(ctx, ids, from, maxIncidents)
	if err != nil {
		return nil, err
	}

	return result, nil
}
