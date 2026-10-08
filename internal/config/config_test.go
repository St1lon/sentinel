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

func TestLoadAPIConfig_ShortSecretRejectedInProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "at least 44 characters in production")
}

func TestLoadAPIConfig_PlaceholderSecretsRejectedInProduction(t *testing.T) {
	// Все значения достаточной длины — отвергаются именно как заглушки.
	placeholders := []string{
		"change-me-in-production-at-least-32-chars-here",
		"k8s-local-secret-at-least-32-characters-long",
		"example-secret-value-for-the-application-dev",
		"insecure-placeholder-secret-value-for-tests!!",
		"TODO-replace-this-secret-before-going-to-prod",
	}

	for _, secret := range placeholders {
		t.Run(secret, func(t *testing.T) {
			setBaseEnv(t)
			t.Setenv("APP_ENV", "production")
			t.Setenv("AUTH_JWT_SECRET", secret)

			_, err := config.LoadAPIConfig(config.NewValidator())
			require.ErrorContains(t, err, "looks like a placeholder")
		})
	}
}

func TestLoadAPIConfig_GeneratedSecretAcceptedInProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	// Результат `openssl rand -base64 32`.
	t.Setenv("AUTH_JWT_SECRET", "jK8vQ2mZ4pR7tY1wB6nF3hL9sD5gX0cV8aU2eI4oP6M=")

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, "production", cfg.App.Env)
}

func TestLoadAPIConfig_PlaceholderAllowedOutsideProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_JWT_SECRET", "k8s-local-secret-at-least-32-characters")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err, "в локальном окружении заглушка допустима")
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
	t.Setenv("ALLOW_PRIVATE_TARGETS", "true")

	_, err := config.LoadWorkerConfig(config.NewValidator())
	require.ErrorContains(t, err, "ALLOW_PRIVATE_TARGETS")
}

func TestLoadWorkerConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadWorkerConfig(config.NewValidator())
	require.NoError(t, err)

	require.Equal(t, 8, cfg.Worker.Concurrency)
	require.Equal(t, 50, cfg.Worker.BatchSize)
	require.False(t, cfg.Targets.AllowPrivate)
}

func TestLoadCleanupConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadCleanupConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, 30, cfg.Retention.CheckRetentionDays)
}
