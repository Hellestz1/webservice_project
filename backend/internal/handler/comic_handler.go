package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"backend/internal/domain"
	"backend/internal/middleware"
	"backend/internal/usecase"
)

type ComicHandler struct {
	usecase *usecase.ComicUsecase
}

func NewComicHandler(usecase *usecase.ComicUsecase) *ComicHandler {
	return &ComicHandler{usecase: usecase}
}

// ListComics godoc
//
//	@Summary		List comics
//	@Description	Returns a paginated list of comics. Available to all plans.
//	@Tags			Comics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			limit	query		int				false	"Number of results per page (1-100, default 20)"
//	@Param			page	query		int				false	"Page number (default 1)"
//	@Success		200		{object}	map[string]any	"data array of comics"
//	@Failure		400		{object}	ErrorResponse	"invalid_query"
//	@Failure		401		{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		429		{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500		{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/comics [get]
func (h *ComicHandler) ListComics() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 20
		if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
			value, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid_query", "limit must be a number")
				return
			}
			limit = value
		}
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		page := 1
		if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
			value, err := strconv.Atoi(rawPage)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid_query", "page must be a number")
				return
			}
			page = value
		}
		if page <= 0 {
			page = 1
		}

		comics, err := h.usecase.ListComics(limit, page)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{
			"data": comics,
		})
	}
}

// GetComicDetail godoc
//
//	@Summary		Get comic detail
//	@Description	Returns detailed information for a single comic by ID. Available to all plans.
//	@Tags			Comics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string			true	"Comic ID"
//	@Success		200	{object}	map[string]any	"data object of comic"
//	@Failure		401	{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		404	{object}	ErrorResponse	"comic_not_found"
//	@Failure		429	{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500	{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/comics/{id} [get]
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

// ListChapters godoc
//
//	@Summary		List chapters of a comic
//	@Description	Returns all chapters for the given comic ID. Available to all plans.
//	@Tags			Comics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id	path		string			true	"Comic ID"
//	@Success		200	{object}	map[string]any	"data array of chapters"
//	@Failure		401	{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		404	{object}	ErrorResponse	"comic_not_found"
//	@Failure		429	{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500	{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/comics/{id}/chapters [get]
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

// SearchComics godoc
//
//	@Summary		Search comics
//	@Description	Search comics by various filters. Standard plan supports title/genre/author/country. Premium plan additionally supports age_rating, type, sort, order, year, year_from, year_to.
//	@Tags			Comics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			q			query		string			false	"Keyword search by title (alias: title)"
//	@Param			title		query		string			false	"Title keyword (alias of q)"
//	@Param			genre		query		string			false	"Genre/category slug (alias: category)"
//	@Param			category	query		string			false	"Category slug"
//	@Param			author		query		string			false	"Author name keyword"
//	@Param			country		query		string			false	"Country keyword"
//	@Param			age_rating	query		string			false	"Age rating filter (premium only)"
//	@Param			type		query		string			false	"Book type: manga, manhua, manhwa, comic, lightnovel (premium only)"
//	@Param			sort		query		string			false	"Sort field: created_at, publish_year, title (premium only)"
//	@Param			order		query		string			false	"Sort order: asc, desc (premium only)"
//	@Param			year		query		int				false	"Exact publish year (premium only)"
//	@Param			year_from	query		int				false	"Publish year from (premium only)"
//	@Param			year_to		query		int				false	"Publish year to (premium only)"
//	@Param			limit		query		int				false	"Results per page (default 20, max 100)"
//	@Param			page		query		int				false	"Page number (default 1)"
//	@Success		200			{object}	map[string]any	"data array of comics"
//	@Failure		400			{object}	ErrorResponse	"invalid_query"
//	@Failure		401			{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		403			{object}	ErrorResponse	"feature_not_allowed (free plan)"
//	@Failure		429			{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500			{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/comics/search [get]
func (h *ComicHandler) SearchComics() gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := middleware.GetClientProfile(c)
		if !ok {
			writeError(c, http.StatusUnauthorized, "missing_client_profile", "missing client profile")
			return
		}

		filters, err := parseComicSearchFilters(c, profile.Plan)
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

