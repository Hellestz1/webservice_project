# Database Design (PostgreSQL)

This folder contains DB schema for the comic provider API.

The backend now reads comic data, API keys, and package policies from PostgreSQL.

## Files

- `001_init_postgres.sql`: initial schema + indexes + default package plan seeds

It also seeds:

- demo plans and plan features
- demo comics/categories/chapters
- demo API keys (`free-demo-key`, `standard-demo-key`, `premium-demo-key`) as SHA-256 hashes

## API Query -> DB Column Mapping

For endpoint: `GET /api/v1/catoon-books` (recommended rename to `/api/v1/cartoon-books` or keep existing `/api/v1/comics`)

- `q` -> `comics.title` (trigram index)
- `author` -> `comics.author` (trigram index)
- `country` -> `comics.country` (trigram index)
- `category_id` -> `comic_categories.category_id`
- `main_category_id` -> `categories.main_category_id`
- `min_year` / `max_year` -> `comics.publish_year`
- `rate` (age rating) -> `comics.age_rating`
- `min_price` / `max_price` -> `comics.price`
- `sort` + `order` -> whitelist: `price`, `publish_year`, `created_at`
- `page` + `limit` -> OFFSET/LIMIT pagination

## Header Usage

These headers are useful for tracking and can be saved into `api_request_logs`:

- `X-Request-Id` -> `request_id`
- `X-Client-Platform` -> `client_platform`
- `X-Client-Version` -> `client_version`
- `Accept-Language` -> `accept_language`

Auth flow:

- `Authorization` is optional for dashboard/JWT use cases.
- API provider calls should mainly use `X-API-Key`.

## Sample Search SQL

```sql
SELECT c.*
FROM comics c
LEFT JOIN comic_categories cc ON cc.comic_id = c.id
LEFT JOIN categories cat ON cat.id = cc.category_id
WHERE c.deleted_at IS NULL
  AND ($1::text IS NULL OR c.title ILIKE '%' || $1 || '%')
  AND ($2::bigint IS NULL OR cc.category_id = $2)
  AND ($3::bigint IS NULL OR cat.main_category_id = $3)
  AND ($4::smallint IS NULL OR c.publish_year >= $4)
  AND ($5::smallint IS NULL OR c.publish_year <= $5)
  AND ($6::text IS NULL OR c.age_rating = $6)
  AND ($7::numeric IS NULL OR c.price >= $7)
  AND ($8::numeric IS NULL OR c.price <= $8);
```

Sort order should be whitelisted in application code to avoid SQL injection.
