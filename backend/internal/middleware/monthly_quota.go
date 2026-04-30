package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonthlyQuotaMiddleware struct {
	db       *pgxpool.Pool
	policies map[string]PlanPolicy
}

func NewMonthlyQuotaMiddleware(db *pgxpool.Pool, policies map[string]PlanPolicy) *MonthlyQuotaMiddleware {
	return &MonthlyQuotaMiddleware{db: db, policies: policies}
}

func (m *MonthlyQuotaMiddleware) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := getClientProfile(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"code":    "missing_client_profile",
				"message": "missing client profile",
			})
			return
		}

		policy, ok := m.policies[profile.Plan]
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, map[string]string{
				"code":    "plan_policy_not_found",
				"message": "plan policy not found",
			})
			return
		}

		quota := policy.MonthlyQuota
		if quota > 0 {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			count, err := m.countMonthRequests(ctx, profile.APIKeyID)
			cancel()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
					"code":    "quota_unavailable",
					"message": "quota service unavailable",
				})
				return
			}

			remaining := quota - count
			if remaining < 0 {
				remaining = 0
			}
			resetAt := nextMonthUTC(time.Now().UTC())

			c.Header("X-Quota-Limit", strconv.Itoa(quota))
			c.Header("X-Quota-Remaining", strconv.Itoa(remaining))
			c.Header("X-Quota-Reset", resetAt.Format(time.RFC3339))

			if count >= quota {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, map[string]string{
					"code":    "monthly_quota_exceeded",
					"message": "monthly quota exceeded",
				})
				return
			}
		}

		start := time.Now().UTC()
		c.Next()
		m.logRequest(c, profile.APIKeyID, start)
	}
}

func (m *MonthlyQuotaMiddleware) countMonthRequests(ctx context.Context, apiKeyID int64) (int, error) {
	const q = `
SELECT COUNT(*)
FROM api_request_logs
WHERE api_key_id = $1
  AND requested_at >= date_trunc('month', NOW())
  AND requested_at < date_trunc('month', NOW()) + interval '1 month'`

	var count int
	if err := m.db.QueryRow(ctx, q, apiKeyID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (m *MonthlyQuotaMiddleware) logRequest(c *gin.Context, apiKeyID int64, start time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	queryParams, err := json.Marshal(c.Request.URL.Query())
	if err != nil {
		queryParams = []byte("{}")
	}

	const q = `
INSERT INTO api_request_logs (
  request_id,
  api_key_id,
  path,
  method,
  status_code,
  client_platform,
  client_version,
  accept_language,
  query_params,
  response_ms,
  requested_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`

	_, _ = m.db.Exec(
		ctx,
		q,
		c.GetHeader("X-Request-Id"),
		apiKeyID,
		c.Request.URL.Path,
		c.Request.Method,
		c.Writer.Status(),
		c.GetHeader("X-Client-Platform"),
		c.GetHeader("X-Client-Version"),
		c.GetHeader("Accept-Language"),
		queryParams,
		int(time.Since(start).Milliseconds()),
	)
}

func nextMonthUTC(now time.Time) time.Time {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start.AddDate(0, 1, 0)
}
