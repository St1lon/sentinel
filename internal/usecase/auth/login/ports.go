package loginuser

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// UserRepo — порт чтения пользователя по адресу.
type UserRepo interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// PasswordHasher — порт сверки пароля с хешем.
type PasswordHasher interface {
	Compare(hash, password string) error
}

// TokenIssuer — порт выдачи access-токена.
type TokenIssuer interface {
	Issue(userID string) (string, time.Time, error)
}
