package handler

import (
	"net/http"
	"strings"

	"backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *service.AuthService
}

// registerRequest defines the request body for user registration.
type registerRequest struct {
	Email    string `json:"email"    example:"user@example.com"`
	Password string `json:"password" example:"secret1234"`
	Plan     string `json:"plan"     example:"free" enums:"free,standard,premium"`
}

// loginRequest defines the request body for login.
type loginRequest struct {
	Email    string `json:"email"    example:"user@example.com"`
	Password string `json:"password" example:"secret1234"`
}

// apiKeyRequest defines the request body for issuing an API key.
type apiKeyRequest struct {
	Email    string `json:"email"    example:"user@example.com"`
	Password string `json:"password" example:"secret1234"`
}

// changePlanRequest defines the request body for changing a subscription plan.
type changePlanRequest struct {
	Email    string `json:"email"    example:"user@example.com"`
	Password string `json:"password" example:"secret1234"`
	Plan     string `json:"plan"     example:"standard" enums:"free,standard,premium"`
}

// authResponse is returned after successful registration or API key issuance.
type authResponse struct {
	APIKey string `json:"api_key" example:"sk-abc123"`
	Plan   string `json:"plan"    example:"free"`
}

// loginResponse is returned after a successful login.
type loginResponse struct {
	Plan string `json:"plan" example:"free"`
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register godoc
//
//	@Summary		Register a new user
//	@Description	Create a new user account with a chosen subscription plan. Returns an API key on success.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		registerRequest	true	"Registration payload"
//	@Success		201		{object}	authResponse
//	@Failure		400		{object}	ErrorResponse	"invalid_request / invalid_input / plan_not_found"
//	@Failure		409		{object}	ErrorResponse	"email_exists"
//	@Failure		500		{object}	ErrorResponse	"register_failed"
//	@Router			/auth/register [post]
func (h *AuthHandler) Register() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req registerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		req.Plan = strings.TrimSpace(req.Plan)
		result, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.Plan)
		if err != nil {
			switch err {
			case service.ErrEmailExists:
				writeError(c, http.StatusConflict, "email_exists", "email already registered")
			case service.ErrPlanNotFound:
				writeError(c, http.StatusBadRequest, "plan_not_found", "plan not found")
			case service.ErrInvalidInput:
				writeError(c, http.StatusBadRequest, "invalid_input", "invalid email or password")
			default:
				writeError(c, http.StatusInternalServerError, "register_failed", "register failed")
			}
			return
		}

		writeJSON(c, http.StatusCreated, authResponse{APIKey: result.APIKey, Plan: result.Plan})
	}
}

// Login godoc
//
//	@Summary		Login
//	@Description	Authenticate with email and password. Returns the user's active plan.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		loginRequest	true	"Login payload"
//	@Success		200		{object}	loginResponse
//	@Failure		400		{object}	ErrorResponse	"invalid_request / invalid_input"
//	@Failure		401		{object}	ErrorResponse	"invalid_credentials"
//	@Failure		403		{object}	ErrorResponse	"user_inactive / plan_not_found"
//	@Failure		500		{object}	ErrorResponse	"login_failed"
//	@Router			/auth/login [post]
func (h *AuthHandler) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		result, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			switch err {
			case service.ErrInvalidCredentials:
				writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			case service.ErrUserInactive:
				writeError(c, http.StatusForbidden, "user_inactive", "user is inactive")
			case service.ErrPlanNotFound:
				writeError(c, http.StatusForbidden, "plan_not_found", "plan not found")
			case service.ErrInvalidInput:
				writeError(c, http.StatusBadRequest, "invalid_input", "invalid email or password")
			default:
				writeError(c, http.StatusInternalServerError, "login_failed", "login failed")
			}
			return
		}

		writeJSON(c, http.StatusOK, loginResponse{Plan: result.Plan})
	}
}

// IssueAPIKey godoc
//
//	@Summary		Issue API key
//	@Description	Generate a new API key for an authenticated user. The key is used to access /api/v1/* endpoints via the X-API-Key header.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		apiKeyRequest	true	"Credentials"
//	@Success		200		{object}	authResponse
//	@Failure		400		{object}	ErrorResponse	"invalid_request / invalid_input"
//	@Failure		401		{object}	ErrorResponse	"invalid_credentials"
//	@Failure		403		{object}	ErrorResponse	"user_inactive / plan_not_found"
//	@Failure		500		{object}	ErrorResponse	"api_key_failed"
//	@Router			/auth/api-key [post]
func (h *AuthHandler) IssueAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req apiKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		result, err := h.auth.IssueAPIKey(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			switch err {
			case service.ErrInvalidCredentials:
				writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			case service.ErrUserInactive:
				writeError(c, http.StatusForbidden, "user_inactive", "user is inactive")
			case service.ErrPlanNotFound:
				writeError(c, http.StatusForbidden, "plan_not_found", "plan not found")
			case service.ErrInvalidInput:
				writeError(c, http.StatusBadRequest, "invalid_input", "invalid email or password")
			default:
				writeError(c, http.StatusInternalServerError, "api_key_failed", "api key generation failed")
			}
			return
		}

		writeJSON(c, http.StatusOK, authResponse{APIKey: result.APIKey, Plan: result.Plan})
	}
}

// ChangePlan godoc
//
//	@Summary		Change subscription plan
//	@Description	Upgrade or downgrade the subscription plan for an authenticated user.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		changePlanRequest	true	"Change plan payload"
//	@Success		200		{object}	authResponse
//	@Failure		400		{object}	ErrorResponse	"invalid_request / invalid_input / plan_not_found"
//	@Failure		401		{object}	ErrorResponse	"invalid_credentials"
//	@Failure		403		{object}	ErrorResponse	"user_inactive"
//	@Failure		500		{object}	ErrorResponse	"change_plan_failed"
//	@Router			/auth/plan [post]
func (h *AuthHandler) ChangePlan() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req changePlanRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}

		result, err := h.auth.ChangePlan(c.Request.Context(), req.Email, req.Password, req.Plan)
		if err != nil {
			switch err {
			case service.ErrInvalidCredentials:
				writeError(c, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			case service.ErrUserInactive:
				writeError(c, http.StatusForbidden, "user_inactive", "user is inactive")
			case service.ErrPlanNotFound:
				writeError(c, http.StatusBadRequest, "plan_not_found", "plan not found")
			case service.ErrInvalidInput:
				writeError(c, http.StatusBadRequest, "invalid_input", "invalid email or password")
			default:
				writeError(c, http.StatusInternalServerError, "change_plan_failed", "change plan failed")
			}
			return
		}

		writeJSON(c, http.StatusOK, authResponse{Plan: result.Plan})
	}
}
