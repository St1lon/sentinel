package mapper

import (
	"time"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
)

const (
	overallOperational = "operational"
	overallDegraded    = "degraded"
	overallDown        = "down"
	overallUnknown     = "unknown"
)

func StatusPage(
	from, to time.Time,
	monitors []*domain.Monitor,
	buckets []*domain.Bucket,
	incidents []*domain.Incident,
	now time.Time,
) dto.StatusPage {
	bucketsByMonitor := make(map[string][]*domain.Bucket, len(monitors))
	for _, bucket := range buckets {
		bucketsByMonitor[bucket.MonitorID] = append(bucketsByMonitor[bucket.MonitorID], bucket)
	}

	namesByMonitor := make(map[string]string, len(monitors))
	services := make([]dto.StatusPageService, 0, len(monitors))

	for _, monitor := range monitors {
		namesByMonitor[monitor.ID] = monitor.Name
		monitorBuckets := bucketsByMonitor[monitor.ID]

		services = append(services, dto.StatusPageService{
			Name:        monitor.Name,
			Status:      string(monitor.Status),
			UptimeRatio: uptimeOverBuckets(monitorBuckets),
			Buckets:     Buckets(monitorBuckets),
			LastCheckAt: monitor.LastCheckedAt,
		})
	}

	return dto.StatusPage{
		From:      from,
		To:        to,
		Overall:   overallStatus(monitors),
		Services:  services,
		Incidents: statusPageIncidents(incidents, namesByMonitor, now),
	}
}

func statusPageIncidents(
	incidents []*domain.Incident, names map[string]string, now time.Time,
) []dto.StatusPageIncident {
	items := make([]dto.StatusPageIncident, 0, len(incidents))

	for _, incident := range incidents {
		items = append(items, dto.StatusPageIncident{
			Service:         names[incident.MonitorID],
			StartedAt:       incident.StartedAt,
			ResolvedAt:      incident.ResolvedAt,
			Cause:           incident.Cause,
			DurationSeconds: int64(incident.Duration(now).Seconds()),
		})
	}

	return items
}

func uptimeOverBuckets(buckets []*domain.Bucket) float64 {
	var total, successful int

	for _, bucket := range buckets {
		total += bucket.Total
		successful += bucket.Successful
	}

	if total == 0 {
		return 0
	}

	return float64(successful) / float64(total)
}

func overallStatus(monitors []*domain.Monitor) string {
	if len(monitors) == 0 {
		return overallUnknown
	}

	pending := 0

	for _, monitor := range monitors {
		switch monitor.Status {
		case domain.MonitorStatusDown:
			return overallDown
		case domain.MonitorStatusPending, domain.MonitorStatusPaused:
			pending++
		case domain.MonitorStatusUp:
		}
	}

	if pending == len(monitors) {
		return overallUnknown
	}

	if pending > 0 {
		return overallDegraded
	}

	return overallOperational
}
