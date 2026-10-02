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

const slugEntropyBytes = 10

type Result struct {
	User      *domain.User
	Token     string
	ExpiresAt time.Time
}

type Usecase struct {
	users  UserRepo
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewUsecase(users UserRepo, hasher PasswordHasher, tokens TokenIssuer) *Usecase {
	return &Usecase{users: users, hasher: hasher, tokens: tokens}
}

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

func newStatusPageSlug() (string, error) {
	buf := make([]byte, slugEntropyBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate status page slug: %w", err)
	}

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)

	return strings.ToLower(encoded), nil
}
