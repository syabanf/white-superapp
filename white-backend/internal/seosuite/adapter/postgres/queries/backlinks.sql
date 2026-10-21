-- name: ListBacklinks :many
SELECT * FROM seosuite.backlinks WHERE property_id = $1 ORDER BY is_lost ASC, domain_rank DESC NULLS LAST, id ASC;

-- name: ListBacklinkKeys :many
SELECT source_url, target_url, is_lost FROM seosuite.backlinks WHERE property_id = $1;

-- name: UpsertBacklink :batchexec
INSERT INTO seosuite.backlinks (id, client_id, property_id, source_url, source_domain, target_url, anchor, dofollow, domain_rank, spam_score, first_seen, last_seen, is_lost)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (property_id, source_url, target_url) DO UPDATE
SET anchor = EXCLUDED.anchor, dofollow = EXCLUDED.dofollow, domain_rank = EXCLUDED.domain_rank,
    spam_score = EXCLUDED.spam_score, last_seen = EXCLUDED.last_seen, is_lost = EXCLUDED.is_lost;

-- name: UpsertBacklinkSnapshot :exec
INSERT INTO seosuite.backlink_snapshots (id, client_id, property_id, date, backlinks, referring_domains, dofollow, nofollow, new_backlinks, lost_backlinks, domain_rank, toxic_share)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (property_id, date) DO UPDATE
SET backlinks = EXCLUDED.backlinks, referring_domains = EXCLUDED.referring_domains, dofollow = EXCLUDED.dofollow,
    nofollow = EXCLUDED.nofollow, new_backlinks = EXCLUDED.new_backlinks, lost_backlinks = EXCLUDED.lost_backlinks,
    domain_rank = EXCLUDED.domain_rank, toxic_share = EXCLUDED.toxic_share;

-- name: GetLatestBacklinkSnapshot :one
SELECT * FROM seosuite.backlink_snapshots WHERE property_id = $1 ORDER BY date DESC LIMIT 1;

-- name: GetBacklinkSnapshotOnOrBefore :one
SELECT * FROM seosuite.backlink_snapshots WHERE property_id = $1 AND date <= $2 ORDER BY date DESC LIMIT 1;

-- name: ListBacklinkSnapshots :many
SELECT * FROM seosuite.backlink_snapshots WHERE property_id = $1 AND date >= sqlc.arg(from_date) AND date <= sqlc.arg(to_date) ORDER BY date ASC;

-- name: DeleteBacklinkSnapshotsBefore :execrows
DELETE FROM seosuite.backlink_snapshots WHERE date < $1;
