package loginuser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/St1lon/sentinel/internal/domain"
)

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
		return nil, domain.ErrInvalidCredentials
	}

	user, err := uc.users.GetByEmail(ctx, req.Email)

	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, domain.ErrInvalidCredentials
	}

	if err != nil {
		return nil, err
	}

	if err := uc.hasher.Compare(user.PasswordHash, req.Password); err != nil {
		return nil, err
	}

	token, expiresAt, err := uc.tokens.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &Result{User: user, Token: token, ExpiresAt: expiresAt}, nil
}
