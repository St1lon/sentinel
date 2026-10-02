package registeruser

import "github.com/St1lon/sentinel/internal/usecase/common/validate"

// Request — входные данные регистрации.
type Request struct {
	Email    string
	Password string
}

// validate нормализует email и проверяет требования к паролю.
func (req *Request) validate() error {
	email, err := validate.Email(req.Email)
	if err != nil {
		return err
	}

	req.Email = email

	return validate.Password(req.Password)
}
