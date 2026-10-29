package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/St1lon/sentinel/internal/domain"
)

// Fleet — текущее состояние всех мониторов сразу. В отличие от счётчиков это
// gauge: значение не накапливается, а снимается целиком. Реализует порт
// worker.FleetMetrics.
type Fleet struct {
	monitors      *prometheus.GaugeVec
	openIncidents prometheus.Gauge
}

func NewFleet(reg prometheus.Registerer) *Fleet {
	factory := promauto.With(reg)

	return &Fleet{
		monitors: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "monitors",
			Help:      "Сколько мониторов сейчас в каждом статусе.",
		}, []string{"status"}),

		openIncidents: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "incidents_open",
			Help:      "Сколько инцидентов открыто прямо сейчас.",
		}),
	}
}

func (f *Fleet) SetFleetSummary(summary *domain.FleetSummary) {
	// Статусы перечисляются все: иначе gauge статуса, в котором не осталось
	// мониторов, застынет на последнем ненулевом значении.
	for _, status := range domain.KnownMonitorStatuses() {
		f.monitors.WithLabelValues(string(status)).Set(float64(summary.ByStatus[status]))
	}

	f.openIncidents.Set(float64(summary.OpenIncidents))
}
