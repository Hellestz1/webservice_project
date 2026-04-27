package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/usecase"
)

type ComicHandler struct {
	usecase *usecase.ComicUsecase
}

func NewComicHandler(usecase *usecase.ComicUsecase) *ComicHandler {
	return &ComicHandler{usecase: usecase}
}

func (h *ComicHandler) ListComics() gin.HandlerFunc {
	return func(c *gin.Context) {
		comics, err := h.usecase.ListComics()
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{
			"data": comics,
		})
	}
}

func (h *ComicHandler) GetComicDetail() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		comic, err := h.usecase.GetComicDetail(id)
		if err != nil {
			if errors.Is(err, usecase.ErrComicNotFound) {
				writeError(c, http.StatusNotFound, "comic_not_found", "comic not found")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{"data": comic})
	}
}

func (h *ComicHandler) ListChapters() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		chapters, err := h.usecase.ListChapters(id)
		if err != nil {
			if errors.Is(err, usecase.ErrComicNotFound) {
				writeError(c, http.StatusNotFound, "comic_not_found", "comic not found")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{"data": chapters})
	}
}

func (h *ComicHandler) SearchComics() gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		comics, err := h.usecase.SearchComics(query)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{
			"data": comics,
		})
	}
}
