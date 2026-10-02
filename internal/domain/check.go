package domain

import "time"

type Check struct {
	ID         int64
	MonitorID  string
	CheckedAt  time.Time
	Up         bool
	StatusCode *int
	LatencyMS  int
	Error      *string
}

type MonitorStats struct {
	MonitorID        string
	From             time.Time
	To               time.Time
	TotalChecks      int
	SuccessfulChecks int
	UptimeRatio      float64
	AvgLatencyMS     float64
	P95LatencyMS     int
	LastCheckedAt    *time.Time
}

type Bucket struct {
	MonitorID  string
	BucketTime time.Time
	Total      int
	Successful int
}

type BucketSize string

const (
	BucketSizeHour BucketSize = "hour"
	BucketSizeDay  BucketSize = "day"
)

func (b BucketSize) Valid() bool {
	return b == BucketSizeHour || b == BucketSizeDay
}

func (b *Bucket) UptimeRatio() float64 {
	if b.Total == 0 {
		return 0
	}

	return float64(b.Successful) / float64(b.Total)
}
