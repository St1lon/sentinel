package registeruser

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

type UserRepo interface {
	Create(ctx context.Context, user *domain.User) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type TokenIssuer interface {
	Issue(userID string) (string, time.Time, error)
}
