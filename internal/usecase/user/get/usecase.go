// Package getuser реализует чтение текущего пользователя.
package getuser

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — идентификатор пользователя из токена.
type Request struct {
	UserID string
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	return nil
}

// Usecase — чтение текущего пользователя.
type Usecase struct {
	users UserRepo
}

// NewUsecase собирает usecase чтения пользователя.
func NewUsecase(users UserRepo) *Usecase {
	return &Usecase{users: users}
}

// Execute возвращает пользователя по идентификатору из токена.
func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.User, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.users.GetByID(ctx, req.UserID)
}
