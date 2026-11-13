package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
)

const (
	minJWTSecretLength = 32
)

type AppConfig struct {
	Env     string `env:"APP_ENV"     envDefault:"development" validate:"oneof=development staging production"`
	Version string `env:"APP_VERSION" envDefault:"dev"         validate:"required"`
}

type LogConfig struct {
	Level  string `env:"LOG_LEVEL"  envDefault:"info" validate:"oneof=debug info warn error"`
	Format string `env:"LOG_FORMAT" envDefault:"json" validate:"oneof=json text"`
}

type PostgresConfig struct {
	DSN             string        `env:"DATABASE_URL,required"       validate:"required,startswith=postgres"`
	MaxConns        int32         `env:"POSTGRES_MAX_CONNS"          envDefault:"10" validate:"min=1,max=1000"`
	MinConns        int32         `env:"POSTGRES_MIN_CONNS"          envDefault:"2"  validate:"min=0,max=1000"`
	MaxConnLifetime time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME"  envDefault:"30m" validate:"min=1s"`
	MaxConnIdleTime time.Duration `env:"POSTGRES_MAX_CONN_IDLE_TIME" envDefault:"5m"  validate:"min=1s"`
	ConnectTimeout  time.Duration `env:"POSTGRES_CONNECT_TIMEOUT"    envDefault:"5s"  validate:"min=100ms"`
}

type HTTPConfig struct {
	Port               int           `env:"HTTP_PORT"                  envDefault:"8080" validate:"min=1,max=65535"`
	ReadHeaderTimeout  time.Duration `env:"HTTP_READ_HEADER_TIMEOUT"   envDefault:"5s"   validate:"min=100ms"`
	ReadTimeout        time.Duration `env:"HTTP_READ_TIMEOUT"          envDefault:"15s"  validate:"min=100ms"`
	WriteTimeout       time.Duration `env:"HTTP_WRITE_TIMEOUT"         envDefault:"15s"  validate:"min=100ms"`
	IdleTimeout        time.Duration `env:"HTTP_IDLE_TIMEOUT"          envDefault:"60s"  validate:"min=1s"`
	ShutdownTimeout    time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT"      envDefault:"15s"  validate:"min=1s"`
	CORSAllowedOrigins []string      `env:"HTTP_CORS_ALLOWED_ORIGINS"  envDefault:"*"    envSeparator:"," validate:"min=1"`
	RequestBodyLimit   int64         `env:"HTTP_REQUEST_BODY_LIMIT"    envDefault:"65536" validate:"min=1024"`
}

// MetricsConfig — выделенный порт сбора метрик: эндпоинт без авторизации
// не должен попадать в Service приложения.
type MetricsConfig struct {
	Port              int           `env:"METRICS_PORT"                envDefault:"9090"     validate:"min=1,max=65535"`
	Path              string        `env:"METRICS_PATH"                envDefault:"/metrics" validate:"required,startswith=/"`
	ReadHeaderTimeout time.Duration `env:"METRICS_READ_HEADER_TIMEOUT" envDefault:"5s"       validate:"min=100ms"`
	ShutdownTimeout   time.Duration `env:"METRICS_SHUTDOWN_TIMEOUT"    envDefault:"5s"       validate:"min=1s"`
}

// MetricsPushConfig — куда разовая задача отдаёт свои метрики.
// Пустой URL отключает отправку.
type MetricsPushConfig struct {
	URL     string        `env:"METRICS_PUSH_URL"     validate:"omitempty,url"`
	Timeout time.Duration `env:"METRICS_PUSH_TIMEOUT" envDefault:"5s" validate:"min=100ms"`
}

type AuthConfig struct {
	JWTSecret  string        `env:"AUTH_JWT_SECRET,required" validate:"required,min=32"`
	TokenTTL   time.Duration `env:"AUTH_TOKEN_TTL"  envDefault:"24h" validate:"min=1m"`
	BcryptCost int           `env:"AUTH_BCRYPT_COST" envDefault:"10" validate:"min=4,max=31"`
	Issuer     string        `env:"AUTH_JWT_ISSUER" envDefault:"sentinel" validate:"required"`
}

type WorkerConfig struct {
	Concurrency      int           `env:"WORKER_CONCURRENCY"           envDefault:"8"  validate:"min=1,max=1024"`
	BatchSize        int           `env:"WORKER_BATCH_SIZE"            envDefault:"50"  validate:"min=1,max=1000"`
	PollInterval     time.Duration `env:"WORKER_POLL_INTERVAL"         envDefault:"1s"  validate:"min=100ms"`
	ShutdownTimeout  time.Duration `env:"WORKER_SHUTDOWN_TIMEOUT"      envDefault:"30s" validate:"min=1s"`
	UserAgent        string        `env:"WORKER_USER_AGENT"            envDefault:"SentinelUptimeBot/1.0" validate:"required"`
	MaxResponseBytes int64         `env:"WORKER_MAX_RESPONSE_BYTES"    envDefault:"65536" validate:"min=0"`
	MaxRedirects     int           `env:"WORKER_MAX_REDIRECTS"         envDefault:"3"   validate:"min=0,max=10"`
	StatsInterval    time.Duration `env:"WORKER_STATS_INTERVAL"        envDefault:"15s" validate:"min=1s"`
}

type TargetsConfig struct {
	AllowPrivate bool `env:"ALLOW_PRIVATE_TARGETS" envDefault:"false"`
}

