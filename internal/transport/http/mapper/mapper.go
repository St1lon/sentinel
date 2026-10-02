// Package mapper переводит доменные структуры в wire-DTO и обратно.
// Благодаря этому слою изменение схемы БД или домена не ломает контракт API.
package mapper

import (
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
)

// User переводит пользователя в DTO.
func User(user *domain.User) dto.User {
	return dto.User{
		ID:             user.ID,
		Email:          user.Email,
		StatusPageSlug: user.StatusPageSlug,
		CreatedAt:      user.CreatedAt,
	}
}

// Monitor переводит монитор в DTO.
func Monitor(monitor *domain.Monitor) dto.Monitor {
	return dto.Monitor{
		ID:                  monitor.ID,
		Name:                monitor.Name,
		Kind:                string(monitor.Kind),
		Target:              monitor.Target,
		Method:              monitor.Method,
		IntervalSeconds:     monitor.IntervalSeconds,
		TimeoutSeconds:      monitor.TimeoutSeconds,
		ExpectedStatus:      monitor.ExpectedStatus,
		FailureThreshold:    monitor.FailureThreshold,
		IsPublic:            monitor.IsPublic,
		Paused:              monitor.Paused,
		Status:              string(monitor.Status),
		ConsecutiveFailures: monitor.ConsecutiveFailures,
		LastCheckedAt:       monitor.LastCheckedAt,
		NextCheckAt:         monitor.NextCheckAt,
		CreatedAt:           monitor.CreatedAt,
		UpdatedAt:           monitor.UpdatedAt,
	}
}

// Monitors переводит список мониторов.
func Monitors(monitors []*domain.Monitor) []dto.Monitor {
	items := make([]dto.Monitor, 0, len(monitors))

	for _, monitor := range monitors {
		items = append(items, Monitor(monitor))
	}

	return items
}

// Check переводит результат проверки.
func Check(check *domain.Check) dto.Check {
	return dto.Check{
		ID:         check.ID,
		CheckedAt:  check.CheckedAt,
		Up:         check.Up,
		StatusCode: check.StatusCode,
		LatencyMS:  check.LatencyMS,
		Error:      check.Error,
	}
}

// Checks переводит список проверок.
func Checks(checks []*domain.Check) []dto.Check {
	items := make([]dto.Check, 0, len(checks))

	for _, check := range checks {
		items = append(items, Check(check))
	}

	return items
}

// Buckets переводит корзины графика.
func Buckets(buckets []*domain.Bucket) []dto.Bucket {
	items := make([]dto.Bucket, 0, len(buckets))

	for _, bucket := range buckets {
		items = append(items, dto.Bucket{
			BucketTime:  bucket.BucketTime,
			Total:       bucket.Total,
			Successful:  bucket.Successful,
			UptimeRatio: bucket.UptimeRatio(),
		})
	}

	return items
}

// MonitorStats переводит агрегат и корзины.
func MonitorStats(stats *domain.MonitorStats, buckets []*domain.Bucket) dto.MonitorStats {
	return dto.MonitorStats{
		MonitorID:        stats.MonitorID,
		From:             stats.From,
		To:               stats.To,
		TotalChecks:      stats.TotalChecks,
		SuccessfulChecks: stats.SuccessfulChecks,
		UptimeRatio:      stats.UptimeRatio,
		AvgLatencyMS:     stats.AvgLatencyMS,
		P95LatencyMS:     stats.P95LatencyMS,
		LastCheckedAt:    stats.LastCheckedAt,
		Buckets:          Buckets(buckets),
	}
}

// Incident переводит инцидент.
func Incident(incident *domain.Incident, now time.Time) dto.Incident {
	return dto.Incident{
		ID:              incident.ID,
		MonitorID:       incident.MonitorID,
		StartedAt:       incident.StartedAt,
		ResolvedAt:      incident.ResolvedAt,
		Cause:           incident.Cause,
		IsOpen:          incident.IsOpen(),
		DurationSeconds: int64(incident.Duration(now).Seconds()),
	}
}

// Incidents переводит список инцидентов.
func Incidents(incidents []*domain.Incident, now time.Time) []dto.Incident {
	items := make([]dto.Incident, 0, len(incidents))

	for _, incident := range incidents {
		items = append(items, Incident(incident, now))
	}

	return items
}
