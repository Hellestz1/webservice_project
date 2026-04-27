package repository

import "backend/internal/domain"

type ComicRepository interface {
	List() ([]domain.Comic, error)
	GetByID(id string) (domain.Comic, bool, error)
	ListChapters(comicID string) ([]domain.Chapter, error)
	SearchByTitle(query string) ([]domain.Comic, error)
}
