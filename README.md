# webservice_project

Comic web service provider for package-based API access.

## Project Goals

- Provide comic metadata APIs for consumers.
- Offer tiered package plans: `free`, `standard`, `premium`.
- Enforce access by API key, feature gating, and rate limiting.

## Current Backend Structure

```text
backend/
	cmd/
		main.go
	internal/
		service/
			access_service.go
		storage/
			postgres.go
		domain/
			comic.go
		handler/
			comic_handler.go
			health_handler.go
			http.go
		middleware/
			apikey.go
			context.go
			feature_gate.go
			rate_limit.go
		repository/
			comic_repository.go
			postgres_comic_repository.go
		usecase/
			comic_usecase.go
```

## Package Plans (v1)

- `free`
	- Quota: `1,000 requests/month`
	- Rate limit: `10 requests/minute`
	- Features: comic list, detail, chapter list
- `standard`
	- Quota: `100,000 requests/month`
	- Rate limit: `120 requests/minute`
	- Features: free + search
- `premium`
	- Quota: unlimited (with rate limiting)
	- Rate limit: `1,000 requests/minute`
	- Features: standard + recommendation + analytics (stubs for next step)

## Plan Capabilities and Test Commands

Plan differences:

- `free`
	- Allowed: list comics, comic detail, chapter list
	- Not allowed: search
- `standard`
	- Allowed: free + search
- `premium`
	- Allowed: standard + recommendation and analytics (not implemented yet)

Test with curl:

```bash
# free: list ok, search blocked
curl -H "X-API-Key: free-demo-key" http://localhost:8080/api/v1/comics
curl -H "X-API-Key: free-demo-key" "http://localhost:8080/api/v1/comics/search?q=sky"

# standard: search allowed
curl -H "X-API-Key: standard-demo-key" "http://localhost:8080/api/v1/comics/search?q=sky"

# premium: same search allowed (premium-only endpoints are not implemented yet)
curl -H "X-API-Key: premium-demo-key" "http://localhost:8080/api/v1/comics/search?q=sky"
```

## Run Backend

1. Prepare PostgreSQL and create database `comic_provider`.
2. Run schema:

```bash
psql "postgres://postgres:postgres@localhost:5432/comic_provider?sslmode=disable" -f database/001_init_postgres.sql
```

3. Start API:

```bash
cd backend
DB_DSN="postgres://postgres:postgres@localhost:5432/comic_provider?sslmode=disable" go run ./cmd
```

## Run With Docker

```bash
cd backend
docker build -t comic-provider-api .
docker run --rm -p 8080:8080 \
	-e DB_DSN="postgres://postgres:postgres@host.docker.internal:5432/comic_provider?sslmode=disable" \
	comic-provider-api
```

For Linux Docker Engine, if `host.docker.internal` is not available, use your host IP or run with:

```bash
docker run --rm -p 8080:8080 --add-host=host.docker.internal:host-gateway \
	-e DB_DSN="postgres://postgres:postgres@host.docker.internal:5432/comic_provider?sslmode=disable" \
	comic-provider-api
```

Server starts at `http://localhost:8080`.

## Demo API Keys (seeded in DB)

- `free-demo-key`
- `standard-demo-key`
- `premium-demo-key`

## Demo Users (seeded in DB)

- `free@demo.local` / `demo1234` -> free
- `standard@demo.local` / `demo1234` -> standard
- `premium@demo.local` / `demo1234` -> premium

Send in header:

```text
X-API-Key: <your-key>
```

## Available Endpoints

- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/api-key`
- `GET /api/v1/comics`
- `GET /api/v1/comics/{id}`
- `GET /api/v1/comics/{id}/chapters`
- `GET /api/v1/comics/search` (standard/premium)

## Quick Test

```bash
curl -H "X-API-Key: free-demo-key" http://localhost:8080/api/v1/comics
curl -H "X-API-Key: free-demo-key" http://localhost:8080/api/v1/comics/search?q=sky
```

Register (3 plans):

```bash
curl -X POST http://localhost:8080/auth/register \
	-H "Content-Type: application/json" \
	-d '{"email":"new@demo.local","password":"demo1234","plan":"free"}'

curl -X POST http://localhost:8080/auth/register \
	-H "Content-Type: application/json" \
	-d '{"email":"new-standard@demo.local","password":"demo1234","plan":"standard"}'

curl -X POST http://localhost:8080/auth/register \
	-H "Content-Type: application/json" \
	-d '{"email":"new-premium@demo.local","password":"demo1234","plan":"premium"}'
```

Login (3 plans):

```bash
curl -X POST http://localhost:8080/auth/login \
	-H "Content-Type: application/json" \
	-d '{"email":"free@demo.local","password":"demo1234"}'

curl -X POST http://localhost:8080/auth/login \
	-H "Content-Type: application/json" \
	-d '{"email":"standard@demo.local","password":"demo1234"}'

curl -X POST http://localhost:8080/auth/login \
	-H "Content-Type: application/json" \
	-d '{"email":"premium@demo.local","password":"demo1234"}'
```

Issue API key (3 plans):

```bash
curl -X POST http://localhost:8080/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{"email":"free@demo.local","password":"demo1234"}'

curl -X POST http://localhost:8080/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{"email":"standard@demo.local","password":"demo1234"}'

curl -X POST http://localhost:8080/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{"email":"premium@demo.local","password":"demo1234"}'
```

Expected behavior:

- First command returns data.
- Second command returns `403` because search is not in free plan.

Search query parameters (optional):

- `q` keyword (title)
- `title` title keyword (alias of `q`)
- `genre` category slug (alias of `category`)
- `author` author name keyword
- `country` country keyword
- `year` exact publish year
- `year_from` / `year_to` publish year range
- `category` category slug
- `age_rating` age rating
- `type` book type (e.g. manga, manhua, manhwa, comic, lightnovel)
- `sort` one of `created_at`, `publish_year`, `title`
- `order` `asc` or `desc`
- `limit` page size (default 20, max 100)
- `page` page index (default 1)

Plan search scope:

- `standard`: only `title`, `genre`, `author`, `country` plus pagination
- `premium`: all parameters

## Next Suggested Tasks

- Persist rate-limit counters in Redis for multi-instance deployment.
- Add recommendation and analytics endpoints for premium.
- Add tests for middleware and handlers.

## Database Bootstrap

- PostgreSQL schema is prepared at `database/001_init_postgres.sql`.
- Query/filter mapping notes are in `database/README.md`.