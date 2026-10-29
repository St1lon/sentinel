package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP — метрики HTTP-слоя API. Реализует порт middleware.RequestRecorder.
type HTTP struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

func NewHTTP(reg prometheus.Registerer) *HTTP {
	factory := promauto.With(reg)

	return &HTTP{
		requests: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Обработанные HTTP-запросы по маршруту и коду ответа.",
		}, []string{"method", "route", "status"}),

		duration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Время обработки HTTP-запроса.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),

		inFlight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Запросы, обрабатываемые прямо сейчас.",
		}),
	}
}

func (h *HTTP) RequestStarted() {
	h.inFlight.Inc()
}

func (h *HTTP) RequestFinished(method, route string, status int, duration time.Duration) {
	h.inFlight.Dec()
	h.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	h.duration.WithLabelValues(method, route).Observe(duration.Seconds())
}
