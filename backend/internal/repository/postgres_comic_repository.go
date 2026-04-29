package repository

import (
	"context"
	"fmt"
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
	COALESCE(c.author, ''),
	COALESCE(c.country, ''),
  COALESCE(c.description, ''),
	c.book_type,
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
		if err := rows.Scan(&comic.ID, &comic.Title, &comic.Author, &comic.Country, &comic.Description, &comic.BookType, &comic.Status, &comic.Genres); err != nil {
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
	COALESCE(c.author, ''),
	COALESCE(c.country, ''),
  COALESCE(c.description, ''),
	c.book_type,
  c.status,
  COALESCE(array_agg(DISTINCT cat.slug) FILTER (WHERE cat.slug IS NOT NULL), '{}') AS genres
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.id = $1 AND c.deleted_at IS NULL
GROUP BY c.id`

	var comic domain.Comic
	err = r.db.QueryRow(ctx, q, comicID).Scan(&comic.ID, &comic.Title, &comic.Author, &comic.Country, &comic.Description, &comic.BookType, &comic.Status, &comic.Genres)
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

func (r *PostgresComicRepository) Search(filters domain.ComicSearchFilters) ([]domain.Comic, error) {
	ctx := context.Background()

	where := make([]string, 0)
	args := make([]any, 0)

	if strings.TrimSpace(filters.Query) != "" {
		args = append(args, strings.TrimSpace(filters.Query))
		where = append(where, fmt.Sprintf("c.title ILIKE '%%' || $%d || '%%'", len(args)))
	}

	if strings.TrimSpace(filters.Author) != "" {
		args = append(args, strings.TrimSpace(filters.Author))
		where = append(where, fmt.Sprintf("c.author ILIKE '%%' || $%d || '%%'", len(args)))
	}

	if strings.TrimSpace(filters.Country) != "" {
		args = append(args, strings.TrimSpace(filters.Country))
		where = append(where, fmt.Sprintf("c.country ILIKE '%%' || $%d || '%%'", len(args)))
	}

	if filters.YearFrom > 0 {
		args = append(args, filters.YearFrom)
		where = append(where, fmt.Sprintf("c.publish_year >= $%d", len(args)))
	}

	if filters.YearTo > 0 {
		args = append(args, filters.YearTo)
		where = append(where, fmt.Sprintf("c.publish_year <= $%d", len(args)))
	}

	if strings.TrimSpace(filters.AgeRating) != "" {
		args = append(args, strings.TrimSpace(filters.AgeRating))
		where = append(where, fmt.Sprintf("c.age_rating = $%d", len(args)))
	}

	if strings.TrimSpace(filters.BookType) != "" {
		args = append(args, strings.TrimSpace(filters.BookType))
		where = append(where, fmt.Sprintf("c.book_type = $%d", len(args)))
	}

	if strings.TrimSpace(filters.Category) != "" {
		args = append(args, strings.TrimSpace(filters.Category))
		where = append(where, fmt.Sprintf(`EXISTS (
  SELECT 1
  FROM comic_categories cc2
  JOIN categories cat2 ON cat2.id = cc2.category_id
  WHERE cc2.comic_id = c.id AND cat2.slug = $%d
)`, len(args)))
	}

	orderBy := map[string]string{
		"created_at":   "c.created_at",
		"publish_year": "c.publish_year",
		"title":        "c.title",
	}

	sortColumn := orderBy[filters.SortBy]
	if sortColumn == "" {
		sortColumn = "c.created_at"
	}

	sortOrder := strings.ToUpper(filters.Order)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	page := filters.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	args = append(args, limit, offset)
	limitArg := len(args) - 1
	offsetArg := len(args)

	query := `
SELECT
  c.id::text,
  c.title,
	COALESCE(c.author, ''),
	COALESCE(c.country, ''),
  COALESCE(c.description, ''),
	c.book_type,
  c.status,
  COALESCE(array_agg(DISTINCT cat.slug) FILTER (WHERE cat.slug IS NOT NULL), '{}') AS genres
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.deleted_at IS NULL`

	if len(where) > 0 {
		query += "\n  AND " + strings.Join(where, "\n  AND ")
	}

	query += fmt.Sprintf(`
GROUP BY c.id
ORDER BY %s %s
LIMIT $%d
OFFSET $%d`, sortColumn, sortOrder, limitArg, offsetArg)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comics := make([]domain.Comic, 0)
	for rows.Next() {
		var comic domain.Comic
		if err := rows.Scan(&comic.ID, &comic.Title, &comic.Author, &comic.Country, &comic.Description, &comic.BookType, &comic.Status, &comic.Genres); err != nil {
			return nil, err
		}
		comics = append(comics, comic)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return comics, nil
}
