package app

import (
	"log/slog"
	"os"
	"strings"

	"github.com/St1lon/sentinel/internal/config"
)

func NewLogger(logCfg *config.LogConfig, appCfg *config.AppConfig, service string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(logCfg.Level),
		ReplaceAttr: renameStandardKeys,
	}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if logCfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("service.name", service),
		slog.String("version", appCfg.Version),
		slog.String("env", appCfg.Env),
	)
}


func renameStandardKeys(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return attr
	}

	switch attr.Key {
	case slog.TimeKey:
		attr.Key = "timestamp"
	case slog.MessageKey:
		attr.Key = "message"
	case slog.LevelKey:
		attr.Value = slog.StringValue(strings.ToLower(attr.Value.String()))
	}

	return attr
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
