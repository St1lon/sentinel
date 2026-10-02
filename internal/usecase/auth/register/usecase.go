// Package registeruser реализует регистрацию пользователя.
package registeruser

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/St1lon/sentinel/internal/domain"
)

// slugEntropyBytes — размер случайной части слага публичной статус-страницы.
// 10 байт ≈ 80 бит: слаг является capability-ссылкой, поэтому угадываться не должен.
const slugEntropyBytes = 10

// Result — результат регистрации: пользователь и сразу выданный токен,
// чтобы клиенту не требовался отдельный вход после регистрации.
type Result struct {
	User      *domain.User
	Token     string
	ExpiresAt time.Time
}

// Usecase — регистрация пользователя.
type Usecase struct {
	users  UserRepo
	hasher PasswordHasher
	tokens TokenIssuer
}

// NewUsecase собирает usecase регистрации.
func NewUsecase(users UserRepo, hasher PasswordHasher, tokens TokenIssuer) *Usecase {
	return &Usecase{users: users, hasher: hasher, tokens: tokens}
}

// Execute регистрирует пользователя и выдаёт ему токен.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*Result, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	hash, err := uc.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	slug, err := newStatusPageSlug()
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:             uuid.NewString(),
		Email:          req.Email,
		PasswordHash:   hash,
		StatusPageSlug: slug,
		CreatedAt:      time.Now().UTC(),
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	token, expiresAt, err := uc.tokens.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &Result{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

// newStatusPageSlug генерирует непредсказуемый слаг публичной статус-страницы.
func newStatusPageSlug() (string, error) {
	buf := make([]byte, slugEntropyBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate status page slug: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)

	return strings.ToLower(encoded), nil
}
