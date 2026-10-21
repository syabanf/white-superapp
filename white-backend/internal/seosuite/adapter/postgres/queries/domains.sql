-- name: ListCompetitorDomains :many
SELECT * FROM seosuite.competitor_domains WHERE property_id = $1 ORDER BY created_at ASC, id ASC;

-- name: GetCompetitorDomain :one
SELECT * FROM seosuite.competitor_domains WHERE id = $1;

-- name: FindCompetitorDomain :one
SELECT * FROM seosuite.competitor_domains WHERE property_id = $1 AND domain = $2;

-- name: CountCompetitorDomains :one
SELECT count(*)::int FROM seosuite.competitor_domains WHERE property_id = $1;

-- name: InsertCompetitorDomain :exec
INSERT INTO seosuite.competitor_domains (id, client_id, property_id, domain, created_at) VALUES ($1, $2, $3, $4, $5);

-- name: DeleteCompetitorDomain :exec
DELETE FROM seosuite.competitor_domains WHERE id = $1;

-- name: DeleteDomainSnapshotsOf :exec
DELETE FROM seosuite.domain_snapshots WHERE property_id = $1 AND domain = $2;

-- name: UpsertDomainSnapshot :exec
INSERT INTO seosuite.domain_snapshots (id, client_id, property_id, domain, date, organic_keywords, organic_traffic, organic_cost, top3, top10, top100, backlinks, referring_domains, domain_rank)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (property_id, domain, date) DO UPDATE
SET organic_keywords = EXCLUDED.organic_keywords, organic_traffic = EXCLUDED.organic_traffic, organic_cost = EXCLUDED.organic_cost,
    top3 = EXCLUDED.top3, top10 = EXCLUDED.top10, top100 = EXCLUDED.top100, backlinks = EXCLUDED.backlinks,
    referring_domains = EXCLUDED.referring_domains, domain_rank = EXCLUDED.domain_rank;

-- name: ListDomainSnapshots :many
SELECT * FROM seosuite.domain_snapshots
WHERE property_id = $1 AND domain = ANY(sqlc.arg(domains)::text[]) AND date >= sqlc.arg(from_date) AND date <= sqlc.arg(to_date)
ORDER BY date ASC;

-- name: ListLatestDomainSnapshots :many
SELECT DISTINCT ON (domain) * FROM seosuite.domain_snapshots
WHERE property_id = $1 AND domain = ANY(sqlc.arg(domains)::text[])
ORDER BY domain, date DESC;

-- name: DeleteDomainSnapshotsBefore :execrows
DELETE FROM seosuite.domain_snapshots WHERE date < $1;
