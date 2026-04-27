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
	- Quota: `1,000 requests/month` (quota middleware not added yet)
	- Rate limit: `10 requests/minute`
	- Features: comic list, detail, chapter list
- `standard`
	- Quota: `100,000 requests/month` (quota middleware not added yet)
	- Rate limit: `120 requests/minute`
	- Features: free + search
- `premium`
	- Quota: unlimited (with rate limiting)
	- Rate limit: `1,000 requests/minute`
	- Features: standard + recommendation + analytics (stubs for next step)

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

Send in header:

```text
X-API-Key: <your-key>
```

## Available Endpoints

- `GET /health`
- `GET /api/v1/comics`
- `GET /api/v1/comics/{id}`
- `GET /api/v1/comics/{id}/chapters`
- `GET /api/v1/comics/search?q=keyword` (standard/premium)

## Quick Test

```bash
curl -H "X-API-Key: free-demo-key" http://localhost:8080/api/v1/comics
curl -H "X-API-Key: free-demo-key" http://localhost:8080/api/v1/comics/search?q=sky
```

Expected behavior:

- First command returns data.
- Second command returns `403` because search is not in free plan.

## Next Suggested Tasks

- Add monthly quota middleware and usage table.
- Persist rate-limit counters in Redis for multi-instance deployment.
- Add recommendation and analytics endpoints for premium.
- Add tests for middleware and handlers.

## Database Bootstrap

- PostgreSQL schema is prepared at `database/001_init_postgres.sql`.
- Query/filter mapping notes are in `database/README.md`.