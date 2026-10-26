-- +goose Up
-- Last known engagement per published item. The dashboard and reports read
-- analytics first and fall back to this snapshot, so numbers survive an
-- analytics outage and posts analytics never synced (demo seed) still show.
CREATE TABLE content.item_metrics (
    item_id    text PRIMARY KEY REFERENCES content.items(id) ON DELETE CASCADE,
    likes      bigint NOT NULL DEFAULT 0,
    comments   bigint NOT NULL DEFAULT 0,
    shares     bigint NOT NULL DEFAULT 0,
    saves      bigint NOT NULL DEFAULT 0,
    reach      bigint NOT NULL DEFAULT 0,
    views      bigint NOT NULL DEFAULT 0,
    synced_at  timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE content.item_metrics;
