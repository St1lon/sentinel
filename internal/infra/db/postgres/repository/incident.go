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

const incidentColumns = `id, monitor_id, started_at, resolved_at, cause`

type IncidentRepo struct {
	pool *pgxpool.Pool
}

func NewIncidentRepo(pool *pgxpool.Pool) *IncidentRepo {
	return &IncidentRepo{pool: pool}
}

func (r *IncidentRepo) Open(ctx context.Context, incident *domain.Incident) error {
	const query = `
		INSERT INTO incidents (id, monitor_id, started_at, cause)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING`

	_, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query,
		incident.ID, incident.MonitorID, incident.StartedAt, incident.Cause,
	)
	if err != nil {
		if isForeignKeyViolation(err) {
			return domain.ErrMonitorNotFound
		}

		return fmt.Errorf("insert incident: %w", err)
	}

	return nil
}

func (r *IncidentRepo) ResolveOpen(ctx context.Context, monitorID string, resolvedAt time.Time) error {
	const query = `
		UPDATE incidents
		SET resolved_at = $2
		WHERE monitor_id = $1 AND resolved_at IS NULL`

	if _, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query, monitorID, resolvedAt); err != nil {
		return fmt.Errorf("resolve incident: %w", err)
	}

	return nil
}

func (r *IncidentRepo) GetOpenByMonitor(ctx context.Context, monitorID string) (*domain.Incident, error) {
	query := `SELECT ` + incidentColumns + `
		FROM incidents
		WHERE monitor_id = $1 AND resolved_at IS NULL`

	var incident domain.Incident

	err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, query, monitorID).Scan(
		&incident.ID, &incident.MonitorID, &incident.StartedAt, &incident.ResolvedAt, &incident.Cause,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil //nolint:nilnil // отсутствие открытого инцидента — нормальное состояние
	}

	if err != nil {
		return nil, fmt.Errorf("select open incident: %w", err)
	}

	return &incident, nil
}

func (r *IncidentRepo) ListByMonitor(
	ctx context.Context, monitorID string, limit, offset int,
) ([]*domain.Incident, int, error) {
	query := `SELECT ` + incidentColumns + `
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`

	incidents, err := r.queryMany(ctx, query, monitorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	const countQuery = `SELECT count(*) FROM incidents WHERE monitor_id = $1`

	var total int

	if err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, countQuery, monitorID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count incidents: %w", err)
	}

	return incidents, total, nil
}

func (r *IncidentRepo) ListByMonitors(
	ctx context.Context, monitorIDs []string, since time.Time, limit int,
) ([]*domain.Incident, error) {
	if len(monitorIDs) == 0 {
		return []*domain.Incident{}, nil
	}

	query := `SELECT ` + incidentColumns + `
		FROM incidents
		WHERE monitor_id = ANY($1) AND started_at >= $2
		ORDER BY started_at DESC
		LIMIT $3`

	return r.queryMany(ctx, query, monitorIDs, since, limit)
}

func (r *IncidentRepo) queryMany(ctx context.Context, query string, args ...any) ([]*domain.Incident, error) {
	rows, err := postgres.GetQuerier(ctx, r.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]*domain.Incident, 0)

	for rows.Next() {
		var incident domain.Incident

		err := rows.Scan(
			&incident.ID, &incident.MonitorID, &incident.StartedAt, &incident.ResolvedAt, &incident.Cause,
		)
		if err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}

		incidents = append(incidents, &incident)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate incidents: %w", err)
	}

	return incidents, nil
}
