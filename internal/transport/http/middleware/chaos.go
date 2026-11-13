package middleware

import (
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/St1lon/sentinel/internal/transport/http/apierrors"
	"github.com/St1lon/sentinel/internal/transport/http/dto"
)

// chaosPathPrefix — сбои вносятся только в запросы пользователя. Проверки
// живости не трогаются намеренно: 503 на /healthz перезапустит под, и вместо
// управляемой деградации получится падение.
const chaosPathPrefix = "/api/"

const readyzPath = "/readyz"

// Предохранители демонстрационных режимов. Утечка и раздутая память должны
// довести алерт до срабатывания, но не убить под посреди скринкаста.
const (
	// leakedGoroutinesCap — потолок «утекших» горутин: ~20 МБ стеков,
	// далеко за порогом алерта (200) и далеко от отказа процесса.
	leakedGoroutinesCap = 5000

	// leakedGoroutineTTL — «утекшая» горутина через 10 минут всё же
	// умирает: слоты предохранителя освобождаются для повторной демонстрации.
	leakedGoroutineTTL = 10 * time.Minute

	// hoggedBytesCap — потолок удерживаемых аллокаций. Выше порога
	// memory-алерта (~150 МБ RSS), но ниже лимита пода с учётом GOGC.
	hoggedBytesCap = 128 << 20 // 128 MiB
)

type ChaosOptions struct {
	ErrorRate   float64
	Latency     time.Duration
	LatencyRate float64
	ReadyzFail  bool

	// GoroutineLeak — сколько горутин спавнить на каждый запрос /api/,
	// MemHogBytes — сколько байт удерживать на каждый запрос.
	GoroutineLeak int
	MemHogBytes   int64
}

// Chaos вносит в ответы API управляемые отказы и задержки. Нужен для
// демонстрации: SLI должен меняться от поведения приложения, а не от
// правок запросов. Ставится последним в цепочке, чтобы внесённый сбой
// попал и в метрики, и в логи.
func Chaos(opts ChaosOptions, logger *slog.Logger) func(http.Handler) http.Handler {
	logger.Warn("chaos injection enabled",
		slog.Float64("error_rate", opts.ErrorRate),
		slog.Duration("latency", opts.Latency),
		slog.Float64("latency_rate", opts.LatencyRate),
		slog.Bool("readyz_fail", opts.ReadyzFail),
		slog.Int("goroutine_leak", opts.GoroutineLeak),
		slog.Int64("mem_hog_bytes", opts.MemHogBytes),
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if opts.ReadyzFail && r.URL.Path == readyzPath {
				writeChaosError(w, r, "readiness probe failed by chaos injection")

				return
			}

			if !strings.HasPrefix(r.URL.Path, chaosPathPrefix) {
				next.ServeHTTP(w, r)

				return
			}

			// Ресурсные режимы срабатывают на каждом /api/ запросе: и на
			// успешном, и на отложенном, и на сломанном — утечка не должна
			// зависеть от того, какой из остальных режимов включён.
			if opts.GoroutineLeak > 0 {
				leakGoroutines(opts.GoroutineLeak)
			}

			if opts.MemHogBytes > 0 {
				hogMemory(opts.MemHogBytes)
			}

			if opts.Latency > 0 && hits(opts.LatencyRate) {
				select {
				case <-time.After(opts.Latency):
				case <-r.Context().Done():
					return
				}
			}

			if hits(opts.ErrorRate) {
				writeChaosError(w, r, "request failed by chaos injection")

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Состояние ресурсных режимов — на уровне пакета, а не ChaosOptions:
// middleware создаётся один раз на старте процесса, и накапливать ресурс
// нужно в одном месте, чтобы предохранитель видел весь объём.
var (
	leakedGoroutines atomic.Int64

	hogMu         sync.Mutex
	hoggedBuffers [][]byte
	hoggedBytes   atomic.Int64
)

// leakGoroutines спавнит n горутин, которые ничего не делают и висят
// до TTL. Имитация утечки: где-то в коде запросы оставили после себя
// горутину, и их число только растёт.
func leakGoroutines(n int) {
	for range n {
		if leakedGoroutines.Add(1) > leakedGoroutinesCap {
			leakedGoroutines.Add(-1)

			return
		}

		go func() {
			defer leakedGoroutines.Add(-1)
			time.Sleep(leakedGoroutineTTL)
		}()
	}
}

// hogMemory аллоцирует и удерживает n байт на каждый вызов. Ссылку держит
// пакетная переменная: сборщик мусора буфер вернуть не может. Пока среда
// выполнения не трогает новые страницы, ОС их не выделяет, поэтому буфер
// заполняется: без записи рост RSS завис бы на уровне аллокаций.
func hogMemory(n int64) {
	hogMu.Lock()
	defer hogMu.Unlock()

	if hoggedBytes.Load()+n > hoggedBytesCap {
		return
	}

	buffer := make([]byte, n)
	for i := range buffer {
		buffer[i] = byte(i)
	}

	hoggedBuffers = append(hoggedBuffers, buffer)
	hoggedBytes.Store(hoggedBytes.Load() + n)
}

// hits — выпал ли случай с заданной вероятностью. Генератор не
// криптографический, и здесь это ровно то, что нужно.
func hits(rate float64) bool {
	if rate <= 0 {
		return false
	}

	return rand.Float64() < rate //nolint:gosec // вносим сбои, а не генерируем секреты
}

func writeChaosError(w http.ResponseWriter, r *http.Request, description string) {
	resolveRoute(r)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)

	_ = json.NewEncoder(w).Encode(dto.ErrorResponse{
		Code:        apierrors.CodeServiceUnavailable,
		Description: description,
	})
}

// resolveRoute проставляет в контексте chi шаблон маршрута. Прерванный запрос
// до маршрутизации не доходит, и без этого метрика припишет его метке
// unmatched, которая из SLI доступности исключена: внесённый сбой стал бы
// на графиках невидим.
func resolveRoute(r *http.Request) {
	routeCtx := chi.RouteContext(r.Context())
	if routeCtx == nil || routeCtx.Routes == nil || len(routeCtx.RoutePatterns) > 0 {
		return
	}

	matched := chi.NewRouteContext()
	if routeCtx.Routes.Match(matched, r.Method, r.URL.Path) {
		routeCtx.RoutePatterns = matched.RoutePatterns
	}
}
