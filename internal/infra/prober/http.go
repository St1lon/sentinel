package prober

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// Options — параметры HTTP-prober'а. Инфраструктура не зависит от пакета config:
// composition root сам перекладывает конфигурацию в эту структуру.
type Options struct {
	UserAgent        string
	MaxResponseBytes int64
	MaxRedirects     int
	AllowPrivate     bool
}

// HTTPProber выполняет HTTP-проверку цели.
type HTTPProber struct {
	client    *http.Client
	guard     *Guard
	userAgent string
	maxBody   int64
	now       func() time.Time
}

// NewHTTPProber собирает prober с диалером, проверяющим адрес перед соединением.
func NewHTTPProber(opts Options) *HTTPProber {
	guard := NewGuard(opts.AllowPrivate)

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			return guard.CheckDialAddress(address)
		},
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > opts.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", opts.MaxRedirects)
			}
			// Каждый редирект проверяется заново: цель могла увести на внутренний адрес.
			if _, err := guard.CheckURL(req.URL.String()); err != nil {
				return err
			}

			return nil
		},
	}

	return &HTTPProber{
		client:    client,
		guard:     guard,
		userAgent: opts.UserAgent,
		maxBody:   opts.MaxResponseBytes,
		now:       time.Now,
	}
}

// Probe выполняет одну проверку монитора и возвращает её результат.
//
// Неудача проверки — это нормальный результат (Check.Up = false), а не ошибка:
// ошибка возвращается только когда проверку невозможно выполнить в принципе
// (неподдерживаемый вид монитора).
func (p *HTTPProber) Probe(ctx context.Context, monitor *domain.Monitor) (*domain.Check, error) {
	if monitor.Kind != domain.MonitorKindHTTP {
		return nil, fmt.Errorf("%w: %s is not implemented", domain.ErrInvalidMonitorKind, monitor.Kind)
	}

	startedAt := p.now()

	target, err := p.guard.CheckURL(monitor.Target)
	if err != nil {
		return p.failedCheck(monitor, startedAt, err), nil
	}

	probeCtx, cancel := context.WithTimeout(ctx, monitor.Timeout())
	defer cancel()

	request, err := http.NewRequestWithContext(probeCtx, monitor.Method, target.String(), nil)
	if err != nil {
		return p.failedCheck(monitor, startedAt, err), nil
	}

	request.Header.Set("User-Agent", p.userAgent)
	request.Header.Set("Accept", "*/*")

	response, err := p.client.Do(request)
	if err != nil {
		return p.failedCheck(monitor, startedAt, unwrapProbeError(err)), nil
	}
	defer func() { _ = response.Body.Close() }()

	// Тело вычитывается ограниченно: цель может отдавать гигабайты,
	// а латентность должна включать получение ответа, а не только заголовков.
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, p.maxBody)); err != nil {
		return p.failedCheck(monitor, startedAt, unwrapProbeError(err)), nil
	}

	latency := p.now().Sub(startedAt)
	statusCode := response.StatusCode

	check := &domain.Check{
		MonitorID:  monitor.ID,
		CheckedAt:  startedAt.UTC(),
		Up:         statusCode == monitor.ExpectedStatus,
		StatusCode: &statusCode,
		LatencyMS:  int(latency.Milliseconds()),
	}

	if !check.Up {
		message := fmt.Sprintf("unexpected status %d, expected %d", statusCode, monitor.ExpectedStatus)
		check.Error = &message
	}

	return check, nil
}

func (p *HTTPProber) failedCheck(monitor *domain.Monitor, startedAt time.Time, cause error) *domain.Check {
	message := cause.Error()
	latency := p.now().Sub(startedAt)

	return &domain.Check{
		MonitorID: monitor.ID,
		CheckedAt: startedAt.UTC(),
		Up:        false,
		LatencyMS: int(latency.Milliseconds()),
		Error:     &message,
	}
}

// unwrapProbeError превращает ошибку http-клиента в короткое сообщение,
// пригодное для показа на статус-странице.
func unwrapProbeError(err error) error {
	var urlErr *net.OpError
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s %s: %w", urlErr.Op, urlErr.Net, urlErr.Err)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("timeout exceeded")
	}

	return err
}
