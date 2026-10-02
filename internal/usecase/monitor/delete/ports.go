package deletemonitor

import "context"

// MonitorRepo — порт удаления монитора с проверкой владельца.
type MonitorRepo interface {
	Delete(ctx context.Context, id, userID string) error
}
