-- +goose Up
CREATE SCHEMA IF NOT EXISTS content;

CREATE TYPE content.plan_status AS ENUM ('DRAFT', 'IN_REVIEW', 'APPROVED');
CREATE TYPE content.platform AS ENUM ('INSTAGRAM', 'FACEBOOK', 'TIKTOK');
CREATE TYPE content.content_format AS ENUM ('FEED', 'CAROUSEL', 'REEL', 'STORY');
CREATE TYPE content.item_status AS ENUM ('IDEA', 'COPYWRITING', 'DESIGN', 'REVIEW', 'REVISION', 'APPROVED', 'SCHEDULED', 'PUBLISHED', 'FAILED');
CREATE TYPE content.design_status AS ENUM ('TODO', 'IN_PROGRESS', 'IN_REVIEW', 'CHANGES_REQUESTED', 'APPROVED');
CREATE TYPE content.asset_kind AS ENUM ('LOGO', 'BRAND_GUIDELINE', 'FONT', 'PHOTO', 'TEMPLATE', 'OTHER');

-- client_id and every *_by / *_id pointing at a user are identity ids
-- without foreign keys: identity owns those rows and cascades arrive as
-- events (white.client.deleted, white.user.deleted).

-- Editorial plan (EP): one per client per month.
CREATE TABLE content.plans (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    month         date NOT NULL,
    title         text NOT NULL DEFAULT '',
    status        content.plan_status NOT NULL DEFAULT 'DRAFT',
    goals         text NOT NULL DEFAULT '',
    pillars       text[] NOT NULL DEFAULT '{}',
    tone          text NOT NULL DEFAULT '',
    notes         text NOT NULL DEFAULT '',
    -- [{"platform":"INSTAGRAM","perWeek":3}, ...]
    frequency     jsonb NOT NULL DEFAULT '[]',
    ai_model      text,
    generated_at  timestamptz,
    submitted_by  text,
    submitted_at  timestamptz,
    approved_by   text,
    approved_at   timestamptz,
    review_note   text,
    created_by    text,
    updated_by    text,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    CHECK (extract(day FROM month) = 1)
);
CREATE UNIQUE INDEX plans_client_month_key ON content.plans (client_id, month) WHERE deleted_at IS NULL;

