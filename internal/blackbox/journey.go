// Package blackbox — пробер критического пользовательского пути: ходит
// в наше же API снаружи и отдаёт результат каждого шага метриками SLI.
package blackbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	StepHealthz    = "healthz"
	StepReadyz     = "readyz"
	StepLogin      = "login"
	StepMonitors   = "monitors"
	StepStatusPage = "status_page"
)

const maxResponseBytes = 64 << 10

var (
	errUnexpectedStatus = errors.New("unexpected status")
	errEmptyToken       = errors.New("login response has no token")
)

type Metrics interface {
	StepFinished(step string, success bool, duration time.Duration)
	JourneyFinished(success bool)
}

type Options struct {
	TargetURL     string
	Interval      time.Duration
	Timeout       time.Duration
	LoginEmail    string
	LoginPassword string
	StatusSlug    string
	UserAgent     string
}

type Journey struct {
	client  *http.Client
	metrics Metrics
	logger  *slog.Logger
	opts    Options
}

func New(metrics Metrics, logger *slog.Logger, opts Options) *Journey {
	return &Journey{
		client:  &http.Client{Timeout: opts.Timeout},
		metrics: metrics,
		logger:  logger,
		opts:    opts,
	}
}

func (j *Journey) Run(ctx context.Context) error {
	j.logger.Info("prober started",
		slog.String("target", j.opts.TargetURL),
		slog.Duration("interval", j.opts.Interval),
		slog.Duration("timeout", j.opts.Timeout),
	)

	ticker := time.NewTicker(j.opts.Interval)
	defer ticker.Stop()

	// Первый прогон сразу: иначе метрик нет до первого тика, и свежий
	// пробер выглядит в графиках как «нет данных».
	j.RunOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			j.logger.Info("prober stopped")

			return nil
		case <-ticker.C:
			j.RunOnce(ctx)
		}
	}
}

// RunOnce прогоняет сценарий один раз. Сценарий успешен только если прошли
// все шаги.
func (j *Journey) RunOnce(ctx context.Context) bool {
	success := true

	for _, request := range []request{
		{name: StepHealthz, method: http.MethodGet, path: "/healthz"},
		{name: StepReadyz, method: http.MethodGet, path: "/readyz"},
	} {
		if !j.call(ctx, request) {
			success = false
		}
	}

	// Список мониторов запрашивается и с пустым токеном: без входа
	// пользователь его тоже не увидит, и шаг обязан стать неуспешным,
	// иначе его gauge застынет на последнем удачном значении.
	token, ok := j.login(ctx)
	if !ok {
		success = false
	}

	if !j.call(ctx, request{
		name:   StepMonitors,
		method: http.MethodGet,
		path:   "/api/v1/monitors",
		token:  token,
	}) {
		success = false
	}

	if !j.statusPage(ctx) {
		success = false
	}

	j.metrics.JourneyFinished(success)

	return success
}

func (j *Journey) login(ctx context.Context) (string, bool) {
	credentials, err := json.Marshal(map[string]string{
		"email":    j.opts.LoginEmail,
		"password": j.opts.LoginPassword,
	})
	if err != nil {
		return "", false
	}

	var token string

	ok := j.call(ctx, request{
		name:   StepLogin,
		method: http.MethodPost,
		path:   "/api/v1/auth/login",
		body:   credentials,
		check: func(payload []byte) error {
			var response struct {
				Token string `json:"token"`
			}

			if err := json.Unmarshal(payload, &response); err != nil {
				return fmt.Errorf("decode login response: %w", err)
			}

			if response.Token == "" {
				return errEmptyToken
			}

			token = response.Token

			return nil
		},
	})

	return token, ok
}

// statusPage пропускается, если slug не задан: он выдаётся при регистрации
// и заранее не известен.
func (j *Journey) statusPage(ctx context.Context) bool {
	if j.opts.StatusSlug == "" {
		return true
	}

	return j.call(ctx, request{
		name:   StepStatusPage,
		method: http.MethodGet,
		path:   "/api/v1/public/status/" + j.opts.StatusSlug,
	})
}

type request struct {
	name   string
	method string
	path   string
	token  string
	body   []byte
	check  func(payload []byte) error
}

func (j *Journey) call(ctx context.Context, req request) bool {
	started := time.Now()

	payload, err := j.do(ctx, req)
	if err == nil && req.check != nil {
		err = req.check(payload)
	}

	duration := time.Since(started)
	j.metrics.StepFinished(req.name, err == nil, duration)

	if err != nil {
		j.logger.WarnContext(ctx, "probe step failed",
			slog.String("step", req.name),
			slog.Duration("duration", duration),
			slog.String("error", err.Error()),
		)

		return false
	}

	j.logger.DebugContext(ctx, "probe step ok",
		slog.String("step", req.name),
		slog.Duration("duration", duration),
	)

	return true
}

func (j *Journey) do(ctx context.Context, req request) ([]byte, error) {
	var body io.Reader
	if req.body != nil {
		body = bytes.NewReader(req.body)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, req.method, j.opts.TargetURL+req.path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpRequest.Header.Set("User-Agent", j.opts.UserAgent)

	if req.body != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}

	if req.token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+req.token)
	}

	response, err := j.client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errUnexpectedStatus, response.StatusCode)
	}

	return payload, nil
}
