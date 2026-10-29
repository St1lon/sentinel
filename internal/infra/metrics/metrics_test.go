package metrics_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/config"
	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/metrics"
)

const testService = "sentinel-test"

type labels map[string]string

func TestBusiness_SplitsAuthAttemptsByOperationAndResult(t *testing.T) {
	t.Parallel()

	registry := metrics.NewJobRegistry(testService)
	business := metrics.NewBusiness(registry.Registerer())

	business.LoginAttempt(true)
	business.LoginAttempt(false)
	business.LoginAttempt(false)
	business.RegisterAttempt(true)

	const name = "sentinel_auth_attempts_total"

	require.Equal(t, 1.0, value(t, registry, name, labels{"operation": "login", "result": "success"}))
	require.Equal(t, 2.0, value(t, registry, name, labels{"operation": "login", "result": "failure"}))
	require.Equal(t, 1.0, value(t, registry, name, labels{"operation": "register", "result": "success"}))
	require.False(t, present(t, registry, name, labels{"operation": "register", "result": "failure"}),
		"серия появляется только после первого события")
}

func TestRegistry_AddsServiceLabelToEverySeries(t *testing.T) {
	t.Parallel()

	registry := metrics.NewJobRegistry(testService)
	metrics.NewBusiness(registry.Registerer()).MonitorDeleted()

	require.Equal(t, 1.0, value(t, registry, "sentinel_monitors_deleted_total", labels{"service": testService}))
}

func TestProbes_CountsProbesLatencyAndIncidents(t *testing.T) {
	t.Parallel()

	registry := metrics.NewJobRegistry(testService)
	probes := metrics.NewProbes(registry.Registerer())

	probes.ProbeRecorded(domain.MonitorKindHTTP, true, 120*time.Millisecond)
	probes.ProbeRecorded(domain.MonitorKindHTTP, false, 5*time.Second)
	probes.IncidentOpened(domain.MonitorKindHTTP)
	probes.IncidentResolved(domain.MonitorKindHTTP)

	kind := labels{"kind": "http"}

	require.Equal(t, 1.0, value(t, registry, "sentinel_probes_total", labels{"kind": "http", "result": "up"}))
	require.Equal(t, 1.0, value(t, registry, "sentinel_probes_total", labels{"kind": "http", "result": "down"}))
	require.Equal(t, 1.0, value(t, registry, "sentinel_incidents_opened_total", kind))
	require.Equal(t, 1.0, value(t, registry, "sentinel_incidents_resolved_total", kind))

	histogram := find(t, registry, "sentinel_probe_latency_seconds", kind).GetHistogram()
	require.Equal(t, uint64(2), histogram.GetSampleCount())
	require.InDelta(t, 5.12, histogram.GetSampleSum(), 0.001)
}

func TestFleet_ZeroesStatusesMissingInSummary(t *testing.T) {
	t.Parallel()

	registry := metrics.NewJobRegistry(testService)
	fleet := metrics.NewFleet(registry.Registerer())

	fleet.SetFleetSummary(&domain.FleetSummary{
		ByStatus:      map[domain.MonitorStatus]int{domain.MonitorStatusDown: 3},
		OpenIncidents: 3,
	})

	require.Equal(t, 3.0, value(t, registry, "sentinel_monitors", labels{"status": "down"}))
	require.Equal(t, 3.0, value(t, registry, "sentinel_incidents_open", nil))

	// Все мониторы поднялись: в срезе статуса down больше нет, и gauge
	// обязан уйти в ноль, а не остаться на трёх.
	fleet.SetFleetSummary(&domain.FleetSummary{
		ByStatus:      map[domain.MonitorStatus]int{domain.MonitorStatusUp: 3},
		OpenIncidents: 0,
	})

	require.Equal(t, 0.0, value(t, registry, "sentinel_monitors", labels{"status": "down"}))
	require.Equal(t, 3.0, value(t, registry, "sentinel_monitors", labels{"status": "up"}))
	require.Equal(t, 0.0, value(t, registry, "sentinel_incidents_open", nil))
}

func TestRegistry_HandlerServesRuntimeMetrics(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)

	metrics.NewRegistry(testService).Handler().ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "go_goroutines")
	require.Contains(t, recorder.Body.String(), `service="`+testService+`"`)
}

func TestPusher_SendsMetricsWithJobLabel(t *testing.T) {
	t.Parallel()

	var (
		gotQuery string
		gotBody  string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		gotQuery = r.URL.RawQuery
		gotBody = string(body)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	registry := metrics.NewJobRegistry(testService)
	metrics.NewCleanup(registry.Registerer()).Finished(17, time.Second, time.Unix(1700000000, 0))

	pusher, err := metrics.NewPusher(
		&config.MetricsPushConfig{URL: server.URL + "/api/v1/import/prometheus", Timeout: time.Second},
		"sentinel-cleanup",
	)
	require.NoError(t, err)
	require.NoError(t, pusher.Push(context.Background(), registry.Gatherer()))

	require.Equal(t, "extra_label=job%3Dsentinel-cleanup", gotQuery)
	require.Contains(t, gotBody, "sentinel_cleanup_checks_deleted")
	require.Contains(t, gotBody, "17")
}

func TestPusher_ReportsRejectedPush(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	pusher, err := metrics.NewPusher(&config.MetricsPushConfig{URL: server.URL, Timeout: time.Second}, "job")
	require.NoError(t, err)

	err = pusher.Push(context.Background(), metrics.NewJobRegistry(testService).Gatherer())
	require.Error(t, err)
}

func find(t *testing.T, registry *metrics.Registry, name string, want labels) *dto.Metric {
	t.Helper()

	metric := lookup(t, registry, name, want)
	require.NotNil(t, metric, "серия %s%v не найдена", name, want)

	return metric
}

func value(t *testing.T, registry *metrics.Registry, name string, want labels) float64 {
	t.Helper()

	metric := find(t, registry, name, want)

	if gauge := metric.GetGauge(); gauge != nil {
		return gauge.GetValue()
	}

	return metric.GetCounter().GetValue()
}

func present(t *testing.T, registry *metrics.Registry, name string, want labels) bool {
	t.Helper()

	return lookup(t, registry, name, want) != nil
}

func lookup(t *testing.T, registry *metrics.Registry, name string, want labels) *dto.Metric {
	t.Helper()

	families, err := registry.Gatherer().Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}

		for _, metric := range family.GetMetric() {
			if matches(metric, want) {
				return metric
			}
		}
	}

	return nil
}

func matches(metric *dto.Metric, want labels) bool {
	actual := make(labels, len(metric.GetLabel()))

	for _, pair := range metric.GetLabel() {
		actual[pair.GetName()] = pair.GetValue()
	}

	for name, expected := range want {
		if actual[name] != expected {
			return false
		}
	}

	return true
}
