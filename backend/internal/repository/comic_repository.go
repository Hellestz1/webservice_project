package repository

import "backend/internal/domain"

type ComicRepository interface {
	List() ([]domain.Comic, error)
	GetByID(id string) (domain.Comic, bool, error)
	GetByTitle(title string) (domain.Comic, bool, error)
	ListChapters(comicID string) ([]domain.Chapter, error)
	Search(filters domain.ComicSearchFilters) ([]domain.Comic, error)
	RecommendByComic(base domain.Comic, limit int) ([]domain.Comic, error)
}
