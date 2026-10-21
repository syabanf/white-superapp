-- +goose Up
CREATE SCHEMA IF NOT EXISTS analytics;

CREATE TYPE analytics.social_platform AS ENUM ('INSTAGRAM', 'FACEBOOK', 'TIKTOK');
CREATE TYPE analytics.seo_dimension AS ENUM ('QUERY', 'PAGE', 'COUNTRY', 'DEVICE');
CREATE TYPE analytics.audit_strategy AS ENUM ('MOBILE', 'DESKTOP');
CREATE TYPE analytics.issue_severity AS ENUM ('ERROR', 'WARNING', 'NOTICE');
CREATE TYPE analytics.job_status AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED');
CREATE TYPE analytics.audit_cadence AS ENUM ('NONE', 'WEEKLY', 'DAILY');

-- ─── Social ──────────────────────────────────────────────────

CREATE TABLE analytics.social_accounts (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    platform      analytics.social_platform NOT NULL,
    external_id   text NOT NULL,
    username      text NOT NULL,
    display_name  text NOT NULL,
    avatar_url    text,
    biography     text,
    is_competitor boolean NOT NULL DEFAULT false,
    connection_id text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, platform, external_id)
);
CREATE INDEX social_accounts_client_platform_idx ON analytics.social_accounts (client_id, platform);

CREATE TABLE analytics.social_snapshots (
    id                text PRIMARY KEY,
    social_account_id text NOT NULL REFERENCES analytics.social_accounts(id) ON DELETE CASCADE,
    date              date NOT NULL,
    followers         integer NOT NULL,
    following         integer,
    media_count       integer NOT NULL DEFAULT 0,
    reach             integer,
    impressions       integer,
    profile_views     integer,
    website_clicks    integer,
    UNIQUE (social_account_id, date)
);

CREATE TABLE analytics.social_posts (
    id                text PRIMARY KEY,
    social_account_id text NOT NULL REFERENCES analytics.social_accounts(id) ON DELETE CASCADE,
    external_id       text NOT NULL,
    caption           text NOT NULL DEFAULT '',
    media_type        text NOT NULL DEFAULT 'IMAGE',
    product_type      text,
    permalink         text,
    media_url         text,
    thumbnail_url     text,
    published_at      timestamptz NOT NULL,
    likes             integer NOT NULL DEFAULT 0,
    comments          integer NOT NULL DEFAULT 0,
    shares            integer NOT NULL DEFAULT 0,
    saves             integer NOT NULL DEFAULT 0,
    views             integer NOT NULL DEFAULT 0,
    reach             integer NOT NULL DEFAULT 0,
    impressions       integer NOT NULL DEFAULT 0,
    synced_at         timestamptz NOT NULL DEFAULT now(),
    UNIQUE (social_account_id, external_id)
);
CREATE INDEX social_posts_account_published_idx ON analytics.social_posts (social_account_id, published_at);

-- ─── SEO ─────────────────────────────────────────────────────

CREATE TABLE analytics.seo_properties (
    id              text PRIMARY KEY,
    client_id       text NOT NULL,
    site_url        text NOT NULL,
    ga4_property_id text,
    connection_id   text,
    location_code   integer NOT NULL DEFAULT 2360,
    language_code   text NOT NULL DEFAULT 'id',
    audit_cadence   analytics.audit_cadence NOT NULL DEFAULT 'NONE',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, site_url)
);

CREATE TABLE analytics.seo_daily_metrics (
    id          text PRIMARY KEY,
    property_id text NOT NULL REFERENCES analytics.seo_properties(id) ON DELETE CASCADE,
    date        date NOT NULL,
    clicks      integer NOT NULL,
    impressions integer NOT NULL,
    ctr         double precision NOT NULL,
    position    double precision NOT NULL,
    UNIQUE (property_id, date)
);

