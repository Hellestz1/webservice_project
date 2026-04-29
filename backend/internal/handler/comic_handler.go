package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
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
		filters, err := parseComicSearchFilters(c)
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}

		comics, err := h.usecase.SearchComics(filters)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{
			"data": comics,
		})
	}
}

func parseComicSearchFilters(c *gin.Context) (domain.ComicSearchFilters, error) {
	filters := domain.ComicSearchFilters{
		Query:     c.Query("q"),
		Category:  c.Query("category"),
		AgeRating: c.Query("age_rating"),
		BookType:  c.Query("type"),
		SortBy:    c.Query("sort"),
		Order:     c.Query("order"),
	}

	if year := c.Query("year"); year != "" {
		value, err := strconv.Atoi(year)
		if err != nil {
			return domain.ComicSearchFilters{}, err
		}
		filters.YearFrom = value
		filters.YearTo = value
	}

	if yearFrom := c.Query("year_from"); yearFrom != "" {
		value, err := strconv.Atoi(yearFrom)
		if err != nil {
			return domain.ComicSearchFilters{}, err
		}
		filters.YearFrom = value
	}

	if yearTo := c.Query("year_to"); yearTo != "" {
		value, err := strconv.Atoi(yearTo)
		if err != nil {
			return domain.ComicSearchFilters{}, err
		}
		filters.YearTo = value
	}

	if limit := c.Query("limit"); limit != "" {
		value, err := strconv.Atoi(limit)
		if err != nil {
			return domain.ComicSearchFilters{}, err
		}
		filters.Limit = value
	}

	if page := c.Query("page"); page != "" {
		value, err := strconv.Atoi(page)
		if err != nil {
			return domain.ComicSearchFilters{}, err
		}
		filters.Page = value
	}

	if filters.YearFrom > 0 && filters.YearTo > 0 && filters.YearFrom > filters.YearTo {
		return domain.ComicSearchFilters{}, errors.New("year_from must be <= year_to")
	}

	return filters, nil
}
