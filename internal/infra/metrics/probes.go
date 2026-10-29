package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/St1lon/sentinel/internal/domain"
)

// Шкала гистограммы задержек: от 10 мс до ~38 с. Проверки, упавшие
// по таймауту, попадают в +Inf — так и надо.
const (
	latencyBucketStart  = 0.01
	latencyBucketFactor = 2.5
	latencyBucketCount  = 10
)

const (
	probeResultUp   = "up"
	probeResultDown = "down"
)

// Probes — бизнес-метрики воркера проверок. Реализует порт worker.ProbeMetrics.
type Probes struct {
	probes            *prometheus.CounterVec
	latency           *prometheus.HistogramVec
	incidentsOpened   *prometheus.CounterVec
	incidentsResolved *prometheus.CounterVec
}

func NewProbes(reg prometheus.Registerer) *Probes {
	factory := promauto.With(reg)

	return &Probes{
		probes: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "probes_total",
			Help:      "Выполненные проверки мониторов по виду и результату.",
		}, []string{"kind", "result"}),

		latency: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "probe_latency_seconds",
			Help:      "Время ответа проверяемой цели.",
			Buckets: prometheus.ExponentialBuckets(
				latencyBucketStart, latencyBucketFactor, latencyBucketCount,
			),
		}, []string{"kind"}),

		incidentsOpened: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "incidents_opened_total",
			Help:      "Открытые инциденты: цель упала и порог подряд неудач пройден.",
		}, []string{"kind"}),

		incidentsResolved: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "incidents_resolved_total",
			Help:      "Закрытые инциденты: цель снова отвечает.",
		}, []string{"kind"}),
	}
}

func (p *Probes) ProbeRecorded(kind domain.MonitorKind, up bool, latency time.Duration) {
	p.probes.WithLabelValues(string(kind), probeResult(up)).Inc()
	p.latency.WithLabelValues(string(kind)).Observe(latency.Seconds())
}

func (p *Probes) IncidentOpened(kind domain.MonitorKind) {
	p.incidentsOpened.WithLabelValues(string(kind)).Inc()
}

func (p *Probes) IncidentResolved(kind domain.MonitorKind) {
	p.incidentsResolved.WithLabelValues(string(kind)).Inc()
}

func probeResult(up bool) string {
	if up {
		return probeResultUp
	}

	return probeResultDown
}
