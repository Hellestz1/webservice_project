package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PlanPolicy struct {
	AllowedFeatures   map[string]bool
	RequestsPerMinute int
	MonthlyQuota      int
}

type FeatureGateMiddleware struct {
	policies map[string]PlanPolicy
}

func NewFeatureGateMiddleware(policies map[string]PlanPolicy) *FeatureGateMiddleware {
	return &FeatureGateMiddleware{policies: policies}
}

func (m *FeatureGateMiddleware) Require(feature string) gin.HandlerFunc {
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

		if !policy.AllowedFeatures[feature] {
			c.AbortWithStatusJSON(http.StatusForbidden, map[string]string{
				"code":    "feature_not_available",
				"message": "feature not available in your plan",
			})
			return
		}

		c.Next()
	}
}
