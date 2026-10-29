package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Cleanup — метрики разовой задачи очистки. Gauge, а не счётчики: процесс
// живёт один запуск, и счётчик каждый раз начинался бы с нуля. Отдаётся
// результат последнего запуска и время его успешного завершения.
type Cleanup struct {
	deleted     prometheus.Gauge
	duration    prometheus.Gauge
	lastSuccess prometheus.Gauge
}

func NewCleanup(reg prometheus.Registerer) *Cleanup {
	factory := promauto.With(reg)

	return &Cleanup{
		deleted: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cleanup_checks_deleted",
			Help:      "Сколько записей проверок удалил последний запуск очистки.",
		}),

		duration: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cleanup_duration_seconds",
			Help:      "Сколько работал последний запуск очистки.",
		}),

		lastSuccess: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "cleanup_last_success_timestamp_seconds",
			Help:      "Время последнего успешного завершения очистки, unix-секунды.",
		}),
	}
}

func (c *Cleanup) Finished(deleted int64, duration time.Duration, at time.Time) {
	c.deleted.Set(float64(deleted))
	c.duration.Set(duration.Seconds())
	c.lastSuccess.Set(float64(at.Unix()))
}
