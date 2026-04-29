-- Initial PostgreSQL schema for comic provider API
-- Target: search/filter/sort/pagination + API package/usage tracking

BEGIN;

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =============================
-- Catalog
-- =============================

CREATE TABLE IF NOT EXISTS main_categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    main_category_id BIGINT REFERENCES main_categories(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(120) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS comics (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    title_normalized VARCHAR(255) GENERATED ALWAYS AS (lower(title)) STORED,
    description TEXT,
    publish_year SMALLINT,
    age_rating VARCHAR(20),
    price NUMERIC(10,2) NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'THB',
    status VARCHAR(20) NOT NULL DEFAULT 'ongoing',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS comic_categories (
    comic_id BIGINT NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (comic_id, category_id)
);

CREATE TABLE IF NOT EXISTS chapters (
    id BIGSERIAL PRIMARY KEY,
    comic_id BIGINT NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
    chapter_no INT NOT NULL,
    title VARCHAR(255) NOT NULL,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (comic_id, chapter_no)
);

-- =============================
-- API package and access
-- =============================

CREATE TABLE IF NOT EXISTS plans (
    id SMALLSERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    monthly_quota INT,
    requests_per_minute INT NOT NULL,
    sla_percent NUMERIC(4,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plan_features (
    plan_id SMALLINT NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    feature_key VARCHAR(100) NOT NULL,
    PRIMARY KEY (plan_id, feature_key)
);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_plans (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id SMALLINT NOT NULL REFERENCES plans(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (key_prefix)
);

CREATE TABLE IF NOT EXISTS api_request_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64),
    api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    path VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INT NOT NULL,
    client_platform VARCHAR(30),
    client_version VARCHAR(30),
    accept_language VARCHAR(50),
    query_params JSONB,
    response_ms INT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================
-- Indexes for filters/search/sort
-- =============================

CREATE INDEX IF NOT EXISTS idx_categories_main_category_id
    ON categories(main_category_id);

CREATE INDEX IF NOT EXISTS idx_comics_publish_year
    ON comics(publish_year);

CREATE INDEX IF NOT EXISTS idx_comics_age_rating
    ON comics(age_rating);

CREATE INDEX IF NOT EXISTS idx_comics_price
    ON comics(price);

CREATE INDEX IF NOT EXISTS idx_comics_status
    ON comics(status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_created_at
    ON comics(created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_title_trgm
    ON comics USING GIN (title gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comic_categories_category_id
    ON comic_categories(category_id);

CREATE INDEX IF NOT EXISTS idx_chapters_comic_id
    ON chapters(comic_id);

CREATE INDEX IF NOT EXISTS idx_api_request_logs_requested_at
    ON api_request_logs(requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_request_logs_api_key_time
    ON api_request_logs(api_key_id, requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_request_logs_request_id
    ON api_request_logs(request_id)
    WHERE request_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_user_plans_user_status
    ON user_plans(user_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_active_plan
    ON user_plans(user_id)
    WHERE status = 'active';

-- =============================
-- Seed plans
-- =============================

INSERT INTO plans (code, name, monthly_quota, requests_per_minute, sla_percent)
VALUES
    ('free', 'Free', 1000, 10, NULL),
    ('standard', 'Standard', 100000, 120, 99.50),
    ('premium', 'Premium', NULL, 1000, 99.90)
ON CONFLICT (code) DO NOTHING;

INSERT INTO plan_features (plan_id, feature_key)
SELECT p.id, f.feature_key
FROM plans p
JOIN (
    VALUES
        ('free', 'comic:list'),
        ('free', 'comic:detail'),
        ('free', 'chapter:list'),
        ('standard', 'comic:list'),
        ('standard', 'comic:detail'),
        ('standard', 'chapter:list'),
        ('standard', 'comic:search'),
        ('premium', 'comic:list'),
        ('premium', 'comic:detail'),
        ('premium', 'chapter:list'),
        ('premium', 'comic:search'),
        ('premium', 'comic:recommend'),
        ('premium', 'analytics:realtime')
) AS f(plan_code, feature_key) ON f.plan_code = p.code
ON CONFLICT (plan_id, feature_key) DO NOTHING;

INSERT INTO main_categories (name, slug)
VALUES
    ('Manga', 'manga'),
    ('Comic', 'comic')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO categories (main_category_id, name, slug)
SELECT mc.id, v.name, v.slug
FROM main_categories mc
JOIN (
    VALUES
        ('manga', 'Action', 'action'),
        ('manga', 'Fantasy', 'fantasy'),
        ('comic', 'Supernatural', 'supernatural')
) AS v(main_slug, name, slug) ON v.main_slug = mc.slug
ON CONFLICT (slug) DO NOTHING;

INSERT INTO comics (title, description, publish_year, age_rating, price, status)
VALUES
    ('Skyblade Academy', 'A rookie swordsman enters the floating academy.', 2022, '13+', 99.00, 'ongoing'),
    ('Midnight Archivist', 'An archivist solves supernatural incidents in old libraries.', 2023, '15+', 149.00, 'ongoing')
ON CONFLICT DO NOTHING;

INSERT INTO comic_categories (comic_id, category_id)
SELECT c.id, cat.id
FROM comics c
JOIN categories cat ON (
    (c.title = 'Skyblade Academy' AND cat.slug IN ('action', 'fantasy')) OR
    (c.title = 'Midnight Archivist' AND cat.slug IN ('supernatural'))
)
ON CONFLICT (comic_id, category_id) DO NOTHING;

INSERT INTO chapters (comic_id, chapter_no, title, published_at)
SELECT c.id, v.chapter_no, v.title, NOW()
FROM comics c
JOIN (
    VALUES
        ('Skyblade Academy', 1, 'The Entrance Trial'),
        ('Skyblade Academy', 2, 'Sky Duel'),
        ('Midnight Archivist', 1, 'The Locked Wing')
) AS v(comic_title, chapter_no, title) ON v.comic_title = c.title
ON CONFLICT (comic_id, chapter_no) DO NOTHING;

INSERT INTO users (email, password_hash, status)
VALUES
    ('free@demo.local', crypt('demo1234', gen_salt('bf')), 'active'),
    ('standard@demo.local', crypt('demo1234', gen_salt('bf')), 'active'),
    ('premium@demo.local', crypt('demo1234', gen_salt('bf')), 'active')
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_plans (user_id, plan_id, status, started_at)
SELECT u.id, p.id, 'active', NOW()
FROM users u
JOIN (
    VALUES
        ('free@demo.local', 'free'),
        ('standard@demo.local', 'standard'),
        ('premium@demo.local', 'premium')
) AS v(email, plan_code) ON v.email = u.email
JOIN plans p ON p.code = v.plan_code
ON CONFLICT DO NOTHING;

INSERT INTO api_keys (user_id, key_prefix, key_hash, is_active)
SELECT
    u.id,
    k.key_prefix,
    encode(digest(k.raw_key, 'sha256'), 'hex'),
    TRUE
FROM users u
JOIN (
    VALUES
        ('free@demo.local', 'free-demo-key', 'free-demo-key'),
        ('standard@demo.local', 'standard-demo-key', 'standard-demo-key'),
        ('premium@demo.local', 'premium-demo-key', 'premium-demo-key')
) AS k(email, key_prefix, raw_key) ON k.email = u.email
ON CONFLICT (key_prefix) DO NOTHING;

COMMIT;
