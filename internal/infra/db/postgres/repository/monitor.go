package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
)

const monitorColumns = `
	id, user_id, name, kind, target, method,
	interval_seconds, timeout_seconds, expected_status, failure_threshold,
	is_public, paused, status, consecutive_failures,
	last_checked_at, next_check_at, created_at, updated_at`

const monitorColumnsQualified = `
	m.id, m.user_id, m.name, m.kind, m.target, m.method,
	m.interval_seconds, m.timeout_seconds, m.expected_status, m.failure_threshold,
	m.is_public, m.paused, m.status, m.consecutive_failures,
	m.last_checked_at, m.next_check_at, m.created_at, m.updated_at`

type MonitorRepo struct {
	pool *pgxpool.Pool
}

func NewMonitorRepo(pool *pgxpool.Pool) *MonitorRepo {
	return &MonitorRepo{pool: pool}
}

func (r *MonitorRepo) Create(ctx context.Context, monitor *domain.Monitor) error {
	const query = `
		INSERT INTO monitors (
			id, user_id, name, kind, target, method,
			interval_seconds, timeout_seconds, expected_status, failure_threshold,
			is_public, paused, status, consecutive_failures,
			next_check_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4::monitor_kind, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13::monitor_status, 0,
			$14, $15, $16
		)`

	_, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query,
		monitor.ID, monitor.UserID, monitor.Name, string(monitor.Kind), monitor.Target, monitor.Method,
		monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.ExpectedStatus, monitor.FailureThreshold,
		monitor.IsPublic, monitor.Paused, string(monitor.Status),
		monitor.NextCheckAt, monitor.CreatedAt, monitor.UpdatedAt,
	)
	if err != nil {
		return r.mapWriteError(err)
	}

	return nil
}

func (r *MonitorRepo) GetByIDForUser(ctx context.Context, id, userID string) (*domain.Monitor, error) {
	query := `SELECT ` + monitorColumns + `
		FROM monitors
		WHERE id = $1 AND user_id = $2`

	var monitor domain.Monitor

	err := scanMonitor(postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, query, id, userID), &monitor)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrMonitorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("select monitor: %w", err)
	}

	return &monitor, nil
}

func (r *MonitorRepo) ListByUser(
	ctx context.Context, userID string, limit, offset int,
) ([]*domain.Monitor, int, error) {
	query := `SELECT ` + monitorColumns + `
		FROM monitors
		WHERE user_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3`

	monitors, err := r.queryMany(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	const countQuery = `SELECT count(*) FROM monitors WHERE user_id = $1`

	var total int

	if err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count monitors: %w", err)
	}

	return monitors, total, nil
}

func (r *MonitorRepo) ListPublicByUser(ctx context.Context, userID string) ([]*domain.Monitor, error) {
	query := `SELECT ` + monitorColumns + `
		FROM monitors
		WHERE user_id = $1 AND is_public = TRUE
		ORDER BY name`

	return r.queryMany(ctx, query, userID)
}

func (r *MonitorRepo) Update(ctx context.Context, monitor *domain.Monitor) error {
	const query = `
		UPDATE monitors SET
			name = $2,
			kind = $3::monitor_kind,
			target = $4,
			method = $5,
			interval_seconds = $6,
			timeout_seconds = $7,
			expected_status = $8,
			failure_threshold = $9,
			is_public = $10,
			paused = $11,
			status = $12::monitor_status,
			consecutive_failures = $13,
			next_check_at = $14,
			updated_at = $15
		WHERE id = $1`

	tag, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query,
		monitor.ID, monitor.Name, string(monitor.Kind), monitor.Target, monitor.Method,
		monitor.IntervalSeconds, monitor.TimeoutSeconds, monitor.ExpectedStatus, monitor.FailureThreshold,
		monitor.IsPublic, monitor.Paused, string(monitor.Status), monitor.ConsecutiveFailures,
		monitor.NextCheckAt, monitor.UpdatedAt,
	)
	if err != nil {
		return r.mapWriteError(err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrMonitorNotFound
	}

	return nil
}

func (r *MonitorRepo) Delete(ctx context.Context, id, userID string) error {
	const query = `DELETE FROM monitors WHERE id = $1 AND user_id = $2`

	tag, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrMonitorNotFound
	}

	return nil
}

// SKIP LOCKED и сдвиг next_check_at в одном запросе: один монитор не достанется двум воркерам.
func (r *MonitorRepo) LeaseDue(ctx context.Context, now time.Time, limit int) ([]*domain.Monitor, error) {
	query := `
		WITH due AS (
			SELECT id
			FROM monitors
			WHERE paused = FALSE AND next_check_at <= $1
			ORDER BY next_check_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE monitors AS m
		SET next_check_at = $1 + make_interval(secs => m.interval_seconds),
		    updated_at = $1
		FROM due
		WHERE m.id = due.id
		RETURNING ` + monitorColumnsQualified

	monitors, err := r.queryMany(ctx, query, now, limit)
	if err != nil {
		return nil, fmt.Errorf("lease due monitors: %w", err)
	}

	return monitors, nil
}

func (r *MonitorRepo) ApplyProbeResult(
	ctx context.Context,
	monitorID string,
	status domain.MonitorStatus,
	consecutiveFailures int,
	checkedAt time.Time,
) error {
	const query = `
		UPDATE monitors
		SET status = $2::monitor_status,
		    consecutive_failures = $3,
		    last_checked_at = $4,
		    updated_at = $4
		WHERE id = $1`

	tag, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query,
		monitorID, string(status), consecutiveFailures, checkedAt,
	)
	if err != nil {
		return fmt.Errorf("apply probe result: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrMonitorNotFound
	}

	return nil
}

func (r *MonitorRepo) queryMany(ctx context.Context, query string, args ...any) ([]*domain.Monitor, error) {
	rows, err := postgres.GetQuerier(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*domain.Monitor, 0)

	for rows.Next() {
		var monitor domain.Monitor

		if err := scanMonitor(rows, &monitor); err != nil {
			return nil, fmt.Errorf("scan monitor: %w", err)
		}

		monitors = append(monitors, &monitor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitors: %w", err)
	}

	return monitors, nil
}

func (r *MonitorRepo) mapWriteError(err error) error {
	if isUniqueViolation(err, constraintMonitorsNameUniquePerUse) {
		return domain.ErrMonitorNameTaken
	}

	if isForeignKeyViolation(err) {
		return domain.ErrUserNotFound
	}

	if isCheckViolation(err) {
		return fmt.Errorf("%w: %w", domain.ErrInvalidTarget, err)
	}

	return fmt.Errorf("write monitor: %w", err)
}

type scanRow interface {
	Scan(dest ...any) error
}

func scanMonitor(row scanRow, monitor *domain.Monitor) error {
	return row.Scan(
		&monitor.ID, &monitor.UserID, &monitor.Name, &monitor.Kind, &monitor.Target, &monitor.Method,
		&monitor.IntervalSeconds, &monitor.TimeoutSeconds, &monitor.ExpectedStatus, &monitor.FailureThreshold,
		&monitor.IsPublic, &monitor.Paused, &monitor.Status, &monitor.ConsecutiveFailures,
		&monitor.LastCheckedAt, &monitor.NextCheckAt, &monitor.CreatedAt, &monitor.UpdatedAt,
	)
}
