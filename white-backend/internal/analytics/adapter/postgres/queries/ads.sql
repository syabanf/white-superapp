-- name: ListAdAccounts :many
SELECT * FROM analytics.ad_accounts WHERE client_id = $1 ORDER BY created_at ASC;

-- name: GetAdAccount :one
SELECT * FROM analytics.ad_accounts WHERE id = $1;

-- name: InsertAdAccount :one
INSERT INTO analytics.ad_accounts (id, client_id, external_id, name, currency, connection_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: DeleteAdAccount :execrows
DELETE FROM analytics.ad_accounts WHERE id = $1;

-- name: DeleteAdAccountsByClient :exec
DELETE FROM analytics.ad_accounts WHERE client_id = $1;

-- name: TouchAdAccounts :exec
UPDATE analytics.ad_accounts SET last_synced_at = $2, updated_at = now() WHERE id = ANY($1::text[]);

-- name: ListCampaigns :many
SELECT * FROM analytics.ad_campaigns WHERE ad_account_id = ANY($1::text[]) ORDER BY name ASC;

-- name: CountCampaigns :one
SELECT COUNT(*)::int FROM analytics.ad_campaigns WHERE ad_account_id = ANY($1::text[]);

-- name: UpsertCampaign :one
INSERT INTO analytics.ad_campaigns (id, ad_account_id, external_id, name, objective, status, daily_budget, start_time)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (ad_account_id, external_id) DO UPDATE SET
    name         = EXCLUDED.name,
    objective    = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.objective ELSE analytics.ad_campaigns.objective END,
    status       = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.status ELSE analytics.ad_campaigns.status END,
    daily_budget = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.daily_budget ELSE analytics.ad_campaigns.daily_budget END,
    start_time   = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.start_time ELSE analytics.ad_campaigns.start_time END,
    updated_at   = now()
RETURNING id;

-- name: ListAdSets :many
SELECT s.* FROM analytics.ad_sets s
JOIN analytics.ad_campaigns c ON c.id = s.campaign_id
WHERE c.ad_account_id = ANY($1::text[])
  AND (sqlc.narg(campaign_id)::text IS NULL OR c.id = sqlc.narg(campaign_id)::text);

-- name: UpsertAdSet :one
INSERT INTO analytics.ad_sets (id, campaign_id, external_id, name, status, daily_budget)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (campaign_id, external_id) DO UPDATE SET
    name         = EXCLUDED.name,
    status       = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.status ELSE analytics.ad_sets.status END,
    daily_budget = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.daily_budget ELSE analytics.ad_sets.daily_budget END,
    updated_at   = now()
RETURNING id;

-- name: ListAds :many
SELECT d.* FROM analytics.ads d
JOIN analytics.ad_sets s ON s.id = d.ad_set_id
JOIN analytics.ad_campaigns c ON c.id = s.campaign_id
WHERE c.ad_account_id = ANY($1::text[])
  AND (sqlc.narg(campaign_id)::text IS NULL OR c.id = sqlc.narg(campaign_id)::text);

-- name: UpsertAd :one
INSERT INTO analytics.ads (id, ad_set_id, external_id, name, status, thumbnail_url)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (ad_set_id, external_id) DO UPDATE SET
    name          = EXCLUDED.name,
    status        = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.status ELSE analytics.ads.status END,
    thumbnail_url = CASE WHEN sqlc.arg(full_update)::boolean THEN EXCLUDED.thumbnail_url ELSE analytics.ads.thumbnail_url END,
    updated_at    = now()
RETURNING id;

-- name: ListInsights :many
SELECT * FROM analytics.ad_daily_insights
WHERE ad_account_id = ANY($1::text[]) AND ad_id IS NOT NULL AND date >= $2 AND date <= $3
  AND (sqlc.narg(campaign_id)::text IS NULL OR campaign_id = sqlc.narg(campaign_id)::text)
ORDER BY date ASC;

-- name: UpsertInsight :exec
INSERT INTO analytics.ad_daily_insights (id, ad_account_id, date, campaign_id, ad_set_id, ad_id, spend, impressions, reach, clicks, link_clicks, results, result_type, purchase_value)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
ON CONFLICT (ad_account_id, date, COALESCE(campaign_id, ''), COALESCE(ad_set_id, ''), COALESCE(ad_id, '')) DO UPDATE SET
    spend = EXCLUDED.spend, impressions = EXCLUDED.impressions, reach = EXCLUDED.reach, clicks = EXCLUDED.clicks,
    link_clicks = EXCLUDED.link_clicks, results = EXCLUDED.results, result_type = EXCLUDED.result_type, purchase_value = EXCLUDED.purchase_value;

-- name: SumAdSpendForClients :one
SELECT COALESCE(SUM(i.spend), 0)::float8 AS spend
FROM analytics.ad_daily_insights i
JOIN analytics.ad_accounts a ON a.id = i.ad_account_id
WHERE a.client_id = ANY($1::text[]) AND i.ad_id IS NOT NULL AND i.date >= $2 AND i.date <= $3;

-- name: ListDemographics :many
SELECT * FROM analytics.ad_demographics
WHERE ad_account_id = ANY($1::text[]) AND date_from <= sqlc.arg(range_to) AND date_to >= sqlc.arg(range_from)
  AND (sqlc.narg(campaign_id)::text IS NULL OR campaign_id = sqlc.narg(campaign_id)::text);

-- name: UpsertDemographic :exec
INSERT INTO analytics.ad_demographics (id, ad_account_id, date_from, date_to, campaign_id, age, gender, spend, impressions, reach, results)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (ad_account_id, date_from, date_to, COALESCE(campaign_id, ''), age, gender) DO UPDATE SET
    spend = EXCLUDED.spend, impressions = EXCLUDED.impressions, reach = EXCLUDED.reach, results = EXCLUDED.results;