// ProberConfig — blackbox-пробер: ходит в наше же API снаружи по сценарию
// критического пользовательского пути и отдаёт результат метриками SLI.
// Пустой PROBER_STATUS_SLUG отключает шаг публичной страницы: slug выдаётся
// при регистрации и заранее не известен.
type ProberConfig struct {
	TargetURL     string        `env:"PROBER_TARGET_URL,required"     validate:"required,url"`
	Interval      time.Duration `env:"PROBER_INTERVAL"                envDefault:"10s" validate:"min=1s"`
	Timeout       time.Duration `env:"PROBER_TIMEOUT"                 envDefault:"3s"  validate:"min=100ms"`
	LoginEmail    string        `env:"PROBER_LOGIN_EMAIL,required"    validate:"required,email"`
	LoginPassword string        `env:"PROBER_LOGIN_PASSWORD,required" validate:"required"`
	StatusSlug    string        `env:"PROBER_STATUS_SLUG"`
	UserAgent     string        `env:"PROBER_USER_AGENT"              envDefault:"SentinelProber/1.0" validate:"required"`
}

type RetentionConfig struct {
	CheckRetentionDays int `env:"RETENTION_CHECK_DAYS" envDefault:"30" validate:"min=1,max=3650"`
	BatchSize          int `env:"RETENTION_BATCH_SIZE" envDefault:"10000" validate:"min=100,max=1000000"`
}

// ChaosConfig — управляемая деградация API для демонстрации SLI, SLO и
// алертов. По умолчанию всё выключено, в production включение запрещено.
// Сбои вносятся только в /api/, проверки живости не трогаются: иначе
// kubelet начнёт перезапускать поды, и вместо деградации выйдет падение.
//
// GoroutineLeak и MemHogMB — режимы для алертов на метрики рантайма:
// утечку горутин и рост памяти нельзя вызвать полезной нагрузкой, они
// имитируются напрямую (спавн горутин и удержание аллокаций на запрос).
type ChaosConfig struct {
	ErrorRate     float64       `env:"CHAOS_ERROR_RATE"      envDefault:"0"   validate:"min=0,max=1"`
	Latency       time.Duration `env:"CHAOS_LATENCY"         envDefault:"0s"  validate:"min=0"`
	LatencyRate   float64       `env:"CHAOS_LATENCY_RATE"    envDefault:"0"   validate:"min=0,max=1"`
	ReadyzFail    bool          `env:"CHAOS_READYZ_FAIL"     envDefault:"false"`
	GoroutineLeak int           `env:"CHAOS_GOROUTINE_LEAK"  envDefault:"0"   validate:"min=0,max=1000"`
	MemHogMB      int           `env:"CHAOS_MEM_HOG_MB"      envDefault:"0"   validate:"min=0,max=64"`
}

// Enabled сообщает, нужно ли вообще включать middleware в цепочку.
func (c *ChaosConfig) Enabled() bool {
	return c.ErrorRate > 0 || (c.Latency > 0 && c.LatencyRate > 0) || c.ReadyzFail ||
		c.GoroutineLeak > 0 || c.MemHogMB > 0
}

type APIConfig struct {
	App      AppConfig
	Log      LogConfig
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	Targets  TargetsConfig
	Metrics  MetricsConfig
	Chaos    ChaosConfig
}

type WorkerProcessConfig struct {
	App      AppConfig
	Log      LogConfig
	Postgres PostgresConfig
	Worker   WorkerConfig
	Targets  TargetsConfig
	Metrics  MetricsConfig
}

type ProberProcessConfig struct {
	App     AppConfig
	Log     LogConfig
	Metrics MetricsConfig
	Prober  ProberConfig
}

type CleanupConfig struct {
	App       AppConfig
	Log       LogConfig
	Postgres  PostgresConfig
	Retention RetentionConfig
	Push      MetricsPushConfig
}

func NewValidator() *validator.Validate {
	return validator.New(validator.WithRequiredStructEnabled())
}

func LoadAPIConfig(validate *validator.Validate) (*APIConfig, error) {
	cfg, err := parse[APIConfig](validate)
	if err != nil {
		return nil, err
	}

	if err := checkPostgres(&cfg.Postgres); err != nil {
		return nil, err
	}

	if err := checkHTTP(&cfg.HTTP); err != nil {
		return nil, err
	}

	if err := checkMetrics(cfg.HTTP.Port, &cfg.Metrics); err != nil {
		return nil, err
	}

	if err := checkAuth(&cfg.App, &cfg.Auth); err != nil {
		return nil, err
	}

	if err := checkTargets(&cfg.App, &cfg.Targets); err != nil {
		return nil, err
	}

	if err := checkChaos(&cfg.App, &cfg.Chaos); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadWorkerConfig(validate *validator.Validate) (*WorkerProcessConfig, error) {
	cfg, err := parse[WorkerProcessConfig](validate)
	if err != nil {
		return nil, err
	}

	if err := checkPostgres(&cfg.Postgres); err != nil {
		return nil, err
	}

	if int(cfg.Postgres.MaxConns) < cfg.Worker.Concurrency {
		return nil, fmt.Errorf(
			"%w: POSTGRES_MAX_CONNS (%d) must be >= WORKER_CONCURRENCY (%d)",
			errInvalidConfig, cfg.Postgres.MaxConns, cfg.Worker.Concurrency,
		)
	}

	if err := checkTargets(&cfg.App, &cfg.Targets); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadProberConfig(validate *validator.Validate) (*ProberProcessConfig, error) {
	cfg, err := parse[ProberProcessConfig](validate)
	if err != nil {
		return nil, err
	}

	if err := checkProber(&cfg.Prober); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadCleanupConfig(validate *validator.Validate) (*CleanupConfig, error) {
	cfg, err := parse[CleanupConfig](validate)
	if err != nil {
		return nil, err
	}

	if err := checkPostgres(&cfg.Postgres); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parse[T any](validate *validator.Validate) (*T, error) {
	var cfg T

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}
