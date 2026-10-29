package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/St1lon/sentinel/internal/config"
)

// Pusher отправляет метрики в эндпоинт импорта Victoria Metrics. Pull-модель
// для задачи по расписанию не работает: она живёт меньше интервала опроса,
// поэтому результат отдаёт сама, одним POST перед выходом.
type Pusher struct {
	client *http.Client
	url    string
}

func NewPusher(cfg *config.MetricsPushConfig, job string) (*Pusher, error) {
	target, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse metrics push url: %w", err)
	}

	// Метка job выставляется параметром запроса: у push-метрик нет таргета,
	// от которого система мониторинга взяла бы её сама.
	query := target.Query()
	query.Set("extra_label", "job="+job)
	target.RawQuery = query.Encode()

	return &Pusher{
		client: &http.Client{Timeout: cfg.Timeout},
		url:    target.String(),
	}, nil
}

func (p *Pusher) Push(ctx context.Context, gatherer prometheus.Gatherer) error {
	body, err := encode(gatherer)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, body)
	if err != nil {
		return fmt.Errorf("build metrics push request: %w", err)
	}

	request.Header.Set("Content-Type", string(expfmt.NewFormat(expfmt.TypeTextPlain)))

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("push metrics: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s", errPushRejected, response.Status)
	}

	return nil
}

func encode(gatherer prometheus.Gatherer) (*bytes.Buffer, error) {
	families, err := gatherer.Gather()
	if err != nil {
		return nil, fmt.Errorf("gather metrics: %w", err)
	}

	var body bytes.Buffer

	encoder := expfmt.NewEncoder(&body, expfmt.NewFormat(expfmt.TypeTextPlain))

	for _, family := range families {
		if err := encoder.Encode(family); err != nil {
			return nil, fmt.Errorf("encode metrics: %w", err)
		}
	}

	return &body, nil
}
