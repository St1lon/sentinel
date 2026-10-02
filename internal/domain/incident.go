package domain

import "time"

// Incident — непрерывный период недоступности монитора.
// Инвариант "не больше одного открытого инцидента на монитор" обеспечен
// частичным уникальным индексом в БД, а не только кодом.
type Incident struct {
	ID         string
	MonitorID  string
	StartedAt  time.Time
	ResolvedAt *time.Time
	Cause      string
}

// IsOpen сообщает, что инцидент ещё не закрыт.
func (i *Incident) IsOpen() bool {
	return i.ResolvedAt == nil
}

// Duration возвращает длительность инцидента; для открытого — от начала до now.
func (i *Incident) Duration(now time.Time) time.Duration {
	if i.ResolvedAt != nil {
		return i.ResolvedAt.Sub(i.StartedAt)
	}

	return now.Sub(i.StartedAt)
}
