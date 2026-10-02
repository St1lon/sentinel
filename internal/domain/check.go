package domain

import "time"

// Check — результат одной проверки. Записи неизменяемы: только INSERT и чтение
// агрегатами. Именно этот поток в Phase 2 переезжает в ClickHouse.
type Check struct {
	ID         int64
	MonitorID  string
	CheckedAt  time.Time
	Up         bool
	StatusCode *int
	LatencyMS  int
	Error      *string
}

// MonitorStats — агрегат по проверкам за период.
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

// Bucket — свёрнутый интервал для графика аптайма.
type Bucket struct {
	MonitorID  string
	BucketTime time.Time
	Total      int
	Successful int
}

// BucketSize — допустимый размер корзины агрегации.
type BucketSize string

const (
	BucketSizeHour BucketSize = "hour"
	BucketSizeDay  BucketSize = "day"
)

// Valid проверяет, что размер корзины входит в белый список.
// Значение подставляется в date_trunc, поэтому белый список обязателен.
func (b BucketSize) Valid() bool {
	return b == BucketSizeHour || b == BucketSizeDay
}

// UptimeRatio — доля успешных проверок в корзине, 0..1.
func (b *Bucket) UptimeRatio() float64 {
	if b.Total == 0 {
		return 0
	}

	return float64(b.Successful) / float64(b.Total)
}
