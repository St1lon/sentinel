package recordprobe

import (
	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — монитор и результат его проверки.
type Request struct {
	Monitor *domain.Monitor
	Check   *domain.Check
}

func (req *Request) validate() error {
	if req.Monitor == nil || !validate.UUID(req.Monitor.ID) {
		return domain.ErrMonitorNotFound
	}

	if req.Check == nil {
		return domain.ErrInvalidTarget
	}

	if req.Check.MonitorID == "" {
		req.Check.MonitorID = req.Monitor.ID
	}

	if req.Check.MonitorID != req.Monitor.ID {
		return domain.ErrMonitorNotFound
	}

	if req.Check.CheckedAt.IsZero() {
		return domain.ErrInvalidTimeRange
	}

	return nil
}
