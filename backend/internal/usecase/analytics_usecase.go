package usecase

import (
	"backend/internal/domain"
	"backend/internal/repository"
)

type AnalyticsUsecase struct {
	repo repository.AnalyticsRepository
}

func NewAnalyticsUsecase(repo repository.AnalyticsRepository) *AnalyticsUsecase {
	return &AnalyticsUsecase{repo: repo}
}

func (u *AnalyticsUsecase) RemainingQuota(planCode string, userID int64) (int, error) {
	quota, err := u.repo.MonthlyQuota(planCode)
	if err != nil {
		return 0, err
	}
	if quota <= 0 {
		return 0, nil
	}

	count, err := u.repo.MonthlyUsageCount(userID)
	if err != nil {
		return 0, err
	}
	remaining := quota - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (u *AnalyticsUsecase) TopEndpoint(userID int64) (domain.UsageLine, bool, error) {
	return u.repo.TopEndpointUsage(userID)
}

func (u *AnalyticsUsecase) EndpointUsage(userID int64) ([]domain.UsageLine, error) {
	return u.repo.ListEndpointUsage(userID)
}
