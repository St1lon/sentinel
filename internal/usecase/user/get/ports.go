package getuser

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
)

type UserRepo interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}
