package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type requestWindow struct {
	Count      int
	WindowFrom time.Time
}

type RateLimiterMiddleware struct {
	mu       sync.Mutex
	policies map[string]PlanPolicy
	windows  map[string]requestWindow
}

func NewRateLimiterMiddleware(policies map[string]PlanPolicy) *RateLimiterMiddleware {
	return &RateLimiterMiddleware{
		policies: policies,
		windows:  make(map[string]requestWindow),
	}
}

func (m *RateLimiterMiddleware) Require() gin.HandlerFunc {
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

		allowedPerMinute := policy.RequestsPerMinute
		if allowedPerMinute <= 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, map[string]string{
				"code":    "rate_limit_misconfigured",
				"message": "rate limit misconfigured",
			})
			return
		}

		currentWindow := time.Now().UTC().Truncate(time.Minute)
		bucketKey := fmt.Sprintf("%s:%s", profile.ClientID, currentWindow.Format(time.RFC3339))

		m.mu.Lock()
		window := m.windows[bucketKey]
		window.Count++
		window.WindowFrom = currentWindow
		m.windows[bucketKey] = window
		count := window.Count
		m.mu.Unlock()

		remaining := allowedPerMinute - count
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(allowedPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", currentWindow.Add(time.Minute).Format(time.RFC3339))

		if count > allowedPerMinute {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, map[string]string{
				"code":    "rate_limit_exceeded",
				"message": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
