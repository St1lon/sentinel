package loginuser

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type UserRepo interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type PasswordHasher interface {
	Compare(hash, password string) error
}

type TokenIssuer interface {
	Issue(userID string) (string, time.Time, error)
}
