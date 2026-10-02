package registeruser

import (
	"context"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

// UserRepo — порт хранилища пользователей: только то, что нужно регистрации.
type UserRepo interface {
	Create(ctx context.Context, user *domain.User) error
}

// PasswordHasher — порт хеширования паролей.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// TokenIssuer — порт выдачи access-токена.
type TokenIssuer interface {
	Issue(userID string) (string, time.Time, error)
}
