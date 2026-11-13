// notifier-sink — приёмник уведомлений Alertmanager для стенда.
//
// Alertmanager сам ничего не показывает: он рассылает уведомления в
// настроенные receivers (email, Slack, Telegram, webhook). Тип webhook —
// универсальный: POST с JSON-телом уведомления на любой URL. Внешних каналов
// у стенда нет, поэтому уведомления принимает этот крошечный сервис и
// печатает тело в stdout: лог пода и есть «скриншот уведомления» для отчёта.
// Ровно так в реальной жизни отлаживают webhook-интеграции.
//
// Процесс сознательно не входит в слой internal/app: у него нет ни бизнеса,
// ни конфига, ни метрик — это отладочный утилитарный сервис, а не часть
// приложения.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultAddr     = ":8080"
	maxBodyBytes    = 1 << 20 // 1 MiB: уведомление Alertmanager на сотни алертов и то меньше
	shutdownTimeout = 5 * time.Second
)

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	addr := os.Getenv("SINK_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	mux := http.NewServeMux()

	// Проверка живости для Deployment. Отдельно от /alert: kubelet должен
	// опрашивать эндпоинт, на который никто не пишет.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Эндпоинт, на который Alertmanager шлёт уведомления (webhook_configs.url).
	// Тело печатается с отступами: лог читается глазами, а не парсером.
	mux.HandleFunc("POST /alert", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			logger.Warn("unreadable notification", slog.String("error", err.Error()))
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		logger.Info("notification received", slog.String("payload", pretty(body)))

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Graceful shutdown: завершаем начатые записи в лог, а не рвём их.
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("shutdown", slog.String("error", err.Error()))
		}
	}()

	logger.Info("notifier sink started", slog.String("addr", addr))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}

// pretty переформатирует JSON с отступами. Если тело не JSON — вернём как есть:
// sink отладочный, ломаться на произвольном вводе он не должен.
func pretty(body []byte) string {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return string(body)
	}

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return string(body)
	}

	return string(out)
}
