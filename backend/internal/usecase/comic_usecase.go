package usecase

import (
	"errors"

	"backend/internal/domain"
	"backend/internal/repository"
)

var ErrComicNotFound = errors.New("comic not found")

type ComicUsecase struct {
	repo repository.ComicRepository
}

func NewComicUsecase(repo repository.ComicRepository) *ComicUsecase {
	return &ComicUsecase{repo: repo}
}

func (u *ComicUsecase) ListComics() ([]domain.Comic, error) {
	return u.repo.List()
}

func (u *ComicUsecase) GetComicDetail(id string) (domain.Comic, error) {
	comic, ok, err := u.repo.GetByID(id)
	if err != nil {
		return domain.Comic{}, err
	}

	if !ok {
		return domain.Comic{}, ErrComicNotFound
	}
	return comic, nil
}

func (u *ComicUsecase) ListChapters(comicID string) ([]domain.Chapter, error) {
	_, ok, err := u.repo.GetByID(comicID)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, ErrComicNotFound
	}
	return u.repo.ListChapters(comicID)
}

func (u *ComicUsecase) SearchComics(filters domain.ComicSearchFilters) ([]domain.Comic, error) {
	return u.repo.Search(filters)
}

func (u *ComicUsecase) RecommendComics(baseID string, title string, limit int) ([]domain.Comic, error) {
	var (
		base domain.Comic
		ok   bool
		err  error
	)

	if baseID != "" {
		base, ok, err = u.repo.GetByID(baseID)
	} else {
		base, ok, err = u.repo.GetByTitle(title)
	}
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrComicNotFound
	}

	return u.repo.RecommendByComic(base, limit)
}