CREATE TABLE analytics.seo_dimension_metrics (
    id          text PRIMARY KEY,
    property_id text NOT NULL REFERENCES analytics.seo_properties(id) ON DELETE CASCADE,
    date        date NOT NULL,
    dimension   analytics.seo_dimension NOT NULL,
    key         text NOT NULL,
    clicks      integer NOT NULL,
    impressions integer NOT NULL,
    ctr         double precision NOT NULL,
    position    double precision NOT NULL,
    UNIQUE (property_id, date, dimension, key)
);
CREATE INDEX seo_dimension_metrics_dim_date_idx ON analytics.seo_dimension_metrics (property_id, dimension, date);

CREATE TABLE analytics.ga4_daily_metrics (
    id                 text PRIMARY KEY,
    property_id        text NOT NULL REFERENCES analytics.seo_properties(id) ON DELETE CASCADE,
    date               date NOT NULL,
    sessions           integer NOT NULL,
    organic_sessions   integer NOT NULL,
    users              integer NOT NULL,
    engaged_sessions   integer NOT NULL,
    conversions        integer NOT NULL,
    avg_engagement_sec double precision NOT NULL,
    UNIQUE (property_id, date)
);

CREATE TABLE analytics.seo_audits (
    id             text PRIMARY KEY,
    property_id    text NOT NULL REFERENCES analytics.seo_properties(id) ON DELETE CASCADE,
    url            text NOT NULL,
    strategy       analytics.audit_strategy NOT NULL,
    run_at         timestamptz NOT NULL DEFAULT now(),
    performance    integer NOT NULL,
    seo            integer NOT NULL,
    accessibility  integer NOT NULL,
    best_practices integer NOT NULL,
    lcp_ms         double precision,
    inp_ms         double precision,
    cls            double precision,
    fcp_ms         double precision,
    ttfb_ms        double precision,
    tbt_ms         double precision,
    speed_index_ms double precision,
    raw            jsonb
);
CREATE INDEX seo_audits_property_run_idx ON analytics.seo_audits (property_id, run_at);

CREATE TABLE analytics.seo_crawls (
    id            text PRIMARY KEY,
    property_id   text NOT NULL REFERENCES analytics.seo_properties(id) ON DELETE CASCADE,
    started_at    timestamptz NOT NULL DEFAULT now(),
    finished_at   timestamptz,
    status        analytics.job_status NOT NULL DEFAULT 'PENDING',
    pages_crawled integer NOT NULL DEFAULT 0,
    max_pages     integer NOT NULL DEFAULT 50,
    error         text
);
CREATE INDEX seo_crawls_property_started_idx ON analytics.seo_crawls (property_id, started_at);

CREATE TABLE analytics.seo_crawl_pages (
    id                 text PRIMARY KEY,
    crawl_id           text NOT NULL REFERENCES analytics.seo_crawls(id) ON DELETE CASCADE,
    url                text NOT NULL,
    status_code        integer NOT NULL,
    title              text,
    meta_description   text,
    h1_count           integer NOT NULL DEFAULT 0,
    word_count         integer NOT NULL DEFAULT 0,
    canonical          text,
    indexable          boolean NOT NULL DEFAULT true,
    images_total       integer NOT NULL DEFAULT 0,
    images_missing_alt integer NOT NULL DEFAULT 0,
    internal_links     integer NOT NULL DEFAULT 0,
    external_links     integer NOT NULL DEFAULT 0,
    load_ms            integer,
    has_json_ld        boolean NOT NULL DEFAULT false,
    has_hreflang       boolean NOT NULL DEFAULT false
);
CREATE INDEX seo_crawl_pages_crawl_idx ON analytics.seo_crawl_pages (crawl_id);

CREATE TABLE analytics.seo_issues (
    id       text PRIMARY KEY,
    crawl_id text NOT NULL REFERENCES analytics.seo_crawls(id) ON DELETE CASCADE,
    severity analytics.issue_severity NOT NULL,
    code     text NOT NULL,
    message  text NOT NULL,
    url      text NOT NULL
);
CREATE INDEX seo_issues_crawl_severity_idx ON analytics.seo_issues (crawl_id, severity);

-- ─── Meta Ads ────────────────────────────────────────────────

CREATE TABLE analytics.ad_accounts (
    id             text PRIMARY KEY,
    client_id      text NOT NULL,
    external_id    text NOT NULL,
    name           text NOT NULL,
    currency       text NOT NULL DEFAULT 'IDR',
    connection_id  text,
    last_synced_at timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, external_id)
);

