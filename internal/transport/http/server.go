package http

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

// Server — обёртка над http.Server с корректным запуском и остановкой.
// Процесс сам слушает порт из конфигурации и не зависит от внешнего
// контейнера приложений (фактор VII «Port binding»).
type Server struct {
	server *http.Server
	logger *slog.Logger
	cfg    *config.HTTPConfig
}

// NewServer создаёт HTTP-сервер с таймаутами из конфигурации.
func NewServer(cfg *config.HTTPConfig, handler http.Handler, logger *slog.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:              net.JoinHostPort("", strconv.Itoa(cfg.Port)),
			Handler:           handler,
			ReadHeaderTimeout: cfg.ReadHeaderTimeout,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
			IdleTimeout:       cfg.IdleTimeout,
		},
		logger: logger,
		cfg:    cfg,
	}
}

// Run слушает порт до ошибки или остановки сервера.
func (s *Server) Run() error {
	s.logger.Info("http server started", slog.Int("port", s.cfg.Port))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

// Shutdown завершает сервер, давая активным запросам доиграть в пределах
// ShutdownTimeout (фактор IX «Disposability»).
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	s.logger.Info("http server stopped")

	return nil
}
