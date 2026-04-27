package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	writeJSON(c, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
