package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/middleware"
	"backend/internal/usecase"
)

type AnalyticsHandler struct {
	usecase *usecase.AnalyticsUsecase
}

func NewAnalyticsHandler(usecase *usecase.AnalyticsUsecase) *AnalyticsHandler {
	return &AnalyticsHandler{usecase: usecase}
}

// Usage godoc
//
//	@Summary		API usage analytics
//	@Description	Returns quota and endpoint usage stats for the authenticated user.
//	@Description	- **free**: remaining monthly quota only
//	@Description	- **standard**: remaining quota + top endpoint
//	@Description	- **premium**: remaining quota + full endpoint usage breakdown
//	@Tags			Analytics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	map[string]any	"usage data (fields vary by plan)"
//	@Failure		401	{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		403	{object}	ErrorResponse	"feature_not_allowed (free plan)"
//	@Failure		429	{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500	{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/analytics/usage [get]
func (h *AnalyticsHandler) Usage() gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := middleware.GetClientProfile(c)
		if !ok {
			writeError(c, http.StatusUnauthorized, "missing_client_profile", "missing client profile")
			return
		}

		switch profile.Plan {
		case "free":
			remaining, err := h.usecase.RemainingQuota(profile.Plan, profile.UserID)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
				return
			}
			if remaining > 0 {
				remaining--
			}

			writeJSON(c, http.StatusOK, map[string]any{
				"remaining": remaining,
			})
		case "standard":
			remaining, err := h.usecase.RemainingQuota(profile.Plan, profile.UserID)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
				return
			}
			if remaining > 0 {
				remaining--
			}

			line, ok, err := h.usecase.TopEndpoint(profile.UserID)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
				return
			}

			if !ok {
				writeJSON(c, http.StatusOK, map[string]any{
					"remaining": remaining,
					"top":       nil,
				})
				return
			}

			writeJSON(c, http.StatusOK, map[string]any{
				"remaining": remaining,
				"top":       line,
			})
		default:
			remaining, err := h.usecase.RemainingQuota(profile.Plan, profile.UserID)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
				return
			}
			if remaining > 0 {
				remaining--
			}

			lines, err := h.usecase.EndpointUsage(profile.UserID)
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
				return
			}

			writeJSON(c, http.StatusOK, map[string]any{
				"remaining": remaining,
				"lines":     lines,
			})
		}
	}
}
