package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type SLI struct {
	probes    *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	stepUp    *prometheus.GaugeVec
	journeyUp prometheus.Gauge
}

func NewSLI(reg prometheus.Registerer) *SLI {
	factory := promauto.With(reg)

	return &SLI{
		probes: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "sli_probe_total",
			Help:      "Шаги сценария пробера по результату.",
		}, []string{"step", "result"}),

		duration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "sli_probe_duration_seconds",
			Help:      "Время ответа API на шаг сценария пробера.",
			Buckets:   sloLatencyBuckets(),
		}, []string{"step"}),

		stepUp: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sli_step_success",
			Help:      "Результат последнего прогона шага: 1 — успех, 0 — отказ.",
		}, []string{"step"}),

		journeyUp: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "sli_journey_success",
			Help:      "Результат последнего прогона сценария целиком: 1 — успех, 0 — отказ.",
		}),
	}
}

// StepFinished записывает результат одного шага сценария.
func (s *SLI) StepFinished(step string, success bool, duration time.Duration) {
	s.probes.WithLabelValues(step, successResult(success)).Inc()
	s.duration.WithLabelValues(step).Observe(duration.Seconds())
	s.stepUp.WithLabelValues(step).Set(gaugeValue(success))
}

// JourneyFinished записывает результат сценария целиком: успех только если
// прошли все шаги.
func (s *SLI) JourneyFinished(success bool) {
	s.journeyUp.Set(gaugeValue(success))
}

func gaugeValue(success bool) float64 {
	if success {
		return 1
	}

	return 0
}
