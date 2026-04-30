package service

import (
	"context"
	"database/sql"
	"fmt"

	"backend/internal/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessService struct {
	db *pgxpool.Pool
}

func NewAccessService(db *pgxpool.Pool) *AccessService {
	return &AccessService{db: db}
}

func (s *AccessService) Authenticate(ctx context.Context, apiKey string) (middleware.ClientProfile, error) {
	prefix := apiKey
	if len(prefix) > 16 {
		prefix = prefix[:16]
	}

	const q = `
	SELECT u.id::text, u.id, p.code, k.key_hash, k.id
FROM api_keys k
JOIN users u ON u.id = k.user_id
JOIN user_plans up ON up.user_id = u.id
JOIN plans p ON p.id = up.plan_id
WHERE k.key_prefix = $1
	AND k.is_active = TRUE
	AND (k.expires_at IS NULL OR k.expires_at > NOW())
	AND u.status = 'active'
	AND up.status = 'active'
	AND (up.ends_at IS NULL OR up.ends_at > NOW())`

	rows, err := s.db.Query(ctx, q, prefix)
	if err != nil {
		return middleware.ClientProfile{}, err
	}
	defer rows.Close()

	targetHash := sha256Hex(apiKey)
	for rows.Next() {
		var clientID string
		var userID int64
		var planCode string
		var keyHash string
		var apiKeyID int64
		if err := rows.Scan(&clientID, &userID, &planCode, &keyHash, &apiKeyID); err != nil {
			return middleware.ClientProfile{}, err
		}

		if keyHash == targetHash {
			return middleware.ClientProfile{
				ClientID: clientID,
				Plan:     planCode,
				APIKeyID: apiKeyID,
				UserID:   userID,
			}, nil
		}
	}

	if rows.Err() != nil {
		return middleware.ClientProfile{}, rows.Err()
	}

	return middleware.ClientProfile{}, middleware.ErrAPIKeyNotFound
}

func (s *AccessService) LoadPlanPolicies(ctx context.Context) (map[string]middleware.PlanPolicy, error) {
	const q = `
SELECT p.code, p.requests_per_minute, p.monthly_quota, pf.feature_key
FROM plans p
LEFT JOIN plan_features pf ON pf.plan_id = p.id
ORDER BY p.code`

	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make(map[string]middleware.PlanPolicy)
	for rows.Next() {
		var planCode string
		var rpm int
		var monthlyQuota sql.NullInt64
		var feature sql.NullString
		if err := rows.Scan(&planCode, &rpm, &monthlyQuota, &feature); err != nil {
			return nil, err
		}

		policy, ok := policies[planCode]
		if !ok {
			policy = middleware.PlanPolicy{
				AllowedFeatures:   make(map[string]bool),
				RequestsPerMinute: rpm,
				MonthlyQuota:      int(monthlyQuota.Int64),
			}
		}

		if feature.Valid && feature.String != "" {
			policy.AllowedFeatures[feature.String] = true
		}

		policies[planCode] = policy
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("no plan policies found")
	}

	return policies, nil
}
