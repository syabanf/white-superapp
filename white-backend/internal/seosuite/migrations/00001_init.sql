-- +goose Up
CREATE SCHEMA IF NOT EXISTS seosuite;

CREATE TYPE seosuite.device_type AS ENUM ('DESKTOP', 'MOBILE');
CREATE TYPE seosuite.keyword_intent AS ENUM ('INFORMATIONAL', 'NAVIGATIONAL', 'COMMERCIAL', 'TRANSACTIONAL');
CREATE TYPE seosuite.job_status AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED');

-- property_id refers to analytics.seo_properties; client_id is copied onto
-- every table so a client purge never needs another service.
CREATE TABLE seosuite.tracked_keywords (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    property_id   text NOT NULL,
    keyword       text NOT NULL,
    location_code integer NOT NULL DEFAULT 2360,
    language_code text NOT NULL DEFAULT 'id',
    device        seosuite.device_type NOT NULL DEFAULT 'MOBILE',
    tags          text[] NOT NULL DEFAULT '{}',
    target_url    text,
    volume        integer,
    difficulty    integer,
    cpc           double precision,
    intent        seosuite.keyword_intent,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (property_id, keyword, location_code, device)
);
CREATE INDEX tracked_keywords_property_idx ON seosuite.tracked_keywords (property_id, created_at);
CREATE INDEX tracked_keywords_client_idx ON seosuite.tracked_keywords (client_id);

-- Daily SERP position; position NULL = not in the top 100.
CREATE TABLE seosuite.rank_snapshots (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    keyword_id    text NOT NULL REFERENCES seosuite.tracked_keywords(id) ON DELETE CASCADE,
    date          date NOT NULL,
    position      integer,
    url           text,
    serp_features text[] NOT NULL DEFAULT '{}',
    top_results   jsonb,
    UNIQUE (keyword_id, date)
);
CREATE INDEX rank_snapshots_date_idx ON seosuite.rank_snapshots (date);
CREATE INDEX rank_snapshots_client_idx ON seosuite.rank_snapshots (client_id);

-- Cached research run. seed is "{mode}:{seed}", results is KeywordIdea[].
CREATE TABLE seosuite.keyword_research (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    seed          text NOT NULL,
    location_code integer NOT NULL DEFAULT 2360,
    language_code text NOT NULL DEFAULT 'id',
    provider      text NOT NULL,
    results       jsonb NOT NULL,
    fetched_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX keyword_research_key_idx ON seosuite.keyword_research (client_id, seed, location_code, language_code, fetched_at DESC);
CREATE INDEX keyword_research_fetched_idx ON seosuite.keyword_research (fetched_at);

CREATE TABLE seosuite.backlinks (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    property_id   text NOT NULL,
    source_url    text NOT NULL,
    source_domain text NOT NULL,
    target_url    text NOT NULL,
    anchor        text NOT NULL DEFAULT '',
    dofollow      boolean NOT NULL DEFAULT true,
    domain_rank   integer,
    spam_score    integer,
    first_seen    date NOT NULL,
    last_seen     date NOT NULL,
    is_lost       boolean NOT NULL DEFAULT false,
    UNIQUE (property_id, source_url, target_url)
);
CREATE INDEX backlinks_property_lost_idx ON seosuite.backlinks (property_id, is_lost, last_seen);
CREATE INDEX backlinks_client_idx ON seosuite.backlinks (client_id);

CREATE TABLE seosuite.backlink_snapshots (
    id                text PRIMARY KEY,
    client_id         text NOT NULL,
    property_id       text NOT NULL,
    date              date NOT NULL,
    backlinks         integer NOT NULL,
    referring_domains integer NOT NULL,
    dofollow          integer NOT NULL DEFAULT 0,
    nofollow          integer NOT NULL DEFAULT 0,
    new_backlinks     integer NOT NULL DEFAULT 0,
    lost_backlinks    integer NOT NULL DEFAULT 0,
    domain_rank       integer,
    toxic_share       double precision,
    UNIQUE (property_id, date)
);
CREATE INDEX backlink_snapshots_date_idx ON seosuite.backlink_snapshots (date);
CREATE INDEX backlink_snapshots_client_idx ON seosuite.backlink_snapshots (client_id);

CREATE TABLE seosuite.competitor_domains (
    id          text PRIMARY KEY,
    client_id   text NOT NULL,
    property_id text NOT NULL,
    domain      text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (property_id, domain)
);
CREATE INDEX competitor_domains_client_idx ON seosuite.competitor_domains (client_id);

-- One row per domain (own or competitor) per day.
CREATE TABLE seosuite.domain_snapshots (
    id                text PRIMARY KEY,
    client_id         text NOT NULL,
    property_id       text NOT NULL,
    domain            text NOT NULL,
    date              date NOT NULL,
    organic_keywords  integer NOT NULL,
    organic_traffic   integer NOT NULL,
    organic_cost      double precision,
    top3              integer NOT NULL DEFAULT 0,
    top10             integer NOT NULL DEFAULT 0,
    top100            integer NOT NULL DEFAULT 0,
    backlinks         integer,
    referring_domains integer,
    domain_rank       integer,
    UNIQUE (property_id, domain, date)
);
CREATE INDEX domain_snapshots_date_idx ON seosuite.domain_snapshots (date);
CREATE INDEX domain_snapshots_client_idx ON seosuite.domain_snapshots (client_id);

-- Kinds: SEO_SUITE_RANKS, SEO_SUITE_BACKLINKS, SEO_SUITE_DOMAINS, SEO_SUITE_DAILY.
CREATE TABLE seosuite.sync_jobs (
    id          text PRIMARY KEY,
    client_id   text NOT NULL,
    kind        text NOT NULL,
    status      seosuite.job_status NOT NULL DEFAULT 'PENDING',
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    message     text,
    error       text
);
CREATE INDEX sync_jobs_client_kind_idx ON seosuite.sync_jobs (client_id, kind, started_at DESC);
CREATE INDEX sync_jobs_started_idx ON seosuite.sync_jobs (started_at);

-- +goose Down
DROP SCHEMA seosuite CASCADE;