// RecommendComics godoc
//
//	@Summary		Recommend similar comics
//	@Description	Returns comics similar to the given comic ID or title. Requires premium plan.
//	@Tags			Comics
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			id		query		string			false	"Base comic ID to find similar comics"
//	@Param			title	query		string			false	"Base comic title to find similar comics"
//	@Param			limit	query		int				false	"Max results (1-50, default 10)"
//	@Success		200		{object}	map[string]any	"data array of recommended comics"
//	@Failure		400		{object}	ErrorResponse	"invalid_query (id or title required)"
//	@Failure		401		{object}	ErrorResponse	"missing or invalid API key"
//	@Failure		403		{object}	ErrorResponse	"feature_not_allowed (non-premium)"
//	@Failure		404		{object}	ErrorResponse	"comic_not_found"
//	@Failure		429		{object}	ErrorResponse	"rate_limit_exceeded / quota_exceeded"
//	@Failure		500		{object}	ErrorResponse	"internal_error"
//	@Router			/api/v1/comics/recommend [get]
func (h *ComicHandler) RecommendComics() gin.HandlerFunc {
	return func(c *gin.Context) {
		baseID := strings.TrimSpace(c.Query("id"))
		title := strings.TrimSpace(c.Query("title"))
		if baseID == "" && title == "" {
			writeError(c, http.StatusBadRequest, "invalid_query", "id or title is required")
			return
		}

		limit := 10
		if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
			value, err := strconv.Atoi(rawLimit)
			if err != nil {
				writeError(c, http.StatusBadRequest, "invalid_query", "limit must be a number")
				return
			}
			limit = value
		}
		if limit <= 0 {
			limit = 10
		}
		if limit > 50 {
			limit = 50
		}

		comics, err := h.usecase.RecommendComics(baseID, title, limit)
		if err != nil {
			if errors.Is(err, usecase.ErrComicNotFound) {
				writeError(c, http.StatusNotFound, "comic_not_found", "comic not found")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", "unexpected error")
			return
		}

		writeJSON(c, http.StatusOK, map[string]any{
			"data": comics,
		})
	}
}

func parseComicSearchFilters(c *gin.Context, plan string) (domain.ComicSearchFilters, error) {
	filters := domain.ComicSearchFilters{
		Query:     c.Query("q"),
		Category:  c.Query("category"),
		AgeRating: c.Query("age_rating"),
		BookType:  c.Query("type"),
		SortBy:    c.Query("sort"),
		Order:     c.Query("order"),
		Author:    c.Query("author"),
		Country:   c.Query("country"),
	}

	if filters.Query == "" {
		filters.Query = c.Query("title")
	}
	if filters.Category == "" {
		filters.Category = c.Query("genre")
	}

	allowed := map[string]bool{
		"author":   true,
		"category": true,
		"country":  true,
		"genre":    true,
		"limit":    true,
		"page":     true,
		"q":        true,
		"title":    true,
	}
	if plan != "standard" {
		allowed["age_rating"] = true
		allowed["order"] = true
		allowed["sort"] = true
		allowed["type"] = true
		allowed["year"] = true
		allowed["year_from"] = true
		allowed["year_to"] = true
	}
	for key := range c.Request.URL.Query() {
		if !allowed[key] {
			return domain.ComicSearchFilters{}, fmt.Errorf("unsupported query parameter: %s", key)
		}
	}

	if plan != "standard" {
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

		if filters.YearFrom > 0 && filters.YearTo > 0 && filters.YearFrom > filters.YearTo {
			return domain.ComicSearchFilters{}, errors.New("year_from must be <= year_to")
		}
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

	return filters, nil
}
