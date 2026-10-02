// Package app — composition root: ручная сборка зависимостей и управление
// жизненным циклом процессов. Выше этого пакета только cmd/*/main.go.
package app

import (
	"log/slog"
	"os"

	"github.com/St1lon/sentinel/internal/config"
)

// NewLogger собирает структурированный логгер, пишущий в stdout.
//
// Приложение не управляет местом хранения логов: это поток событий, который
// перенаправляет среда исполнения (фактор XI «Logs»). Поэтому ни файлов,
// ни ротации, ни отправки логов во внешние системы здесь нет.
func NewLogger(logCfg *config.LogConfig, appCfg *config.AppConfig) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(logCfg.Level)}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if logCfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("version", appCfg.Version),
		slog.String("env", appCfg.Env),
	)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
