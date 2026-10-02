package getuser

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

// UserRepo — порт чтения пользователя по идентификатору.
type UserRepo interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}
