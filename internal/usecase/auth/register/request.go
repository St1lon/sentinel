package registeruser

import "github.com/St1lon/sentinel/internal/usecase/common/validate"

type Request struct {
	Email    string
	Password string
}

func (req *Request) validate() error {
	email, err := validate.Email(req.Email)
	if err != nil {
		return err
	}

	req.Email = email

	return validate.Password(req.Password)
}
