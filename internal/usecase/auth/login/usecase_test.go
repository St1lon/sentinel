package loginuser_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	loginuser "github.com/St1lon/sentinel/internal/usecase/auth/login"
)

type userRepoStub struct {
	user       *domain.User
	err        error
	gotEmail   string
	callsCount int
}

func (s *userRepoStub) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	s.gotEmail = email
	s.callsCount++

	return s.user, s.err
}

type hasherStub struct {
	err error
}

func (s *hasherStub) Compare(string, string) error {
	return s.err
}

type tokenStub struct{}

func (tokenStub) Issue(string) (string, time.Time, error) {
	return "jwt", time.Now().Add(time.Hour), nil
}

func existingUser() *domain.User {
	return &domain.User{ID: "u-1", Email: "user@example.com", PasswordHash: "hash"}
}

func TestExecute_IssuesTokenOnValidCredentials(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{user: existingUser()}
	usecase := loginuser.NewUsecase(repo, &hasherStub{}, tokenStub{})

	result, err := usecase.Execute(context.Background(), &loginuser.Request{
		Email:    "USER@example.com",
		Password: "password123",
	})

	require.NoError(t, err)
	require.Equal(t, "jwt", result.Token)
	require.Equal(t, "user@example.com", repo.gotEmail, "поиск идёт по нормализованному email")
}

func TestExecute_UnknownUserLooksLikeWrongPassword(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{err: domain.ErrUserNotFound}
	usecase := loginuser.NewUsecase(repo, &hasherStub{}, tokenStub{})

	_, err := usecase.Execute(context.Background(), &loginuser.Request{
		Email:    "nobody@example.com",
		Password: "password123",
	})

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestExecute_WrongPasswordRejected(t *testing.T) {
	t.Parallel()

	usecase := loginuser.NewUsecase(
		&userRepoStub{user: existingUser()},
		&hasherStub{err: domain.ErrInvalidCredentials},
		tokenStub{},
	)

	_, err := usecase.Execute(context.Background(), &loginuser.Request{
		Email:    "user@example.com",
		Password: "wrong-password",
	})

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestExecute_InvalidInputDoesNotReachRepository(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{user: existingUser()}
	usecase := loginuser.NewUsecase(repo, &hasherStub{}, tokenStub{})

	for _, req := range []*loginuser.Request{
		{Email: "broken", Password: "password123"},
		{Email: "user@example.com", Password: ""},
	} {
		_, err := usecase.Execute(context.Background(), req)
		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	}

	require.Zero(t, repo.callsCount, "некорректный ввод не должен доходить до БД")
}
