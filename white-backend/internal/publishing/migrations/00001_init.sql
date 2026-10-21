-- +goose Up
CREATE SCHEMA IF NOT EXISTS publishing;

CREATE TYPE publishing.post_status AS ENUM ('DRAFT', 'IN_REVIEW', 'APPROVED', 'SCHEDULED', 'PUBLISHING', 'PUBLISHED', 'FAILED', 'REJECTED');
CREATE TYPE publishing.media_kind AS ENUM ('IMAGE', 'VIDEO');
CREATE TYPE publishing.job_status AS ENUM ('PENDING', 'RUNNING', 'SUCCESS', 'FAILED');

-- client_id, uploaded_by_id, created_by_id, approved_by_id and user_id point
-- at identity rows but carry no foreign key: identity owns those tables and
-- cascades arrive as events.
CREATE TABLE publishing.media_assets (
    id             text PRIMARY KEY,
    client_id      text NOT NULL,
    kind           publishing.media_kind NOT NULL,
    url            text NOT NULL,
    thumbnail_url  text,
    filename       text NOT NULL,
    mime_type      text NOT NULL,
    size_bytes     integer NOT NULL DEFAULT 0,
    width          integer,
    height         integer,
    duration_sec   double precision,
    alt_text       text,
    tags           text[] NOT NULL DEFAULT '{}',
    uploaded_by_id text,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX media_assets_client_idx ON publishing.media_assets (client_id, created_at DESC);

CREATE TABLE publishing.content_posts (
    id             text PRIMARY KEY,
    client_id      text NOT NULL,
    title          text NOT NULL DEFAULT '',
    body           text NOT NULL DEFAULT '',
    status         publishing.post_status NOT NULL DEFAULT 'DRAFT',
    scheduled_at   timestamptz,
    published_at   timestamptz,
    link_url       text,
    first_comment  text,
    labels         text[] NOT NULL DEFAULT '{}',
    utm            jsonb,
    review_note    text,
    created_by_id  text,
    approved_by_id text,
    approved_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX content_posts_client_status_idx ON publishing.content_posts (client_id, status, scheduled_at);
CREATE INDEX content_posts_client_scheduled_idx ON publishing.content_posts (client_id, scheduled_at);

-- One destination account per post. `platform` is copied from the analytics
-- social account at save time so lists and the calendar need no cross-service call.
CREATE TABLE publishing.post_targets (
    id                text PRIMARY KEY,
    post_id           text NOT NULL REFERENCES publishing.content_posts(id) ON DELETE CASCADE,
    social_account_id text NOT NULL,
    platform          text NOT NULL DEFAULT 'INSTAGRAM',
    body_override     text,
    status            publishing.post_status NOT NULL DEFAULT 'DRAFT',
    external_id       text,
    permalink         text,
    error             text,
    attempts          integer NOT NULL DEFAULT 0,
    published_at      timestamptz,
    UNIQUE (post_id, social_account_id)
);
CREATE INDEX post_targets_status_idx ON publishing.post_targets (status);

CREATE TABLE publishing.post_media (
    post_id    text NOT NULL REFERENCES publishing.content_posts(id) ON DELETE CASCADE,
    asset_id   text NOT NULL REFERENCES publishing.media_assets(id) ON DELETE CASCADE,
    sort_order integer NOT NULL DEFAULT 0,
    PRIMARY KEY (post_id, asset_id)
);
CREATE INDEX post_media_order_idx ON publishing.post_media (post_id, sort_order);

CREATE TABLE publishing.post_comments (
    id         text PRIMARY KEY,
    post_id    text NOT NULL REFERENCES publishing.content_posts(id) ON DELETE CASCADE,
    user_id    text,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX post_comments_post_idx ON publishing.post_comments (post_id, created_at);

CREATE TABLE publishing.post_activities (
    id         text PRIMARY KEY,
    post_id    text NOT NULL REFERENCES publishing.content_posts(id) ON DELETE CASCADE,
    user_id    text,
    type       text NOT NULL,
    meta       jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX post_activities_post_idx ON publishing.post_activities (post_id, created_at);

CREATE TABLE publishing.sync_jobs (
    id          text PRIMARY KEY,
    client_id   text,
    kind        text NOT NULL,
    status      publishing.job_status NOT NULL DEFAULT 'PENDING',
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    message     text,
    error       text
);
CREATE INDEX sync_jobs_client_idx ON publishing.sync_jobs (client_id, started_at);

-- +goose Down
DROP SCHEMA publishing CASCADE;
