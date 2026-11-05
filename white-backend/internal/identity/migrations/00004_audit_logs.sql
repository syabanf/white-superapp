-- +goose Up
-- Who changed what, recorded at the gateway. Request bodies are never stored.
CREATE TABLE identity.audit_logs (
    id         text PRIMARY KEY,
    at         timestamptz NOT NULL,
    user_id    text NOT NULL DEFAULT '',
    user_name  text NOT NULL DEFAULT '',
    role       text NOT NULL DEFAULT '',
    method     text NOT NULL,
    path       text NOT NULL,
    status     integer NOT NULL,
    ip         text NOT NULL DEFAULT '',
    user_agent text NOT NULL DEFAULT ''
);
CREATE INDEX audit_logs_at_idx ON identity.audit_logs (at DESC);
CREATE INDEX audit_logs_user_idx ON identity.audit_logs (user_id, at DESC);

-- +goose Down
DROP TABLE identity.audit_logs;
