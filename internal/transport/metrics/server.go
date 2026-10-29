// Package metrics — отдельный HTTP-сервер для сбора метрик. Порт выделенный:
// /metrics отдаётся без авторизации и не должен попадать в Service приложения.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"

	"github.com/St1lon/sentinel/internal/config"
)

type Server struct {
	server *http.Server
	logger *slog.Logger
	cfg    *config.MetricsConfig
}

func NewServer(cfg *config.MetricsConfig, handler http.Handler, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	mux.Handle(cfg.Path, handler)

	return &Server{
		server: &http.Server{
			Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
			Handler:           mux,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		},
		logger: logger,
		cfg:    cfg,
	}
}

func (s *Server) Run() error {
	s.logger.Info("metrics server started",
		slog.Int("port", s.cfg.Port),
		slog.String("path", s.cfg.Path),
	)

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve metrics: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown metrics server: %w", err)
	}

	s.logger.Info("metrics server stopped")

	return nil
}
