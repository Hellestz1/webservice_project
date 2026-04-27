package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ClientProfile struct {
	ClientID string
	Plan     string
}

type APIKeyAuthMiddleware struct {
	authenticator APIKeyAuthenticator
}

var ErrAPIKeyNotFound = errors.New("api key not found")

type APIKeyAuthenticator interface {
	Authenticate(ctx context.Context, apiKey string) (ClientProfile, error)
}

func NewAPIKeyAuthMiddleware(authenticator APIKeyAuthenticator) *APIKeyAuthMiddleware {
	return &APIKeyAuthMiddleware{authenticator: authenticator}
}

func (m *APIKeyAuthMiddleware) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"code":    "missing_api_key",
				"message": "missing API key",
			})
			return
		}

		profile, err := m.authenticator.Authenticate(c.Request.Context(), apiKey)
		if err != nil {
			if errors.Is(err, ErrAPIKeyNotFound) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
					"code":    "invalid_api_key",
					"message": "invalid API key",
				})
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, map[string]string{
				"code":    "auth_unavailable",
				"message": "authentication service unavailable",
			})
			return
		}

		setClientProfile(c, profile)
		c.Next()
	}
}
