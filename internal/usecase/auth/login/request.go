package loginuser

import (
	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

// Request — входные данные входа.
type Request struct {
	Email    string
	Password string
}

// validate нормализует email; требования к паролю при входе не проверяются
// по существу — важно лишь, что он непустой, остальное решает сравнение хеша.
func (req *Request) validate() error {
	email, err := validate.Email(req.Email)
	if err != nil {
		return err
	}

	req.Email = email

	if req.Password == "" {
		return domain.ErrInvalidCredentials
	}

	return nil
}
