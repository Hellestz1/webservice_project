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
    author VARCHAR(255),
    country VARCHAR(100),
    description TEXT,
    publish_year SMALLINT,
    age_rating VARCHAR(20),
    book_type VARCHAR(20) NOT NULL DEFAULT 'comic',
    status VARCHAR(20) NOT NULL DEFAULT 'ongoing',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE comics
    ADD COLUMN IF NOT EXISTS author VARCHAR(255);

ALTER TABLE comics
    ADD COLUMN IF NOT EXISTS country VARCHAR(100);

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
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
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

ALTER TABLE api_request_logs
    ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

-- =============================
-- Indexes for filters/search/sort
-- =============================

CREATE INDEX IF NOT EXISTS idx_categories_main_category_id
    ON categories(main_category_id);

CREATE INDEX IF NOT EXISTS idx_comics_publish_year
    ON comics(publish_year);

CREATE INDEX IF NOT EXISTS idx_comics_age_rating
    ON comics(age_rating);

CREATE INDEX IF NOT EXISTS idx_comics_book_type
    ON comics(book_type);

CREATE INDEX IF NOT EXISTS idx_comics_status
    ON comics(status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_created_at
    ON comics(created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_title_trgm
    ON comics USING GIN (title gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_author_trgm
    ON comics USING GIN (author gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comics_country_trgm
    ON comics USING GIN (country gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comic_categories_category_id
    ON comic_categories(category_id);

CREATE INDEX IF NOT EXISTS idx_chapters_comic_id
    ON chapters(comic_id);

CREATE INDEX IF NOT EXISTS idx_api_request_logs_requested_at
    ON api_request_logs(requested_at DESC);

CREATE INDEX IF NOT EXISTS idx_api_request_logs_user_time
    ON api_request_logs(user_id, requested_at DESC);

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
        ('free', 'analytics:usage'),
        ('standard', 'comic:list'),
        ('standard', 'comic:detail'),
        ('standard', 'chapter:list'),
        ('standard', 'comic:search'),
        ('standard', 'analytics:usage'),
        ('premium', 'comic:list'),
        ('premium', 'comic:detail'),
        ('premium', 'chapter:list'),
        ('premium', 'comic:search'),
        ('premium', 'comic:recommend'),
        ('premium', 'analytics:usage'),
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
        ('manga', 'Romance', 'romance'),
        ('manga', 'Sci-Fi', 'sci-fi'),
        ('manga', 'Slice of Life', 'slice-of-life'),
        ('manga', 'Adventure', 'adventure'),
        ('comic', 'Supernatural', 'supernatural'),
        ('comic', 'Mystery', 'mystery'),
        ('comic', 'Horror', 'horror'),
        ('comic', 'Drama', 'drama'),
        ('comic', 'Comedy', 'comedy')
) AS v(main_slug, name, slug) ON v.main_slug = mc.slug
ON CONFLICT (slug) DO NOTHING;

INSERT INTO comics (title, author, country, description, publish_year, age_rating, book_type, status)
VALUES
    ('Skyblade Academy', 'Ava Nolan', 'Japan', 'A rookie swordsman enters the floating academy.', 2022, '13+', 'manga', 'ongoing'),
    ('Midnight Archivist', 'Liam Brooks', 'USA', 'An archivist solves supernatural incidents in old libraries.', 2023, '15+', 'comic', 'ongoing'),
    ('Lantern District', 'Mila Hart', 'UK', 'Detectives track a string of vanishings in a fog city.', 2021, '13+', 'comic', 'ongoing'),
    ('Crimson Harbor', 'Noah Pierce', 'Canada', 'Smugglers clash with a royal navy blockade.', 2019, '15+', 'comic', 'complete'),
    ('Horizon Runner', 'Eva Quinn', 'Australia', 'A courier races across a shattered continent.', 2020, '13+', 'manga', 'ongoing'),
    ('Paper Crane Pact', 'Kai Foster', 'Japan', 'Two rivals forge a pact to save their clans.', 2018, '13+', 'manga', 'complete'),
    ('Iron Lotus', 'Zoe Bishop', 'China', 'A healer discovers a forbidden technique.', 2022, '13+', 'manhua', 'ongoing'),
    ('Glass Orchard', 'Leo Grant', 'France', 'A botanist explores a city made of glass.', 2024, '13+', 'lightnovel', 'ongoing'),
    ('Silent Beacon', 'Iris Bennett', 'USA', 'An island lighthouse signals a hidden war.', 2017, '15+', 'comic', 'complete'),
    ('Ashen Meridian', 'Owen Clarke', 'Germany', 'A cartographer maps a realm of ash storms.', 2016, '13+', 'manhwa', 'complete'),
    ('Golden Warden', 'Nina Cole', 'China', 'A guardian spirit bonds with a runaway prince.', 2020, '13+', 'manhua', 'ongoing'),
    ('Moonlit Circuit', 'Ezra Lane', 'Japan', 'Racers compete in a citywide night league.', 2021, '15+', 'manga', 'ongoing'),
    ('Violet Current', 'Ruby Stone', 'UK', 'A diver hears voices from the deep.', 2023, '15+', 'comic', 'ongoing'),
    ('Cinder Crown', 'Mason Reed', 'Korea', 'A blacksmith crafts a crown that changes fate.', 2019, '13+', 'manhwa', 'complete'),
    ('Stormwright', 'Luna Gray', 'USA', 'An engineer builds ships to ride the storms.', 2018, '13+', 'comic', 'complete'),
    ('Neon Nomad', 'Caleb Frost', 'Canada', 'A drifter hacks the megacity grid.', 2022, '15+', 'manhwa', 'ongoing'),
    ('Jade Mechanic', 'Aria Knox', 'China', 'A mechanic repairs sacred machines.', 2020, '13+', 'manhua', 'ongoing'),
    ('Frostbound Opera', 'Logan Shaw', 'France', 'A singer breaks a winter curse.', 2016, '13+', 'lightnovel', 'complete'),
    ('Copper Atlas', 'Sage Carter', 'Australia', 'Explorers chart floating ruins.', 2017, '13+', 'manga', 'complete'),
    ('River of Kites', 'Isla Park', 'Thailand', 'Festival rivals race across river winds.', 2019, '13+', 'comic', 'complete'),
    ('Echoes of Varn', 'Eli Porter', 'Japan', 'A village keeps a dangerous secret.', 2015, '15+', 'manga', 'complete'),
    ('Aurora Guild', 'Tessa Ward', 'Korea', 'Adventurers chase a skyborn relic.', 2021, '13+', 'manhwa', 'ongoing'),
    ('Clockwork Saffron', 'Nolan Price', 'UK', 'A chef unlocks time-bending recipes.', 2023, '13+', 'lightnovel', 'ongoing'),
    ('Saltglass Dunes', 'Maya Vale', 'China', 'Nomads follow a mirage trail.', 2018, '13+', 'manhua', 'complete'),
    ('Pearl Circuit', 'Dean Holt', 'USA', 'A robotics club fights corporate theft.', 2022, '13+', 'manga', 'ongoing'),
    ('Black Timber', 'Vera West', 'Canada', 'A ranger hunts a forest anomaly.', 2017, '15+', 'comic', 'complete'),
    ('Starling Ferry', 'Jude Hayes', 'Australia', 'A ferry captain navigates star tides.', 2020, '13+', 'lightnovel', 'ongoing'),
    ('Ruinforge', 'Clara Finch', 'Korea', 'Miners awaken a buried forge.', 2016, '13+', 'manhwa', 'complete'),
    ('Bamboo Horizon', 'Ronan Cruz', 'Japan', 'A monk journeys to a distant shrine.', 2021, '13+', 'manga', 'ongoing'),
    ('Cloud Pantry', 'Elena Fox', 'France', 'A baker opens a shop in the sky.', 2024, '13+', 'lightnovel', 'ongoing'),
    ('Marble Chorus', 'Hugo Price', 'UK', 'A choir keeps a city safe.', 2015, '13+', 'comic', 'complete'),
    ('Silver Transit', 'Piper Lane', 'China', 'A conductor rides intercity rails.', 2019, '13+', 'manhua', 'complete'),
    ('Warden of Tides', 'Miles Drake', 'Japan', 'A guardian calms a raging sea.', 2020, '13+', 'manga', 'ongoing'),
    ('Tideglass Academy', 'Nora Bloom', 'Korea', 'Students craft glass ships.', 2023, '13+', 'manhwa', 'ongoing'),
    ('Thorn Signal', 'Aiden Ross', 'USA', 'A scout deciphers a coded distress.', 2018, '15+', 'comic', 'complete'),
    ('Dragonleaf Market', 'Keira Blake', 'China', 'Merchants barter with spirits.', 2022, '13+', 'manhua', 'ongoing'),
    ('Sunken Library', 'Felix Stone', 'UK', 'Divers retrieve lost volumes.', 2017, '13+', 'comic', 'complete'),
    ('Night Orchard', 'Ivy Quinn', 'Japan', 'A keeper guards a forbidden grove.', 2021, '15+', 'manga', 'ongoing'),
    ('Tempest Parade', 'Grant Wells', 'Korea', 'Performers uncover a weather cult.', 2019, '13+', 'manhwa', 'complete'),
    ('Harbor of Threads', 'Sasha Cole', 'France', 'Weavers protect a coastal city.', 2016, '13+', 'lightnovel', 'complete'),
    ('Arcane Loom', 'Wyatt King', 'Japan', 'A tailor stitches living spells.', 2024, '13+', 'manga', 'ongoing'),
    ('Quartz Divide', 'Hazel Young', 'Canada', 'Two nations fight over crystal fields.', 2018, '15+', 'comic', 'complete'),
    ('Winter Couriers', 'Rowan Chase', 'Korea', 'Messengers cross frozen kingdoms.', 2020, '13+', 'manhwa', 'ongoing'),
    ('Cobalt Orchard', 'Mara Scott', 'China', 'Farmers cultivate rare blue fruit.', 2021, '13+', 'manhua', 'ongoing'),
    ('Vale of Mirrors', 'Theo James', 'UK', 'A traveler faces mirrored foes.', 2015, '13+', 'lightnovel', 'complete'),
    ('Craneforge', 'Lucy Hart', 'USA', 'Engineers rebuild an ancient crane.', 2022, '13+', 'comic', 'ongoing'),
    ('Amber Tide', 'Finn Ward', 'Australia', 'A fisher finds a glowing current.', 2017, '13+', 'manga', 'complete'),
    ('Lantern Keep', 'Olive Reed', 'Japan', 'Guardians defend a floating keep.', 2023, '13+', 'manhwa', 'ongoing'),
    ('Verdant Signal', 'Evan Knight', 'France', 'A botanist decodes forest beacons.', 2024, '13+', 'lightnovel', 'ongoing')
ON CONFLICT DO NOTHING;

INSERT INTO comic_categories (comic_id, category_id)
SELECT c.id, cat.id
FROM comics c
JOIN categories cat ON (
    (c.title = 'Skyblade Academy' AND cat.slug IN ('action', 'fantasy')) OR
    (c.title = 'Midnight Archivist' AND cat.slug IN ('supernatural', 'mystery')) OR
    (c.title = 'Lantern District' AND cat.slug IN ('mystery', 'drama')) OR
    (c.title = 'Crimson Harbor' AND cat.slug IN ('action', 'drama')) OR
    (c.title = 'Horizon Runner' AND cat.slug IN ('adventure', 'sci-fi')) OR
    (c.title = 'Paper Crane Pact' AND cat.slug IN ('romance', 'drama')) OR
    (c.title = 'Iron Lotus' AND cat.slug IN ('action', 'fantasy')) OR
    (c.title = 'Glass Orchard' AND cat.slug IN ('slice-of-life', 'fantasy')) OR
    (c.title = 'Silent Beacon' AND cat.slug IN ('mystery', 'horror')) OR
    (c.title = 'Ashen Meridian' AND cat.slug IN ('adventure', 'fantasy')) OR
    (c.title = 'Golden Warden' AND cat.slug IN ('fantasy', 'romance')) OR
    (c.title = 'Moonlit Circuit' AND cat.slug IN ('action', 'sci-fi')) OR
    (c.title = 'Violet Current' AND cat.slug IN ('mystery', 'drama')) OR
    (c.title = 'Cinder Crown' AND cat.slug IN ('fantasy', 'drama')) OR
    (c.title = 'Stormwright' AND cat.slug IN ('adventure', 'sci-fi')) OR
    (c.title = 'Neon Nomad' AND cat.slug IN ('sci-fi', 'action')) OR
    (c.title = 'Jade Mechanic' AND cat.slug IN ('action', 'sci-fi')) OR
    (c.title = 'Frostbound Opera' AND cat.slug IN ('drama', 'romance')) OR
    (c.title = 'Copper Atlas' AND cat.slug IN ('adventure', 'fantasy')) OR
    (c.title = 'River of Kites' AND cat.slug IN ('drama', 'romance')) OR
    (c.title = 'Echoes of Varn' AND cat.slug IN ('horror', 'mystery')) OR
    (c.title = 'Aurora Guild' AND cat.slug IN ('adventure', 'fantasy')) OR
    (c.title = 'Clockwork Saffron' AND cat.slug IN ('comedy', 'slice-of-life')) OR
    (c.title = 'Saltglass Dunes' AND cat.slug IN ('adventure', 'drama')) OR
    (c.title = 'Pearl Circuit' AND cat.slug IN ('sci-fi', 'action')) OR
    (c.title = 'Black Timber' AND cat.slug IN ('horror', 'mystery')) OR
    (c.title = 'Starling Ferry' AND cat.slug IN ('fantasy', 'slice-of-life')) OR
    (c.title = 'Ruinforge' AND cat.slug IN ('action', 'fantasy')) OR
    (c.title = 'Bamboo Horizon' AND cat.slug IN ('adventure', 'slice-of-life')) OR
    (c.title = 'Cloud Pantry' AND cat.slug IN ('comedy', 'slice-of-life')) OR
    (c.title = 'Marble Chorus' AND cat.slug IN ('drama', 'comedy')) OR
    (c.title = 'Silver Transit' AND cat.slug IN ('sci-fi', 'drama')) OR
    (c.title = 'Warden of Tides' AND cat.slug IN ('fantasy', 'action')) OR
    (c.title = 'Tideglass Academy' AND cat.slug IN ('fantasy', 'adventure')) OR
    (c.title = 'Thorn Signal' AND cat.slug IN ('mystery', 'drama')) OR
    (c.title = 'Dragonleaf Market' AND cat.slug IN ('fantasy', 'comedy')) OR
    (c.title = 'Sunken Library' AND cat.slug IN ('mystery', 'horror')) OR
    (c.title = 'Night Orchard' AND cat.slug IN ('romance', 'fantasy')) OR
    (c.title = 'Tempest Parade' AND cat.slug IN ('action', 'drama')) OR
    (c.title = 'Harbor of Threads' AND cat.slug IN ('drama', 'romance')) OR
    (c.title = 'Arcane Loom' AND cat.slug IN ('fantasy', 'romance')) OR
    (c.title = 'Quartz Divide' AND cat.slug IN ('action', 'sci-fi')) OR
    (c.title = 'Winter Couriers' AND cat.slug IN ('adventure', 'drama')) OR
    (c.title = 'Cobalt Orchard' AND cat.slug IN ('slice-of-life', 'comedy')) OR
    (c.title = 'Vale of Mirrors' AND cat.slug IN ('fantasy', 'mystery')) OR
    (c.title = 'Craneforge' AND cat.slug IN ('action', 'comedy')) OR
    (c.title = 'Amber Tide' AND cat.slug IN ('adventure', 'fantasy')) OR
    (c.title = 'Lantern Keep' AND cat.slug IN ('fantasy', 'action')) OR
    (c.title = 'Verdant Signal' AND cat.slug IN ('slice-of-life', 'mystery'))
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
