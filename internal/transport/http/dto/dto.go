// Package dto описывает wire-формат HTTP API. Доменные структуры наружу
// не отдаются никогда: между ними и этими типами стоит пакет mapper.
package dto

import "time"

// ErrorResponse — единый формат ошибки API.
type ErrorResponse struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// RegisterRequest — тело запроса регистрации.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest — тело запроса входа.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse — выданный токен и его владелец.
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// User — публичное представление пользователя. Хеш пароля здесь отсутствует
// по построению: его просто нет в структуре.
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	StatusPageSlug string    `json:"status_page_slug"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateMonitorRequest — тело создания монитора.
// Необязательные поля — указатели: так отличается «не передано» от нуля.
type CreateMonitorRequest struct {
	Name             string `json:"name"`
	Kind             string `json:"kind,omitempty"`
	Target           string `json:"target"`
	Method           string `json:"method,omitempty"`
	IntervalSeconds  *int   `json:"interval_seconds,omitempty"`
	TimeoutSeconds   *int   `json:"timeout_seconds,omitempty"`
	ExpectedStatus   *int   `json:"expected_status,omitempty"`
	FailureThreshold *int   `json:"failure_threshold,omitempty"`
	IsPublic         *bool  `json:"is_public,omitempty"`
	Paused           *bool  `json:"paused,omitempty"`
}

// UpdateMonitorRequest — тело частичного обновления монитора.
type UpdateMonitorRequest struct {
	Name             *string `json:"name,omitempty"`
	Target           *string `json:"target,omitempty"`
	Method           *string `json:"method,omitempty"`
	IntervalSeconds  *int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds   *int    `json:"timeout_seconds,omitempty"`
	ExpectedStatus   *int    `json:"expected_status,omitempty"`
	FailureThreshold *int    `json:"failure_threshold,omitempty"`
	IsPublic         *bool   `json:"is_public,omitempty"`
	Paused           *bool   `json:"paused,omitempty"`
}

// Monitor — представление монитора в API.
type Monitor struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Kind                string     `json:"kind"`
	Target              string     `json:"target"`
	Method              string     `json:"method"`
	IntervalSeconds     int        `json:"interval_seconds"`
	TimeoutSeconds      int        `json:"timeout_seconds"`
	ExpectedStatus      int        `json:"expected_status"`
	FailureThreshold    int        `json:"failure_threshold"`
	IsPublic            bool       `json:"is_public"`
	Paused              bool       `json:"paused"`
	Status              string     `json:"status"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	NextCheckAt         time.Time  `json:"next_check_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// MonitorList — страница мониторов.
type MonitorList struct {
	Items  []Monitor `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// Check — результат одной проверки.
type Check struct {
	ID         int64     `json:"id"`
	CheckedAt  time.Time `json:"checked_at"`
	Up         bool      `json:"up"`
	StatusCode *int      `json:"status_code"`
	LatencyMS  int       `json:"latency_ms"`
	Error      *string   `json:"error"`
}

// CheckList — проверки монитора за период.
type CheckList struct {
	MonitorID string  `json:"monitor_id"`
	Items     []Check `json:"items"`
}

// Bucket — свёрнутый интервал для графика.
type Bucket struct {
	BucketTime  time.Time `json:"bucket"`
	Total       int       `json:"total"`
	Successful  int       `json:"successful"`
	UptimeRatio float64   `json:"uptime_ratio"`
}

// MonitorStats — агрегат по монитору за период.
type MonitorStats struct {
	MonitorID        string     `json:"monitor_id"`
	From             time.Time  `json:"from"`
	To               time.Time  `json:"to"`
	TotalChecks      int        `json:"total_checks"`
	SuccessfulChecks int        `json:"successful_checks"`
	UptimeRatio      float64    `json:"uptime_ratio"`
	AvgLatencyMS     float64    `json:"avg_latency_ms"`
	P95LatencyMS     int        `json:"p95_latency_ms"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	Buckets          []Bucket   `json:"buckets"`
}

// Incident — инцидент в API.
type Incident struct {
	ID              string     `json:"id"`
	MonitorID       string     `json:"monitor_id"`
	StartedAt       time.Time  `json:"started_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	Cause           string     `json:"cause"`
	IsOpen          bool       `json:"is_open"`
	DurationSeconds int64      `json:"duration_seconds"`
}

// IncidentList — страница инцидентов.
type IncidentList struct {
	Items  []Incident `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

// StatusPageService — монитор на публичной статус-странице.
//
// Здесь намеренно нет поля target: статус-страница публична, а URL цели
// может раскрывать внутренние адреса и служебные эндпоинты.
type StatusPageService struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	UptimeRatio float64    `json:"uptime_ratio"`
	Buckets     []Bucket   `json:"buckets"`
	LastCheckAt *time.Time `json:"last_check_at"`
}

// StatusPageIncident — инцидент в публичной ленте.
type StatusPageIncident struct {
	Service         string     `json:"service"`
	StartedAt       time.Time  `json:"started_at"`
	ResolvedAt      *time.Time `json:"resolved_at"`
	Cause           string     `json:"cause"`
	DurationSeconds int64      `json:"duration_seconds"`
}

// StatusPage — публичная статус-страница.
type StatusPage struct {
	From      time.Time            `json:"from"`
	To        time.Time            `json:"to"`
	Overall   string               `json:"overall"`
	Services  []StatusPageService  `json:"services"`
	Incidents []StatusPageIncident `json:"incidents"`
}

// Health — ответ технических эндпоинтов.
type Health struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Env     string `json:"env"`
}
