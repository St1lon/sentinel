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

type AuthConfig struct {
	JWTSecret  string        `env:"AUTH_JWT_SECRET,required" validate:"required,min=32"`
	TokenTTL   time.Duration `env:"AUTH_TOKEN_TTL"  envDefault:"24h" validate:"min=1m"`
	BcryptCost int           `env:"AUTH_BCRYPT_COST" envDefault:"10" validate:"min=4,max=31"`
	Issuer     string        `env:"AUTH_JWT_ISSUER" envDefault:"sentinel" validate:"required"`
}

type WorkerConfig struct {
	Concurrency         int           `env:"WORKER_CONCURRENCY"           envDefault:"8"  validate:"min=1,max=1024"`
	BatchSize           int           `env:"WORKER_BATCH_SIZE"            envDefault:"50"  validate:"min=1,max=1000"`
	PollInterval        time.Duration `env:"WORKER_POLL_INTERVAL"         envDefault:"1s"  validate:"min=100ms"`
	ShutdownTimeout     time.Duration `env:"WORKER_SHUTDOWN_TIMEOUT"      envDefault:"30s" validate:"min=1s"`
	UserAgent           string        `env:"WORKER_USER_AGENT"            envDefault:"SentinelUptimeBot/1.0" validate:"required"`
	MaxResponseBytes    int64         `env:"WORKER_MAX_RESPONSE_BYTES"    envDefault:"65536" validate:"min=0"`
	MaxRedirects        int           `env:"WORKER_MAX_REDIRECTS"         envDefault:"3"   validate:"min=0,max=10"`
	AllowPrivateTargets bool          `env:"WORKER_ALLOW_PRIVATE_TARGETS" envDefault:"false"`
}

type RetentionConfig struct {
	CheckRetentionDays int `env:"RETENTION_CHECK_DAYS" envDefault:"30" validate:"min=1,max=3650"`
	BatchSize          int `env:"RETENTION_BATCH_SIZE" envDefault:"10000" validate:"min=100,max=1000000"`
}

type APIConfig struct {
	App      AppConfig
	Log      LogConfig
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Auth     AuthConfig
}

type WorkerProcessConfig struct {
	App      AppConfig
	Log      LogConfig
	Postgres PostgresConfig
	Worker   WorkerConfig
}

type CleanupConfig struct {
	App       AppConfig
	Log       LogConfig
	Postgres  PostgresConfig
	Retention RetentionConfig
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

	if err := checkAuth(&cfg.App, &cfg.Auth); err != nil {
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

	if cfg.App.Env == envProduction && cfg.Worker.AllowPrivateTargets {
		return nil, fmt.Errorf(
			"%w: WORKER_ALLOW_PRIVATE_TARGETS must be false in production",
			errInvalidConfig,
		)
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
