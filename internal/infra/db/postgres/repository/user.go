package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	const query = `
		INSERT INTO users (id, email, password_hash, status_page_slug, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.StatusPageSlug, user.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err, constraintUsersEmailUnique) {
			return domain.ErrEmailAlreadyUsed
		}

		if isUniqueViolation(err, constraintUsersSlugUnique) {
			return fmt.Errorf("status page slug collision: %w", err)
		}

		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, status_page_slug, created_at
		FROM users
		WHERE email = $1`

	return r.queryOne(ctx, query, email)
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, status_page_slug, created_at
		FROM users
		WHERE id = $1`

	return r.queryOne(ctx, query, id)
}

func (r *UserRepo) GetByStatusPageSlug(ctx context.Context, slug string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, status_page_slug, created_at
		FROM users
		WHERE status_page_slug = $1`

	user, err := r.queryOne(ctx, query, slug)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil, domain.ErrStatusPageNotFound
	}

	return user, err
}

func (r *UserRepo) queryOne(ctx context.Context, query string, args ...any) (*domain.User, error) {
	var user domain.User

	err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.StatusPageSlug, &user.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}

	return &user, nil
}