CREATE TABLE analytics.ad_campaigns (
    id            text PRIMARY KEY,
    ad_account_id text NOT NULL REFERENCES analytics.ad_accounts(id) ON DELETE CASCADE,
    external_id   text NOT NULL,
    name          text NOT NULL,
    objective     text NOT NULL DEFAULT 'OUTCOME_TRAFFIC',
    status        text NOT NULL DEFAULT 'ACTIVE',
    daily_budget  double precision,
    start_time    timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (ad_account_id, external_id)
);

CREATE TABLE analytics.ad_sets (
    id           text PRIMARY KEY,
    campaign_id  text NOT NULL REFERENCES analytics.ad_campaigns(id) ON DELETE CASCADE,
    external_id  text NOT NULL,
    name         text NOT NULL,
    status       text NOT NULL DEFAULT 'ACTIVE',
    daily_budget double precision,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (campaign_id, external_id)
);

CREATE TABLE analytics.ads (
    id            text PRIMARY KEY,
    ad_set_id     text NOT NULL REFERENCES analytics.ad_sets(id) ON DELETE CASCADE,
    external_id   text NOT NULL,
    name          text NOT NULL,
    status        text NOT NULL DEFAULT 'ACTIVE',
    thumbnail_url text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (ad_set_id, external_id)
);

-- One row per (account, date, level). The level is implied by which ids are
-- set. The unique index coalesces the nullable ids so an upsert can target it.
CREATE TABLE analytics.ad_daily_insights (
    id             text PRIMARY KEY,
    ad_account_id  text NOT NULL REFERENCES analytics.ad_accounts(id) ON DELETE CASCADE,
    date           date NOT NULL,
    campaign_id    text REFERENCES analytics.ad_campaigns(id) ON DELETE CASCADE,
    ad_set_id      text REFERENCES analytics.ad_sets(id) ON DELETE CASCADE,
    ad_id          text REFERENCES analytics.ads(id) ON DELETE CASCADE,
    spend          double precision NOT NULL DEFAULT 0,
    impressions    integer NOT NULL DEFAULT 0,
    reach          integer NOT NULL DEFAULT 0,
    clicks         integer NOT NULL DEFAULT 0,
    link_clicks    integer NOT NULL DEFAULT 0,
    results        integer NOT NULL DEFAULT 0,
    result_type    text,
    purchase_value double precision
);
CREATE UNIQUE INDEX ad_daily_insights_key_idx ON analytics.ad_daily_insights
    (ad_account_id, date, COALESCE(campaign_id, ''), COALESCE(ad_set_id, ''), COALESCE(ad_id, ''));
CREATE INDEX ad_daily_insights_campaign_date_idx ON analytics.ad_daily_insights (campaign_id, date);
CREATE INDEX ad_daily_insights_ad_date_idx ON analytics.ad_daily_insights (ad_id, date);

CREATE TABLE analytics.ad_demographics (
    id            text PRIMARY KEY,
    ad_account_id text NOT NULL REFERENCES analytics.ad_accounts(id) ON DELETE CASCADE,
    date_from     date NOT NULL,
    date_to       date NOT NULL,
    campaign_id   text,
    age           text NOT NULL,
    gender        text NOT NULL,
    spend         double precision NOT NULL DEFAULT 0,
    impressions   integer NOT NULL DEFAULT 0,
    reach         integer NOT NULL DEFAULT 0,
    results       integer NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX ad_demographics_key_idx ON analytics.ad_demographics
    (ad_account_id, date_from, date_to, COALESCE(campaign_id, ''), age, gender);

-- ─── Sync jobs owned by this service ─────────────────────────

CREATE TABLE analytics.sync_jobs (
    id          text PRIMARY KEY,
    client_id   text NOT NULL,
    kind        text NOT NULL,
    status      analytics.job_status NOT NULL DEFAULT 'PENDING',
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    message     text,
    error       text
);
CREATE INDEX sync_jobs_client_started_idx ON analytics.sync_jobs (client_id, started_at);

-- +goose Down
DROP SCHEMA analytics CASCADE;
