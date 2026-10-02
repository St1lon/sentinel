package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgCodeUniqueViolation     = "23505"
	pgCodeForeignKeyViolation = "23503"
	pgCodeCheckViolation      = "23514"
)

const (
	constraintUsersEmailUnique         = "users_email_unique"
	constraintUsersSlugUnique          = "users_status_page_slug_unique"
	constraintMonitorsNameUniquePerUse = "monitors_name_unique_per_user"
)

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeUniqueViolation && pgErr.ConstraintName == constraint
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeCheckViolation
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeForeignKeyViolation
}
