// Package metrics — реестр метрик, адаптеры под порты вызывающих пакетов
// и отправка метрик разовой задачи push-моделью.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// namespace — общий префикс всех метрик приложения.
const namespace = "sentinel"

// Registry — реестр метрик одного процесса. Собственный, а не
// prometheus.DefaultRegisterer: в выдачу попадает только зарегистрированное
// явно. Метка service совпадает с полем service.name в логах.
type Registry struct {
	registry   *prometheus.Registry
	registerer prometheus.Registerer
}

// NewRegistry — реестр долгоживущего процесса: вместе с бизнес-метриками
// отдаются метрики рантайма Go и процесса.
func NewRegistry(service string) *Registry {
	reg := newRegistry(service)

	reg.registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return reg
}

// NewJobRegistry — реестр разовой задачи. Метрик рантайма в нём нет:
// секунды жизни процесса, который запустился, посчитал и вышел, ничего
// не говорят о состоянии системы.
func NewJobRegistry(service string) *Registry {
	return newRegistry(service)
}

func newRegistry(service string) *Registry {
	registry := prometheus.NewRegistry()

	return &Registry{
		registry:   registry,
		registerer: prometheus.WrapRegistererWith(prometheus.Labels{"service": service}, registry),
	}
}

// Registerer отдаётся конструкторам адаптеров метрик.
func (r *Registry) Registerer() prometheus.Registerer {
	return r.registerer
}

// Gatherer нужен отправке метрик push-моделью.
func (r *Registry) Gatherer() prometheus.Gatherer {
	return r.registry
}

// Handler отдаёт метрики в текстовом формате OpenMetrics. ContinueOnError:
// коллектор процесса вне Linux возвращает ошибку, и из-за неё не должна
// пропадать остальная выдача.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
	})
}
