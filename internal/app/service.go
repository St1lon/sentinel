package app

// Имена процессов: одно значение уходит и в поле service.name логов,
// и в метку service метрик.
const (
	ServiceAPI     = "sentinel-api"
	ServiceWorker  = "sentinel-worker"
	ServiceCleanup = "sentinel-cleanup"
)
