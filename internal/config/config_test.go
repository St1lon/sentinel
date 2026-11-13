package config_test

import (
	"testing"
	"time"

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

func TestLoadAPIConfig_ChaosDisabledByDefault(t *testing.T) {
	setBaseEnv(t)

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)
	require.False(t, cfg.Chaos.Enabled(), "деградация включается только вручную")
}

func TestLoadAPIConfig_ChaosForbiddenInProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_JWT_SECRET", "jK8vQ2mZ4pR7tY1wB6nF3hL9sD5gX0cV8aU2eI4oP6M=")
	t.Setenv("CHAOS_ERROR_RATE", "0.3")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "CHAOS_")
}

func TestLoadAPIConfig_ChaosRateAboveOneRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("CHAOS_ERROR_RATE", "1.5")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err)
}

// Задержка без доли запросов никого не задержала бы — это молчаливая
// опечатка в демонстрации, поэтому отвергается на старте.
func TestLoadAPIConfig_ChaosLatencyWithoutRateRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("CHAOS_LATENCY", "800ms")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "CHAOS_LATENCY_RATE")
}

// Ресурсные режимы включают деградацию так же, как остальные CHAOS_*:
// они имитируют утечки для алертов на метрики рантайма.
func TestLoadAPIConfig_ChaosResourceModesEnableDegradation(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("CHAOS_GOROUTINE_LEAK", "10")
	t.Setenv("CHAOS_MEM_HOG_MB", "8")

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)
	require.True(t, cfg.Chaos.Enabled())
	require.Equal(t, 10, cfg.Chaos.GoroutineLeak)
	require.Equal(t, 8, cfg.Chaos.MemHogMB)
}

func TestLoadAPIConfig_ChaosResourceModesForbiddenInProduction(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_JWT_SECRET", "jK8vQ2mZ4pR7tY1wB6nF3hL9sD5gX0cV8aU2eI4oP6M=")
	t.Setenv("CHAOS_GOROUTINE_LEAK", "10")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.ErrorContains(t, err, "CHAOS_")
}

func setProberEnv(t *testing.T) {
	t.Helper()

	t.Setenv("PROBER_TARGET_URL", "http://api.sentinel.svc.cluster.local:8080")
	t.Setenv("PROBER_LOGIN_EMAIL", "demo@sentinel.local")
	t.Setenv("PROBER_LOGIN_PASSWORD", "demo-password-123")
}

func TestLoadProberConfig_Defaults(t *testing.T) {
	setProberEnv(t)

	cfg, err := config.LoadProberConfig(config.NewValidator())
	require.NoError(t, err)

	require.Equal(t, 10*time.Second, cfg.Prober.Interval)
	require.Equal(t, 3*time.Second, cfg.Prober.Timeout)
	require.Equal(t, "SentinelProber/1.0", cfg.Prober.UserAgent)
	require.Equal(t, 9090, cfg.Metrics.Port)
	require.Empty(t, cfg.Prober.StatusSlug, "без slug шаг публичной страницы пропускается")
}

func TestLoadProberConfig_RequiredTargetMissing(t *testing.T) {
	t.Setenv("PROBER_LOGIN_EMAIL", "demo@sentinel.local")
	t.Setenv("PROBER_LOGIN_PASSWORD", "demo-password-123")

	_, err := config.LoadProberConfig(config.NewValidator())
	require.Error(t, err, "без адреса API проберу некуда ходить")
}

func TestLoadProberConfig_TrailingSlashTrimmed(t *testing.T) {
	setProberEnv(t)
	t.Setenv("PROBER_TARGET_URL", "http://localhost:8081/")

	cfg, err := config.LoadProberConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8081", cfg.Prober.TargetURL,
		"иначе шаги сценария дадут двойной слеш в пути")
}

func TestLoadProberConfig_NonHTTPTargetRejected(t *testing.T) {
	setProberEnv(t)
	t.Setenv("PROBER_TARGET_URL", "api.sentinel.svc:8080")

	_, err := config.LoadProberConfig(config.NewValidator())
	require.Error(t, err)
}

func TestLoadProberConfig_TimeoutAboveIntervalRejected(t *testing.T) {
	setProberEnv(t)
	t.Setenv("PROBER_INTERVAL", "2s")
	t.Setenv("PROBER_TIMEOUT", "5s")

	_, err := config.LoadProberConfig(config.NewValidator())
	require.ErrorContains(t, err, "PROBER_TIMEOUT")
}

func TestLoadCleanupConfig_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadCleanupConfig(config.NewValidator())
	require.NoError(t, err)
	require.Equal(t, 30, cfg.Retention.CheckRetentionDays)
}

func TestLoadAPIConfig_MetricsDefaults(t *testing.T) {
	setBaseEnv(t)

	cfg, err := config.LoadAPIConfig(config.NewValidator())
	require.NoError(t, err)

	require.Equal(t, 9090, cfg.Metrics.Port)
	require.Equal(t, "/metrics", cfg.Metrics.Path)
}

func TestLoadAPIConfig_MetricsPortEqualToHTTPPortRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("METRICS_PORT", "8080")

	_, err := config.LoadAPIConfig(config.NewValidator())
	require.Error(t, err, "метрики слушают отдельный порт, иначе они попадут в публичный Service")
}

func TestLoadCleanupConfig_PushDisabledByDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)

	cfg, err := config.LoadCleanupConfig(config.NewValidator())
	require.NoError(t, err)
	require.Empty(t, cfg.Push.URL, "без адреса отправки задача просто не шлёт метрики")
}

func TestLoadCleanupConfig_PushURLValidated(t *testing.T) {
	t.Setenv("DATABASE_URL", validDSN)
	t.Setenv("METRICS_PUSH_URL", "not-a-url")

	_, err := config.LoadCleanupConfig(config.NewValidator())
	require.Error(t, err)
}
