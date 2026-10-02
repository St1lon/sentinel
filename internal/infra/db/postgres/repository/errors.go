// Package repository содержит реализации репозиториев над PostgreSQL.
// Ошибки драйвера транслируются в sentinel-ошибки домена здесь, а не выше:
// слой usecase не знает кодов PostgreSQL.
package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Коды ошибок PostgreSQL, которые транслируются в доменные.
const (
	pgCodeUniqueViolation     = "23505"
	pgCodeForeignKeyViolation = "23503"
	pgCodeCheckViolation      = "23514"
)

// Имена ограничений из миграции 00001_init.up.sql.
const (
	constraintUsersEmailUnique         = "users_email_unique"
	constraintUsersSlugUnique          = "users_status_page_slug_unique"
	constraintMonitorsNameUniquePerUse = "monitors_name_unique_per_user"
)

// isUniqueViolation сообщает, что ошибка — нарушение уникальности
// по указанному ограничению.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeUniqueViolation && pgErr.ConstraintName == constraint
}

// isCheckViolation сообщает, что ошибка — нарушение CHECK-ограничения.
func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeCheckViolation
}

// isForeignKeyViolation сообщает, что ошибка — нарушение внешнего ключа.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == pgCodeForeignKeyViolation
}
