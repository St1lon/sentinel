package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	envProduction = "production"

	exampleJWTSecret = "change-me-in-production-at-least-32-chars"
)

var errInvalidConfig = errors.New("invalid config")

func checkPostgres(cfg *PostgresConfig) error {
	if cfg.MinConns > cfg.MaxConns {
		return fmt.Errorf(
			"%w: POSTGRES_MIN_CONNS (%d) must be <= POSTGRES_MAX_CONNS (%d)",
			errInvalidConfig, cfg.MinConns, cfg.MaxConns,
		)
	}

	if cfg.MaxConnIdleTime > cfg.MaxConnLifetime {
		return fmt.Errorf(
			"%w: POSTGRES_MAX_CONN_IDLE_TIME must be <= POSTGRES_MAX_CONN_LIFETIME",
			errInvalidConfig,
		)
	}

	return nil
}

func checkHTTP(cfg *HTTPConfig) error {
	if cfg.ReadHeaderTimeout > cfg.ReadTimeout {
		return fmt.Errorf(
			"%w: HTTP_READ_HEADER_TIMEOUT must be <= HTTP_READ_TIMEOUT",
			errInvalidConfig,
		)
	}

	for _, origin := range cfg.CORSAllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			return fmt.Errorf("%w: HTTP_CORS_ALLOWED_ORIGINS contains an empty origin", errInvalidConfig)
		}
	}

	return nil
}

func checkAuth(app *AppConfig, cfg *AuthConfig) error {
	if len(cfg.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("%w: AUTH_JWT_SECRET must be at least %d characters",
			errInvalidConfig, minJWTSecretLength)
	}

	if app.Env == envProduction && cfg.JWTSecret == exampleJWTSecret {
		return fmt.Errorf("%w: AUTH_JWT_SECRET must not be the example value in production", errInvalidConfig)
	}

	return nil
}
