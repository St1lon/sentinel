// Package handlers содержит HTTP-хендлеры. Хендлер разбирает запрос,
// вызывает ровно один usecase и переводит результат в DTO через mapper.
// Обращаться к репозиториям или внешним клиентам напрямую он не может.
package handlers

import (
	"context"
	"log/slog"

	registeruser "github.com/St1lon/sentinel/internal/usecase/auth/register"
	listchecks "github.com/St1lon/sentinel/internal/usecase/check/list"
	listincidents "github.com/St1lon/sentinel/internal/usecase/incident/list"
	createmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/create"
	deletemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/delete"
	getmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/get"
	listmonitors "github.com/St1lon/sentinel/internal/usecase/monitor/list"
	monitorstats "github.com/St1lon/sentinel/internal/usecase/monitor/stats"
	getstatuspage "github.com/St1lon/sentinel/internal/usecase/statuspage/get"

	loginuser "github.com/St1lon/sentinel/internal/usecase/auth/login"
	updatemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/update"
	getuser "github.com/St1lon/sentinel/internal/usecase/user/get"
)

// Pinger — порт проверки готовности присоединённого ресурса для /readyz.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps — зависимости хендлеров. Структура вместо длинного списка аргументов:
// состав зависимостей меняется чаще, чем их смысл.
type Deps struct {
	Logger    *slog.Logger
	Version   string
	Env       string
	BodyLimit int64

	Register    *registeruser.Usecase
	Login       *loginuser.Usecase
	CurrentUser *getuser.Usecase

	CreateMonitor *createmonitor.Usecase
	ListMonitors  *listmonitors.Usecase
	GetMonitor    *getmonitor.Usecase
	UpdateMonitor *updatemonitor.Usecase
	DeleteMonitor *deletemonitor.Usecase
	MonitorStats  *monitorstats.Usecase
	ListChecks    *listchecks.Usecase
	ListIncidents *listincidents.Usecase
	StatusPage    *getstatuspage.Usecase

	DB Pinger
}

// Handlers — набор HTTP-хендлеров приложения.
type Handlers struct {
	deps Deps
}

// New создаёт хендлеры.
func New(deps Deps) *Handlers {
	return &Handlers{deps: deps}
}
