package registeruser_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/St1lon/sentinel/internal/domain"
	registeruser "github.com/St1lon/sentinel/internal/usecase/auth/register"
)

type userRepoStub struct {
	created *domain.User
	err     error
}

func (s *userRepoStub) Create(_ context.Context, user *domain.User) error {
	if s.err != nil {
		return s.err
	}

	s.created = user

	return nil
}

type hasherStub struct {
	hash string
	err  error
}

func (s *hasherStub) Hash(string) (string, error) {
	return s.hash, s.err
}

type tokenStub struct {
	token string
	err   error
}

func (s *tokenStub) Issue(string) (string, time.Time, error) {
	return s.token, time.Now().Add(time.Hour), s.err
}

func TestExecute_CreatesUserAndIssuesToken(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{}
	usecase := registeruser.NewUsecase(repo, &hasherStub{hash: "hashed"}, &tokenStub{token: "jwt"})

	result, err := usecase.Execute(context.Background(), &registeruser.Request{
		Email:    "  User@Example.COM ",
		Password: "password123",
	})

	require.NoError(t, err)
	require.Equal(t, "jwt", result.Token)
	require.Equal(t, "user@example.com", result.User.Email, "email нормализуется к нижнему регистру")
	require.Equal(t, "hashed", repo.created.PasswordHash, "в репозиторий уходит хеш, не пароль")
	require.NotEmpty(t, repo.created.ID)
	require.NotEmpty(t, repo.created.StatusPageSlug)
	require.False(t, repo.created.CreatedAt.IsZero())
}

func TestExecute_StatusPageSlugIsUnpredictable(t *testing.T) {
	t.Parallel()

	slugs := make(map[string]struct{}, 20)

	for range 20 {
		repo := &userRepoStub{}
		usecase := registeruser.NewUsecase(repo, &hasherStub{hash: "h"}, &tokenStub{token: "t"})

		_, err := usecase.Execute(context.Background(), &registeruser.Request{
			Email:    "user@example.com",
			Password: "password123",
		})
		require.NoError(t, err)

		slugs[repo.created.StatusPageSlug] = struct{}{}
	}

	require.Len(t, slugs, 20, "слаг публичной страницы не должен повторяться")
}

func TestExecute_RejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	usecase := registeruser.NewUsecase(&userRepoStub{}, &hasherStub{}, &tokenStub{})

	for _, email := range []string{"", "not-an-email", "user@", "@example.com", "user@localhost"} {
		_, err := usecase.Execute(context.Background(), &registeruser.Request{
			Email:    email,
			Password: "password123",
		})

		require.ErrorIs(t, err, domain.ErrInvalidEmail, email)
	}
}

func TestExecute_RejectsWeakPassword(t *testing.T) {
	t.Parallel()

	usecase := registeruser.NewUsecase(&userRepoStub{}, &hasherStub{}, &tokenStub{})

	_, err := usecase.Execute(context.Background(), &registeruser.Request{
		Email:    "user@example.com",
		Password: "short",
	})

	require.ErrorIs(t, err, domain.ErrWeakPassword)
}

func TestExecute_PropagatesDuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := &userRepoStub{err: domain.ErrEmailAlreadyUsed}
	usecase := registeruser.NewUsecase(repo, &hasherStub{hash: "h"}, &tokenStub{token: "t"})

	_, err := usecase.Execute(context.Background(), &registeruser.Request{
		Email:    "user@example.com",
		Password: "password123",
	})

	require.ErrorIs(t, err, domain.ErrEmailAlreadyUsed)
}

func TestExecute_WrapsHasherFailure(t *testing.T) {
	t.Parallel()

	usecase := registeruser.NewUsecase(
		&userRepoStub{},
		&hasherStub{err: errors.New("bcrypt is unhappy")},
		&tokenStub{},
	)

	_, err := usecase.Execute(context.Background(), &registeruser.Request{
		Email:    "user@example.com",
		Password: "password123",
	})

	require.ErrorContains(t, err, "hash password")
}