CREATE TABLE content.plan_comments (
    id         text PRIMARY KEY,
    plan_id    text NOT NULL REFERENCES content.plans(id) ON DELETE CASCADE,
    user_id    text,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX plan_comments_plan_idx ON content.plan_comments (plan_id, created_at);

CREATE TABLE content.items (
    id                 text PRIMARY KEY,
    plan_id            text NOT NULL REFERENCES content.plans(id) ON DELETE CASCADE,
    client_id          text NOT NULL,
    planned_date       date NOT NULL,
    publish_at         timestamptz,
    platform           content.platform NOT NULL,
    format             content.content_format NOT NULL DEFAULT 'FEED',
    pillar             text NOT NULL DEFAULT '',
    title              text NOT NULL DEFAULT '',
    hook               text NOT NULL DEFAULT '',
    caption            text NOT NULL DEFAULT '',
    hashtags           text[] NOT NULL DEFAULT '{}',
    visual_brief       text NOT NULL DEFAULT '',
    copywriter_id      text,
    designer_id        text,
    copy_due           date,
    design_due         date,
    review_due         date,
    status             content.item_status NOT NULL DEFAULT 'IDEA',
    revision_round     integer NOT NULL DEFAULT 0,
    caption_version    integer NOT NULL DEFAULT 0,
    copy_done_at       timestamptz,
    design_done_at     timestamptz,
    approved_at        timestamptz,
    approved_by        text,
    scheduled_by       text,
    publishing_post_id text,
    published_at       timestamptz,
    permalink          text,
    external_ids       text[] NOT NULL DEFAULT '{}',
    publish_error      text,
    created_by         text,
    updated_by         text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    deleted_at         timestamptz
);
CREATE INDEX items_plan_idx ON content.items (plan_id, planned_date) WHERE deleted_at IS NULL;
CREATE INDEX items_client_status_idx ON content.items (client_id, status, planned_date) WHERE deleted_at IS NULL;
CREATE INDEX items_scheduled_idx ON content.items (publish_at) WHERE status = 'SCHEDULED' AND deleted_at IS NULL;

-- Every caption change is a new version; items.caption mirrors the latest.
CREATE TABLE content.caption_revisions (
    id         text PRIMARY KEY,
    item_id    text NOT NULL REFERENCES content.items(id) ON DELETE CASCADE,
    version    integer NOT NULL,
    caption    text NOT NULL,
    hook       text NOT NULL DEFAULT '',
    hashtags   text[] NOT NULL DEFAULT '{}',
    note       text NOT NULL DEFAULT '',
    created_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (item_id, version)
);

-- kind: COMMENT, CHANGES_REQUESTED, APPROVED. round is the item's
-- revision round when the comment was written.
CREATE TABLE content.item_comments (
    id         text PRIMARY KEY,
    item_id    text NOT NULL REFERENCES content.items(id) ON DELETE CASCADE,
    user_id    text,
    body       text NOT NULL,
    kind       text NOT NULL DEFAULT 'COMMENT',
    round      integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX item_comments_item_idx ON content.item_comments (item_id, created_at);

-- Audit trail: who moved what, when.
CREATE TABLE content.item_activities (
    id          text PRIMARY KEY,
    item_id     text NOT NULL REFERENCES content.items(id) ON DELETE CASCADE,
    user_id     text,
    type        text NOT NULL,
    from_status text,
    to_status   text,
    note        text,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX item_activities_item_idx ON content.item_activities (item_id, created_at);

CREATE TABLE content.design_tasks (
    id              text PRIMARY KEY,
    item_id         text NOT NULL REFERENCES content.items(id) ON DELETE CASCADE,
    client_id       text NOT NULL,
    title           text NOT NULL DEFAULT '',
    brief           text NOT NULL DEFAULT '',
    refs            text[] NOT NULL DEFAULT '{}',
    size_preset     text NOT NULL DEFAULT '',
    width           integer NOT NULL DEFAULT 0,
    height          integer NOT NULL DEFAULT 0,
    designer_id     text,
    due_date        date,
    status          content.design_status NOT NULL DEFAULT 'TODO',
    current_version integer NOT NULL DEFAULT 0,
    created_by      text,
    updated_by      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE INDEX design_tasks_item_idx ON content.design_tasks (item_id) WHERE deleted_at IS NULL;
CREATE INDEX design_tasks_client_idx ON content.design_tasks (client_id, status) WHERE deleted_at IS NULL;

-- One version may hold several files (carousel slides), ordered by sort_order.
CREATE TABLE content.design_files (
    id          text PRIMARY KEY,
    task_id     text NOT NULL REFERENCES content.design_tasks(id) ON DELETE CASCADE,
    version     integer NOT NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    file_key    text NOT NULL,
    file_name   text NOT NULL,
    mime_type   text NOT NULL,
    size_bytes  bigint NOT NULL DEFAULT 0,
    uploaded_by text,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX design_files_task_idx ON content.design_files (task_id, version, sort_order);

-- decision: COMMENT, CHANGES_REQUESTED, APPROVED.
CREATE TABLE content.design_feedback (
    id         text PRIMARY KEY,
    task_id    text NOT NULL REFERENCES content.design_tasks(id) ON DELETE CASCADE,
    file_id    text,
    version    integer NOT NULL DEFAULT 0,
    user_id    text,
    decision   text NOT NULL DEFAULT 'COMMENT',
    body       text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX design_feedback_task_idx ON content.design_feedback (task_id, created_at);

CREATE TABLE content.asset_folders (
    id         text PRIMARY KEY,
    client_id  text NOT NULL,
    parent_id  text REFERENCES content.asset_folders(id),
    name       text NOT NULL,
    created_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX asset_folders_name_key ON content.asset_folders (client_id, coalesce(parent_id, ''), lower(name));

CREATE TABLE content.assets (
    id              text PRIMARY KEY,
    client_id       text NOT NULL,
    folder_id       text REFERENCES content.asset_folders(id) ON DELETE SET NULL,
    kind            content.asset_kind NOT NULL DEFAULT 'OTHER',
    name            text NOT NULL,
    description     text NOT NULL DEFAULT '',
    tags            text[] NOT NULL DEFAULT '{}',
    current_version integer NOT NULL DEFAULT 1,
    created_by      text,
    updated_by      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE INDEX assets_client_idx ON content.assets (client_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX assets_tags_idx ON content.assets USING gin (tags);

-- A new upload supersedes the previous version; history is kept.
CREATE TABLE content.asset_versions (
    id          text PRIMARY KEY,
    asset_id    text NOT NULL REFERENCES content.assets(id) ON DELETE CASCADE,
    version     integer NOT NULL,
    file_key    text NOT NULL,
    file_name   text NOT NULL,
    mime_type   text NOT NULL,
    size_bytes  bigint NOT NULL DEFAULT 0,
    note        text NOT NULL DEFAULT '',
    uploaded_by text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (asset_id, version)
);

-- Monthly content report per client; regenerating overwrites the row.
CREATE TABLE content.reports (
    id         text PRIMARY KEY,
    client_id  text NOT NULL,
    month      date NOT NULL,
    data       jsonb NOT NULL,
    summary    text NOT NULL DEFAULT '',
    ai_model   text NOT NULL DEFAULT '',
    created_by text,
    updated_by text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (client_id, month)
);

-- Deadline reminders already sent: one per item, deadline kind and day.
CREATE TABLE content.reminders (
    item_id text NOT NULL,
    kind    text NOT NULL,
    day     date NOT NULL,
    sent_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (item_id, kind, day)
);

-- +goose Down
DROP SCHEMA content CASCADE;
