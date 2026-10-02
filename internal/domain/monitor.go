package domain

import "time"

// Границы значений монитора. Дублируются CHECK-ограничениями в миграции:
// домен валидирует до записи, БД гарантирует инвариант на уровне хранения.
const (
	MinIntervalSeconds = 10
	MaxIntervalSeconds = 86400
	MinTimeoutSeconds  = 1
	MaxTimeoutSeconds  = 120
	MinThreshold       = 1
	MaxThreshold       = 10
	MaxNameLength      = 100
	MaxTargetLength    = 2048

	DefaultExpectedStatus   = 200
	DefaultFailureThreshold = 2
	DefaultIntervalSeconds  = 60
	DefaultTimeoutSeconds   = 10
	DefaultMethod           = "GET"
)

// MonitorKind — вид проверки. Enum заведён с запасом: tls_cert и tcp_port
// добавляются новым прober'ом без изменения схемы БД и контракта API.
type MonitorKind string

const (
	MonitorKindHTTP    MonitorKind = "http"
	MonitorKindTLSCert MonitorKind = "tls_cert"
	MonitorKindTCPPort MonitorKind = "tcp_port"
)

// MonitorStatus — текущее состояние монитора, вычисляется воркером.
type MonitorStatus string

const (
	// MonitorStatusPending — монитор создан, но ещё ни разу не проверялся.
	MonitorStatusPending MonitorStatus = "pending"
	MonitorStatusUp      MonitorStatus = "up"
	MonitorStatusDown    MonitorStatus = "down"
	MonitorStatusPaused  MonitorStatus = "paused"
)

// Monitor — наблюдаемая цель.
type Monitor struct {
	ID                  string
	UserID              string
	Name                string
	Kind                MonitorKind
	Target              string
	Method              string
	IntervalSeconds     int
	TimeoutSeconds      int
	ExpectedStatus      int
	FailureThreshold    int
	IsPublic            bool
	Paused              bool
	Status              MonitorStatus
	ConsecutiveFailures int
	LastCheckedAt       *time.Time
	NextCheckAt         time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// KnownMonitorKinds — список видов проверок, поддерживаемых текущей сборкой.
func KnownMonitorKinds() []MonitorKind {
	return []MonitorKind{MonitorKindHTTP, MonitorKindTLSCert, MonitorKindTCPPort}
}

// IsImplemented сообщает, есть ли в текущей сборке prober для этого вида проверки.
func (k MonitorKind) IsImplemented() bool {
	return k == MonitorKindHTTP
}

// Timeout возвращает таймаут одной проверки как time.Duration.
func (m *Monitor) Timeout() time.Duration {
	return time.Duration(m.TimeoutSeconds) * time.Second
}

// Interval возвращает интервал между проверками как time.Duration.
func (m *Monitor) Interval() time.Duration {
	return time.Duration(m.IntervalSeconds) * time.Second
}
