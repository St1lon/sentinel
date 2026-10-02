package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/config"
)

const (
	validDSN    = "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"
	validSecret = "0123456789abcdef0123456789abcdef"
)

func setBaseEnv(t *testing.T) {
	t.Helper()

	t.Setenv("DATABASE_URL", validDSN)
	t.Setenv("AUTH_JWT_SECRET", validSecret)
}

func TestLoadAPIConfig_DefaultsApplied(t *testing.T) {
	setBaseEnv(t)

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)

	require.Equal(t, "development", cfg.App.Env)
	require.Equal(t, 8080, cfg.HTTP.Port)
	require.Equal(t, "json", cfg.Log.Format)
	require.Equal(t, int32(10), cfg.Postgres.MaxConns)
	require.Equal(t, validDSN, cfg.Postgres.DSN)
}

func TestLoadAPIConfig_RequiredDSNMissing(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", validSecret)

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err, "DATABASE_URL обязателен и не имеет дефолта")
}

func TestLoadAPIConfig_RequiredSecretMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err, "AUTH_JWT_SECRET обязателен и не имеет дефолта")
}

func TestLoadAPIConfig_ShortSecretRejected(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)
	t.Setenv("AUTH_JWT_SECRET", "too-short")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err)
}

func TestLoadAPIConfig_ExampleSecretRejectedInProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_JWT_SECRET", "change-me-in-production-at-least-32-chars")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "must not be the example value")
}

func TestLoadAPIConfig_UnknownEnvRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "staging-2")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err)
}

func TestLoadAPIConfig_MinConnsAboveMaxRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("POSTGRES_MIN_CONNS", "20")
	t.Setenv("POSTGRES_MAX_CONNS", "10")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "POSTGRES_MIN_CONNS")
}

func TestLoadAPIConfig_CORSListParsed(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("HTTP_CORS_ALLOWED_ORIGINS", "https://a.example,https://b.example")

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.HTTP.CORSAllowedOrigins)
}

func TestLoadWorkerConfig_PoolSmallerThanConcurrencyRejected(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)
	t.Setenv("WORKER_CONCURRENCY", "32")
	t.Setenv("POSTGRES_MAX_CONNS", "10")

	_, err := config.LoadWorkerConfig(config.NewValidator())
	require.ErrorContains(t, err, "WORKER_CONCURRENCY")
}

func TestLoadWorkerConfig_PrivateTargetsForbiddenInProduction(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)
	t.Setenv("APP_ENV", "production")
	t.Setenv("WORKER_ALLOW_PRIVATE_TARGETS", "true")

	_, err := config.LoadWorkerConfig(config.NewValidator())
	require.ErrorContains(t, err, "WORKER_ALLOW_PRIVATE_TARGETS")
}

func TestLoadWorkerConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadWorkerConfig(config.NewValidator())
	require.NoError(t, err)

	require.Equal(t, 8, cfg.Worker.Concurrency)
	require.Equal(t, 50, cfg.Worker.BatchSize)
	require.False(t, cfg.Worker.AllowPrivateTargets)
}

func TestLoadCleanupConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadCleanupConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, 30, cfg.Retention.CheckRetentionDays)
}
