package domain

import "time"

type Incident struct {
	ID         string
	MonitorID  string
	StartedAt  time.Time
	ResolvedAt *time.Time
	Cause      string
}

func (i *Incident) IsOpen() bool {
	return i.ResolvedAt == nil
}

func (i *Incident) Duration(now time.Time) time.Duration {
	if i.ResolvedAt != nil {
		return i.ResolvedAt.Sub(i.StartedAt)
	}

	return now.Sub(i.StartedAt)
}
