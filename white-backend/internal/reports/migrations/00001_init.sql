-- +goose Up
CREATE SCHEMA IF NOT EXISTS reports;

-- client_id and created_by_id point at identity rows as plain text: no
-- foreign keys across schemas. Purges arrive as client.deleted and
-- user.deleted events.
CREATE TABLE reports.reports (
    id            text PRIMARY KEY,
    client_id     text NOT NULL,
    title         text NOT NULL,
    date_from     date NOT NULL,
    date_to       date NOT NULL,
    compare       boolean NOT NULL DEFAULT true,
    modules       text[] NOT NULL DEFAULT '{SOCIAL,SEO,ADS}',
    language      text NOT NULL DEFAULT 'id',
    ai_summary    text,
    created_by_id text,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reports_client_created_idx ON reports.reports (client_id, created_at);
CREATE INDEX reports_created_by_idx ON reports.reports (created_by_id);

CREATE TABLE reports.ai_insights (
    id         text PRIMARY KEY,
    client_id  text NOT NULL,
    module     text NOT NULL,
    date_from  date NOT NULL,
    date_to    date NOT NULL,
    language   text NOT NULL DEFAULT 'id',
    model      text NOT NULL,
    content    text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_insights_lookup_idx ON reports.ai_insights (client_id, module, date_from, date_to);

-- +goose Down
DROP SCHEMA IF EXISTS reports CASCADE;
