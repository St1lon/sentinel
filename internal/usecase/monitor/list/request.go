package listmonitors

import (
	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/usecase/common/validate"
)

type Request struct {
	UserID string
	Limit  int
	Offset int
}

func (req *Request) validate() error {
	if !validate.UUID(req.UserID) {
		return domain.ErrUnauthenticated
	}

	limit, offset, err := validate.Paging(req.Limit, req.Offset)
	if err != nil {
		return err
	}

	req.Limit, req.Offset = limit, offset

	return nil
}
