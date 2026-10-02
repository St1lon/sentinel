package getuser

import (
	"context"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

type Request struct {
	UserID string
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	return nil
}

type Usecase struct {
	users UserRepo
}

func NewUsecase(users UserRepo) *Usecase {
	return &Usecase{users: users}
}

func (uc *Usecase) Execute(ctx context.Context, req *Request) (*domain.User, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	return uc.users.GetByID(ctx, req.UserID)
}
