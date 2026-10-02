package monitorstats

import (
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

type Request struct {
	UserID    string
	MonitorID string
	From      time.Time
	To        time.Time
	Bucket    string

	bucketSize domain.BucketSize
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	if !validate.UUID(req.MonitorID) {
		return domain.ErrMonitorNotFound
	}

	if err := validate.TimeRange(req.From, req.To); err != nil {
		return err
	}

	if req.Bucket == "" {
		req.Bucket = string(domain.BucketSizeHour)
	}

	req.bucketSize = domain.BucketSize(req.Bucket)

	if !req.bucketSize.Valid() {
		return domain.ErrInvalidBucket
	}

	return nil
}
