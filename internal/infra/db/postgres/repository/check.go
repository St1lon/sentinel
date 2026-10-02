package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/St1lon/sentinel/internal/domain"
	"github.com/St1lon/sentinel/internal/infra/db/postgres"
)

type CheckRepo struct {
	pool *pgxpool.Pool
}

func NewCheckRepo(pool *pgxpool.Pool) *CheckRepo {
	return &CheckRepo{pool: pool}
}

func (r *CheckRepo) Insert(ctx context.Context, check *domain.Check) error {
	const query = `
		INSERT INTO checks (monitor_id, checked_at, up, status_code, latency_ms, error)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, query,
		check.MonitorID, check.CheckedAt, check.Up, check.StatusCode, check.LatencyMS, check.Error,
	).Scan(&check.ID)
	if err != nil {
		if isForeignKeyViolation(err) {
			return domain.ErrMonitorNotFound
		}

		return fmt.Errorf("insert check: %w", err)
	}

	return nil
}

func (r *CheckRepo) ListByMonitor(
	ctx context.Context, monitorID string, from, to time.Time, limit int,
) ([]*domain.Check, error) {
	const query = `
		SELECT id, monitor_id, checked_at, up, status_code, latency_ms, error
		FROM checks
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at < $3
		ORDER BY checked_at DESC
		LIMIT $4`

	rows, err := postgres.GetQuerier(ctx, r.pool).Query(ctx, query, monitorID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query checks: %w", err)
	}
	defer rows.Close()

	checks := make([]*domain.Check, 0)

	for rows.Next() {
		var check domain.Check

		err := rows.Scan(
			&check.ID, &check.MonitorID, &check.CheckedAt,
			&check.Up, &check.StatusCode, &check.LatencyMS, &check.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("scan check: %w", err)
		}

		checks = append(checks, &check)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate checks: %w", err)
	}

	return checks, nil
}

func (r *CheckRepo) Stats(
	ctx context.Context, monitorID string, from, to time.Time,
) (*domain.MonitorStats, error) {
	const query = `
		SELECT
			count(*)                                                        AS total,
			count(*) FILTER (WHERE up)                                      AS successful,
			coalesce(avg(latency_ms), 0)                                    AS avg_latency,
			coalesce(percentile_disc(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) AS p95_latency,
			max(checked_at)                                                 AS last_checked_at
		FROM checks
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at < $3`

	stats := domain.MonitorStats{MonitorID: monitorID, From: from, To: to}

	err := postgres.GetQuerier(ctx, r.pool).QueryRow(ctx, query, monitorID, from, to).Scan(
		&stats.TotalChecks, &stats.SuccessfulChecks,
		&stats.AvgLatencyMS, &stats.P95LatencyMS, &stats.LastCheckedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("select check stats: %w", err)
	}

	if stats.TotalChecks > 0 {
		stats.UptimeRatio = float64(stats.SuccessfulChecks) / float64(stats.TotalChecks)
	}

	return &stats, nil
}

func (r *CheckRepo) Buckets(
	ctx context.Context, monitorID string, from, to time.Time, size domain.BucketSize,
) ([]*domain.Bucket, error) {
	if !size.Valid() {
		return nil, domain.ErrInvalidBucket
	}

	query := fmt.Sprintf(`
		SELECT date_trunc('%s', checked_at) AS bucket, count(*), count(*) FILTER (WHERE up)
		FROM checks
		WHERE monitor_id = $1 AND checked_at >= $2 AND checked_at < $3
		GROUP BY bucket
		ORDER BY bucket`, string(size))

	rows, err := postgres.GetQuerier(ctx, r.pool).Query(ctx, query, monitorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query buckets: %w", err)
	}
	defer rows.Close()

	buckets := make([]*domain.Bucket, 0)

	for rows.Next() {
		bucket := domain.Bucket{MonitorID: monitorID}

		if err := rows.Scan(&bucket.BucketTime, &bucket.Total, &bucket.Successful); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}

		buckets = append(buckets, &bucket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate buckets: %w", err)
	}

	return buckets, nil
}

func (r *CheckRepo) BucketsForMonitors(
	ctx context.Context, monitorIDs []string, from, to time.Time, size domain.BucketSize,
) ([]*domain.Bucket, error) {
	if !size.Valid() {
		return nil, domain.ErrInvalidBucket
	}

	if len(monitorIDs) == 0 {
		return []*domain.Bucket{}, nil
	}

	query := fmt.Sprintf(`
		SELECT monitor_id, date_trunc('%s', checked_at) AS bucket, count(*), count(*) FILTER (WHERE up)
		FROM checks
		WHERE monitor_id = ANY($1) AND checked_at >= $2 AND checked_at < $3
		GROUP BY monitor_id, bucket
		ORDER BY monitor_id, bucket`, string(size))

	rows, err := postgres.GetQuerier(ctx, r.pool).Query(ctx, query, monitorIDs, from, to)
	if err != nil {
		return nil, fmt.Errorf("query buckets for monitors: %w", err)
	}
	defer rows.Close()

	buckets := make([]*domain.Bucket, 0)

	for rows.Next() {
		var bucket domain.Bucket

		if err := rows.Scan(&bucket.MonitorID, &bucket.BucketTime, &bucket.Total, &bucket.Successful); err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}

		buckets = append(buckets, &bucket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate buckets: %w", err)
	}

	return buckets, nil
}

func (r *CheckRepo) DeleteOlderThan(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	const query = `
		DELETE FROM checks
		WHERE id IN (
			SELECT id FROM checks WHERE checked_at < $1 ORDER BY id LIMIT $2
		)`

	tag, err := postgres.GetQuerier(ctx, r.pool).Exec(ctx, query, before, batchSize)
	if err != nil {
		return 0, fmt.Errorf("delete old checks: %w", err)
	}

	return tag.RowsAffected(), nil
}
