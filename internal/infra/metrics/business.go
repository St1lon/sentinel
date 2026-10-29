package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/St1lon/sentinel/internal/domain"
)

const (
	resultSuccess = "success"
	resultFailure = "failure"
)

// Business — бизнес-метрики, которые видно из HTTP-транспорта: регистрации,
// входы и операции пользователя с мониторами. Реализует порт handlers.Metrics.
type Business struct {
	authAttempts    *prometheus.CounterVec
	monitorsCreated *prometheus.CounterVec
	monitorsDeleted prometheus.Counter
}

func NewBusiness(reg prometheus.Registerer) *Business {
	factory := promauto.With(reg)

	return &Business{
		authAttempts: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "auth_attempts_total",
			Help:      "Попытки регистрации и входа, разбитые по операции и результату.",
		}, []string{"operation", "result"}),

		monitorsCreated: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "monitors_created_total",
			Help:      "Созданные мониторы по виду проверки.",
		}, []string{"kind"}),

		monitorsDeleted: factory.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "monitors_deleted_total",
			Help:      "Удалённые мониторы.",
		}),
	}
}

func (b *Business) RegisterAttempt(success bool) {
	b.authAttempts.WithLabelValues("register", attemptResult(success)).Inc()
}

func (b *Business) LoginAttempt(success bool) {
	b.authAttempts.WithLabelValues("login", attemptResult(success)).Inc()
}

func (b *Business) MonitorCreated(kind domain.MonitorKind) {
	b.monitorsCreated.WithLabelValues(string(kind)).Inc()
}

func (b *Business) MonitorDeleted() {
	b.monitorsDeleted.Inc()
}

func attemptResult(success bool) string {
	if success {
		return resultSuccess
	}

	return resultFailure
}
