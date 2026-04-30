package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health godoc
//
//	@Summary		Health check
//	@Description	Returns service health status
//	@Tags			System
//	@Produce		json
//	@Success		200	{object}	map[string]string	"ok"
//	@Router			/health [get]
func Health(c *gin.Context) {
	writeJSON(c, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
