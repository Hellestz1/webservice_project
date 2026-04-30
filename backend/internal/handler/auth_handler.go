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

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Plan     string `json:"plan"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type apiKeyRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePlanRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Plan     string `json:"plan"`
}

type authResponse struct {
	APIKey string `json:"api_key"`
	Plan   string `json:"plan"`
}

type loginResponse struct {
	Plan string `json:"plan"`
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

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
