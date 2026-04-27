package repository

import (
	"context"
	"strconv"
	"strings"

	"backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresComicRepository struct {
	db *pgxpool.Pool
}

func NewPostgresComicRepository(db *pgxpool.Pool) *PostgresComicRepository {
	return &PostgresComicRepository{db: db}
}

func (r *PostgresComicRepository) List() ([]domain.Comic, error) {
	ctx := context.Background()

	const q = `
SELECT
  c.id::text,
  c.title,
  COALESCE(c.description, ''),
  c.status,
  COALESCE(array_agg(DISTINCT cat.slug) FILTER (WHERE cat.slug IS NOT NULL), '{}') AS genres
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.deleted_at IS NULL
GROUP BY c.id
ORDER BY c.created_at DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comics := make([]domain.Comic, 0)
	for rows.Next() {
		var comic domain.Comic
		if err := rows.Scan(&comic.ID, &comic.Title, &comic.Description, &comic.Status, &comic.Genres); err != nil {
			return nil, err
		}
		comics = append(comics, comic)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return comics, nil
}

func (r *PostgresComicRepository) GetByID(id string) (domain.Comic, bool, error) {
	ctx := context.Background()

	comicID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return domain.Comic{}, false, nil
	}

	const q = `
SELECT
  c.id::text,
  c.title,
  COALESCE(c.description, ''),
  c.status,
  COALESCE(array_agg(DISTINCT cat.slug) FILTER (WHERE cat.slug IS NOT NULL), '{}') AS genres
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.id = $1 AND c.deleted_at IS NULL
GROUP BY c.id`

	var comic domain.Comic
	err = r.db.QueryRow(ctx, q, comicID).Scan(&comic.ID, &comic.Title, &comic.Description, &comic.Status, &comic.Genres)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Comic{}, false, nil
		}
		return domain.Comic{}, false, err
	}

	return comic, true, nil
}

func (r *PostgresComicRepository) ListChapters(comicID string) ([]domain.Chapter, error) {
	ctx := context.Background()

	id, err := strconv.ParseInt(comicID, 10, 64)
	if err != nil {
		return []domain.Chapter{}, nil
	}

	const q = `
SELECT id::text, comic_id::text, chapter_no, title
FROM chapters
WHERE comic_id = $1
ORDER BY chapter_no ASC`

	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chapters := make([]domain.Chapter, 0)
	for rows.Next() {
		var chapter domain.Chapter
		if err := rows.Scan(&chapter.ID, &chapter.ComicID, &chapter.Number, &chapter.Title); err != nil {
			return nil, err
		}
		chapters = append(chapters, chapter)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return chapters, nil
}

func (r *PostgresComicRepository) SearchByTitle(query string) ([]domain.Comic, error) {
	ctx := context.Background()

	query = strings.TrimSpace(query)
	if query == "" {
		return r.List()
	}

	const q = `
SELECT
  c.id::text,
  c.title,
  COALESCE(c.description, ''),
  c.status,
  COALESCE(array_agg(DISTINCT cat.slug) FILTER (WHERE cat.slug IS NOT NULL), '{}') AS genres
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.deleted_at IS NULL
  AND c.title ILIKE '%' || $1 || '%'
GROUP BY c.id
ORDER BY c.created_at DESC`

	rows, err := r.db.Query(ctx, q, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comics := make([]domain.Comic, 0)
	for rows.Next() {
		var comic domain.Comic
		if err := rows.Scan(&comic.ID, &comic.Title, &comic.Description, &comic.Status, &comic.Genres); err != nil {
			return nil, err
		}
		comics = append(comics, comic)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return comics, nil
}
