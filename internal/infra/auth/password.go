// Package auth содержит инфраструктуру аутентификации: хеширование паролей
// и выдачу/проверку JWT. Слой usecase видит только узкие интерфейсы Hasher
// и TokenIssuer, объявленные рядом с их потребителями.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/St1lon/sentinel/internal/domain"
)

// BcryptHasher хеширует и проверяет пароли через bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создаёт хешер с заданной стоимостью.
// Стоимость приходит из конфигурации: в тестах она минимальна, в проде — выше.
func NewBcryptHasher(cost int) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash возвращает bcrypt-хеш пароля.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

// Compare сверяет пароль с хешем. Несовпадение — доменная ошибка
// ErrInvalidCredentials, а не техническая: наружу уйдёт 401, а не 500.
func (h *BcryptHasher) Compare(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return domain.ErrInvalidCredentials
	}

	if err != nil {
		return fmt.Errorf("compare password: %w", err)
	}

	return nil
}
