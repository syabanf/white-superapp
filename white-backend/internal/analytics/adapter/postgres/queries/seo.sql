-- name: ListSeoProperties :many
SELECT * FROM analytics.seo_properties WHERE client_id = $1 ORDER BY created_at ASC;

-- name: GetSeoProperty :one
SELECT * FROM analytics.seo_properties WHERE id = $1;

-- name: InsertSeoProperty :one
INSERT INTO analytics.seo_properties (id, client_id, site_url, ga4_property_id, connection_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SetAuditCadence :exec
UPDATE analytics.seo_properties SET audit_cadence = $2, updated_at = now() WHERE id = $1;

-- name: DeleteSeoProperty :execrows
DELETE FROM analytics.seo_properties WHERE id = $1;

-- name: DeleteSeoPropertiesByClient :exec
DELETE FROM analytics.seo_properties WHERE client_id = $1;

-- name: ListSeoDaily :many
SELECT * FROM analytics.seo_daily_metrics
WHERE property_id = ANY($1::text[]) AND date >= $2 AND date <= $3
ORDER BY date ASC;

-- name: UpsertSeoDaily :exec
INSERT INTO analytics.seo_daily_metrics (id, property_id, date, clicks, impressions, ctr, position)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (property_id, date) DO UPDATE SET
    clicks = EXCLUDED.clicks, impressions = EXCLUDED.impressions, ctr = EXCLUDED.ctr, position = EXCLUDED.position;

-- name: SumSeoClicksForClients :one
SELECT COALESCE(SUM(m.clicks), 0)::bigint AS clicks
FROM analytics.seo_daily_metrics m
JOIN analytics.seo_properties p ON p.id = m.property_id
WHERE p.client_id = ANY($1::text[]) AND m.date >= $2 AND m.date <= $3;

-- name: ListSeoDimension :many
SELECT * FROM analytics.seo_dimension_metrics
WHERE property_id = ANY($1::text[]) AND dimension = $2 AND date >= $3 AND date <= $4;

-- name: UpsertSeoDimension :exec
INSERT INTO analytics.seo_dimension_metrics (id, property_id, date, dimension, key, clicks, impressions, ctr, position)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (property_id, date, dimension, key) DO UPDATE SET
    clicks = EXCLUDED.clicks, impressions = EXCLUDED.impressions, ctr = EXCLUDED.ctr, position = EXCLUDED.position;

-- name: ListGa4Daily :many
SELECT * FROM analytics.ga4_daily_metrics
WHERE property_id = $1 AND date >= $2 AND date <= $3
ORDER BY date ASC;

-- name: UpsertGa4Daily :exec
INSERT INTO analytics.ga4_daily_metrics (id, property_id, date, sessions, organic_sessions, users, engaged_sessions, conversions, avg_engagement_sec)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (property_id, date) DO UPDATE SET
    sessions = EXCLUDED.sessions, organic_sessions = EXCLUDED.organic_sessions, users = EXCLUDED.users,
    engaged_sessions = EXCLUDED.engaged_sessions, conversions = EXCLUDED.conversions, avg_engagement_sec = EXCLUDED.avg_engagement_sec;

-- name: LatestAudit :one
SELECT * FROM analytics.seo_audits
WHERE property_id = ANY($1::text[]) AND strategy = $2
ORDER BY run_at DESC
LIMIT 1;

-- name: LatestAuditAny :one
SELECT * FROM analytics.seo_audits WHERE property_id = $1 ORDER BY run_at DESC LIMIT 1;

-- name: ListMobileAuditHistory :many
SELECT * FROM analytics.seo_audits
WHERE property_id = $1 AND strategy = 'MOBILE'
ORDER BY run_at ASC
LIMIT $2;

-- name: InsertAudit :exec
INSERT INTO analytics.seo_audits (id, property_id, url, strategy, run_at, performance, seo, accessibility, best_practices,
                                  lcp_ms, inp_ms, cls, fcp_ms, ttfb_ms, tbt_ms, speed_index_ms, raw)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);

-- name: LatestCrawl :one
SELECT * FROM analytics.seo_crawls
WHERE property_id = ANY($1::text[])
  AND (sqlc.narg(status)::analytics.job_status IS NULL OR status = sqlc.narg(status)::analytics.job_status)
ORDER BY started_at DESC
LIMIT 1;

-- name: ListRecentSuccessCrawls :many
SELECT * FROM analytics.seo_crawls
WHERE property_id = $1 AND status = 'SUCCESS'
ORDER BY started_at DESC
LIMIT $2;

-- name: InsertCrawl :exec
INSERT INTO analytics.seo_crawls (id, property_id, started_at, status, max_pages)
VALUES ($1, $2, $3, $4, $5);

-- name: UpdateCrawl :exec
UPDATE analytics.seo_crawls SET status = $2, finished_at = $3, pages_crawled = $4, error = $5 WHERE id = $1;

-- name: InsertCrawlPage :copyfrom
INSERT INTO analytics.seo_crawl_pages (id, crawl_id, url, status_code, title, meta_description, h1_count, word_count, canonical, indexable,
                                       images_total, images_missing_alt, internal_links, external_links, load_ms, has_json_ld, has_hreflang)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);

-- name: InsertIssue :copyfrom
INSERT INTO analytics.seo_issues (id, crawl_id, severity, code, message, url)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListCrawlPages :many
SELECT * FROM analytics.seo_crawl_pages WHERE crawl_id = $1 ORDER BY url ASC;

-- name: ListIssues :many
SELECT * FROM analytics.seo_issues WHERE crawl_id = $1 ORDER BY severity ASC, code ASC;

-- name: CountIssuesBySeverity :many
SELECT crawl_id, severity, COUNT(*)::int AS n
FROM analytics.seo_issues
WHERE crawl_id = ANY($1::text[])
GROUP BY crawl_id, severity;
