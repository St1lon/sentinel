package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/infra/auth"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
	"github.com/St1lon/sentinel/internal/infra/db/postgres/repository"
	"github.com/St1lon/sentinel/internal/infra/metrics"
	"github.com/St1lon/sentinel/internal/infra/prober"
	transporthttp "github.com/St1lon/sentinel/internal/transport/http"
	"github.com/St1lon/sentinel/internal/transport/http/handlers"
	transportmetrics "github.com/St1lon/sentinel/internal/transport/metrics"
	loginuser "github.com/St1lon/sentinel/internal/usecase/auth/login"
	registeruser "github.com/St1lon/sentinel/internal/usecase/auth/register"
	listchecks "github.com/St1lon/sentinel/internal/usecase/check/list"
	listincidents "github.com/St1lon/sentinel/internal/usecase/incident/list"
	createmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/create"
	deletemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/delete"
	getmonitor "github.com/St1lon/sentinel/internal/usecase/monitor/get"
	listmonitors "github.com/St1lon/sentinel/internal/usecase/monitor/list"
	monitorstats "github.com/St1lon/sentinel/internal/usecase/monitor/stats"
	updatemonitor "github.com/St1lon/sentinel/internal/usecase/monitor/update"
	getstatuspage "github.com/St1lon/sentinel/internal/usecase/statuspage/get"
	getuser "github.com/St1lon/sentinel/internal/usecase/user/get"
)

type API struct {
	server        *transporthttp.Server
	metricsServer *transportmetrics.Server
	pool          *pgxpool.Pool
	logger        *slog.Logger
}

func BuildAPI(ctx context.Context, cfg *config.APIConfig, logger *slog.Logger) (*API, error) {
	pool, err := postgres.NewPool(ctx, &cfg.Postgres)
	if err != nil {
		return nil, err
	}

	registry := metrics.NewRegistry(ServiceAPI)
	txManager := postgres.NewTxManager(pool)

	users := repository.NewUserRepo(pool)
	monitors := repository.NewMonitorRepo(pool)
	checks := repository.NewCheckRepo(pool)
	incidents := repository.NewIncidentRepo(pool)

	hasher := auth.NewBcryptHasher(cfg.Auth.BcryptCost)
	tokens := auth.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL, cfg.Auth.Issuer)
	targetGuard := prober.NewGuard(cfg.Targets.AllowPrivate)

	httpHandlers := handlers.New(handlers.Deps{
		Logger:    logger,
		Version:   cfg.App.Version,
		Env:       cfg.App.Env,
		BodyLimit: cfg.HTTP.RequestBodyLimit,
		Metrics:   metrics.NewBusiness(registry.Registerer()),

		Register:    registeruser.NewUsecase(users, hasher, tokens),
		Login:       loginuser.NewUsecase(users, hasher, tokens),
		CurrentUser: getuser.NewUsecase(users),

		CreateMonitor: createmonitor.NewUsecase(monitors, targetGuard),
		ListMonitors:  listmonitors.NewUsecase(monitors),
		GetMonitor:    getmonitor.NewUsecase(monitors),
		UpdateMonitor: updatemonitor.NewUsecase(monitors, incidents, targetGuard, txManager),
		DeleteMonitor: deletemonitor.NewUsecase(monitors),
		MonitorStats:  monitorstats.NewUsecase(monitors, checks),
		ListChecks:    listchecks.NewUsecase(monitors, checks),
		ListIncidents: listincidents.NewUsecase(monitors, incidents),
		StatusPage:    getstatuspage.NewUsecase(users, monitors, checks, incidents),

		DB: pool,
	})

	router := transporthttp.NewRouter(transporthttp.RouterDeps{
		Handlers:       httpHandlers,
		TokenParser:    tokens,
		Logger:         logger,
		Requests:       metrics.NewHTTP(registry.Registerer()),
		AllowedOrigins: cfg.HTTP.CORSAllowedOrigins,
	})

	return &API{
		server:        transporthttp.NewServer(&cfg.HTTP, router, logger),
		metricsServer: transportmetrics.NewServer(&cfg.Metrics, registry.Handler(), logger),
		pool:          pool,
		logger:        logger,
	}, nil
}

// Run поднимает два слушателя: публичный API и порт сбора метрик.
func (a *API) Run() error {
	var group errgroup.Group

	group.Go(a.server.Run)
	group.Go(a.metricsServer.Run)

	if err := group.Wait(); err != nil {
		return fmt.Errorf("run api: %w", err)
	}

	return nil
}

func (a *API) Shutdown(ctx context.Context) error {
	err := errors.Join(
		a.server.Shutdown(ctx),
		a.metricsServer.Shutdown(ctx),
	)

	a.pool.Close()
	a.logger.Info("database pool closed")

	return err
}
