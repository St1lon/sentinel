package domain

import "time"

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

type MonitorKind string

const (
	MonitorKindHTTP    MonitorKind = "http"
	MonitorKindTLSCert MonitorKind = "tls_cert"
	MonitorKindTCPPort MonitorKind = "tcp_port"
)

type MonitorStatus string

const (
	MonitorStatusPending MonitorStatus = "pending"
	MonitorStatusUp      MonitorStatus = "up"
	MonitorStatusDown    MonitorStatus = "down"
	MonitorStatusPaused  MonitorStatus = "paused"
)

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

func KnownMonitorKinds() []MonitorKind {
	return []MonitorKind{MonitorKindHTTP, MonitorKindTLSCert, MonitorKindTCPPort}
}

func (k MonitorKind) IsImplemented() bool {
	return k == MonitorKindHTTP
}

func (m *Monitor) Timeout() time.Duration {
	return time.Duration(m.TimeoutSeconds) * time.Second
}

func (m *Monitor) Interval() time.Duration {
	return time.Duration(m.IntervalSeconds) * time.Second
}
