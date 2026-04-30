package repository

import "backend/internal/domain"

type AnalyticsRepository interface {
	MonthlyUsageCount(userID int64) (int, error)
	MonthlyQuota(planCode string) (int, error)
	TopEndpointUsage(userID int64) (domain.UsageLine, bool, error)
	ListEndpointUsage(userID int64) ([]domain.UsageLine, error)
}
