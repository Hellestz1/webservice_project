package repository

import (
	"context"
	"database/sql"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresAnalyticsRepository(db *pgxpool.Pool) *PostgresAnalyticsRepository {
	return &PostgresAnalyticsRepository{db: db}
}

func (r *PostgresAnalyticsRepository) MonthlyUsageCount(userID int64) (int, error) {
	ctx := context.Background()

	const q = `
SELECT COUNT(*)
FROM api_request_logs l
LEFT JOIN api_keys k ON k.id = l.api_key_id
JOIN user_plans up ON up.user_id = COALESCE(l.user_id, k.user_id)
WHERE COALESCE(l.user_id, k.user_id) = $1
  AND up.status = 'active'
  AND (up.ends_at IS NULL OR up.ends_at > NOW())
  AND l.requested_at >= GREATEST(up.started_at, date_trunc('month', NOW()))
  AND l.requested_at < date_trunc('month', NOW()) + interval '1 month'`

	var count int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresAnalyticsRepository) MonthlyQuota(planCode string) (int, error) {
	ctx := context.Background()

	const q = `
SELECT monthly_quota
FROM plans
WHERE code = $1`

	var quota sql.NullInt64
	if err := r.db.QueryRow(ctx, q, planCode).Scan(&quota); err != nil {
		return 0, err
	}
	if !quota.Valid {
		return 0, nil
	}
	return int(quota.Int64), nil
}

func (r *PostgresAnalyticsRepository) TopEndpointUsage(userID int64) (domain.UsageLine, bool, error) {
	ctx := context.Background()

	const q = `
SELECT path, COUNT(*) AS count
FROM api_request_logs l
LEFT JOIN api_keys k ON k.id = l.api_key_id
JOIN user_plans up ON up.user_id = COALESCE(l.user_id, k.user_id)
WHERE COALESCE(l.user_id, k.user_id) = $1
	AND up.status = 'active'
	AND (up.ends_at IS NULL OR up.ends_at > NOW())
	AND l.requested_at >= GREATEST(up.started_at, date_trunc('month', NOW()))
	AND l.requested_at < date_trunc('month', NOW()) + interval '1 month'
GROUP BY path
ORDER BY count DESC, path ASC
LIMIT 1`

	var line domain.UsageLine
	err := r.db.QueryRow(ctx, q, userID).Scan(&line.Path, &line.Count)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.UsageLine{}, false, nil
		}
		return domain.UsageLine{}, false, err
	}
	return line, true, nil
}

func (r *PostgresAnalyticsRepository) ListEndpointUsage(userID int64) ([]domain.UsageLine, error) {
	ctx := context.Background()

	const q = `
SELECT path, COUNT(*) AS count
FROM api_request_logs l
LEFT JOIN api_keys k ON k.id = l.api_key_id
JOIN user_plans up ON up.user_id = COALESCE(l.user_id, k.user_id)
WHERE COALESCE(l.user_id, k.user_id) = $1
	AND up.status = 'active'
	AND (up.ends_at IS NULL OR up.ends_at > NOW())
	AND l.requested_at >= GREATEST(up.started_at, date_trunc('month', NOW()))
	AND l.requested_at < date_trunc('month', NOW()) + interval '1 month'
GROUP BY path
ORDER BY count DESC, path ASC`

	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := make([]domain.UsageLine, 0)
	for rows.Next() {
		var line domain.UsageLine
		if err := rows.Scan(&line.Path, &line.Count); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return lines, nil
}
