package deletemonitor

import "context"

type MonitorRepo interface {
	Delete(ctx context.Context, id, userID string) error
}
