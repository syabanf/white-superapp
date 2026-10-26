-- name: ListSocialAccounts :many
SELECT * FROM analytics.social_accounts
WHERE client_id = $1
  AND (sqlc.arg(include_competitors)::boolean OR NOT is_competitor)
  AND (sqlc.narg(platform)::analytics.social_platform IS NULL OR platform = sqlc.narg(platform)::analytics.social_platform)
ORDER BY is_competitor ASC, platform ASC, created_at ASC;

-- name: GetSocialAccount :one
SELECT * FROM analytics.social_accounts WHERE id = $1;

-- name: FindCompetitorByUsername :one
SELECT * FROM analytics.social_accounts
WHERE client_id = $1 AND platform = 'INSTAGRAM' AND username = $2 AND is_competitor
LIMIT 1;

-- name: InsertSocialAccount :one
INSERT INTO analytics.social_accounts (id, client_id, platform, external_id, username, display_name, avatar_url, biography, is_competitor, connection_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateSocialAccountProfile :exec
UPDATE analytics.social_accounts SET
    username     = COALESCE(sqlc.narg(username), username),
    display_name = $2,
    biography    = CASE WHEN sqlc.arg(set_biography)::boolean THEN sqlc.narg(biography) ELSE biography END,
    avatar_url   = COALESCE(sqlc.narg(avatar_url), avatar_url),
    updated_at   = now()
WHERE id = $1;

-- name: DeleteSocialAccount :execrows
DELETE FROM analytics.social_accounts WHERE id = $1;

-- name: DeleteSocialAccountsByClient :exec
DELETE FROM analytics.social_accounts WHERE client_id = $1;

-- name: LatestFollowersByAccount :many
SELECT DISTINCT ON (social_account_id) social_account_id, followers
FROM analytics.social_snapshots
WHERE social_account_id = ANY($1::text[])
ORDER BY social_account_id, date DESC;

-- name: ListSnapshots :many
SELECT * FROM analytics.social_snapshots
WHERE social_account_id = ANY($1::text[]) AND date >= $2 AND date <= $3
ORDER BY date ASC, social_account_id ASC;

-- name: ListRecentSnapshots :many
SELECT * FROM analytics.social_snapshots
WHERE social_account_id = $1
ORDER BY date DESC
LIMIT $2;

-- name: UpsertSnapshot :exec
INSERT INTO analytics.social_snapshots (id, social_account_id, date, followers, following, media_count, reach, impressions, profile_views, website_clicks)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (social_account_id, date) DO UPDATE SET
    followers      = COALESCE(sqlc.narg(u_followers), analytics.social_snapshots.followers),
    following      = COALESCE(sqlc.narg(u_following), analytics.social_snapshots.following),
    media_count    = COALESCE(sqlc.narg(u_media_count), analytics.social_snapshots.media_count),
    reach          = COALESCE(sqlc.narg(u_reach), analytics.social_snapshots.reach),
    impressions    = COALESCE(sqlc.narg(u_impressions), analytics.social_snapshots.impressions),
    profile_views  = COALESCE(sqlc.narg(u_profile_views), analytics.social_snapshots.profile_views),
    website_clicks = COALESCE(sqlc.narg(u_website_clicks), analytics.social_snapshots.website_clicks);

-- name: InsertSnapshotIgnore :exec
INSERT INTO analytics.social_snapshots (id, social_account_id, date, followers, following, media_count, reach, impressions, profile_views, website_clicks)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT DO NOTHING;

-- name: ListPosts :many
SELECT * FROM analytics.social_posts
WHERE social_account_id = ANY($1::text[]) AND published_at >= $2 AND published_at < $3
ORDER BY published_at DESC;

-- name: ListOwnPostsForClient :many
SELECT p.*, a.platform, a.username
FROM analytics.social_posts p
JOIN analytics.social_accounts a ON a.id = p.social_account_id
WHERE a.client_id = $1 AND NOT a.is_competitor AND p.published_at >= $2 AND p.published_at < $3
ORDER BY p.published_at DESC;

-- name: UpsertPost :exec
INSERT INTO analytics.social_posts (id, social_account_id, external_id, caption, media_type, product_type, permalink, media_url, thumbnail_url,
                                    published_at, likes, comments, shares, saves, views, reach, impressions, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, now())
ON CONFLICT (social_account_id, external_id) DO UPDATE SET
    caption       = COALESCE(sqlc.narg(u_caption), analytics.social_posts.caption),
    media_type    = COALESCE(sqlc.narg(u_media_type), analytics.social_posts.media_type),
    product_type  = COALESCE(sqlc.narg(u_product_type), analytics.social_posts.product_type),
    permalink     = COALESCE(sqlc.narg(u_permalink), analytics.social_posts.permalink),
    media_url     = COALESCE(sqlc.narg(u_media_url), analytics.social_posts.media_url),
    thumbnail_url = COALESCE(sqlc.narg(u_thumbnail_url), analytics.social_posts.thumbnail_url),
    likes         = COALESCE(sqlc.narg(u_likes), analytics.social_posts.likes),
    comments      = COALESCE(sqlc.narg(u_comments), analytics.social_posts.comments),
    shares        = COALESCE(sqlc.narg(u_shares), analytics.social_posts.shares),
    saves         = COALESCE(sqlc.narg(u_saves), analytics.social_posts.saves),
    views         = COALESCE(sqlc.narg(u_views), analytics.social_posts.views),
    reach         = COALESCE(sqlc.narg(u_reach), analytics.social_posts.reach),
    impressions   = COALESCE(sqlc.narg(u_impressions), analytics.social_posts.impressions),
    synced_at     = now();

-- name: InsertPostIgnore :exec
INSERT INTO analytics.social_posts (id, social_account_id, external_id, caption, media_type, product_type, permalink, media_url, thumbnail_url,
                                    published_at, likes, comments, shares, saves, views, reach, impressions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
ON CONFLICT DO NOTHING;

-- name: ListOwnPostsFiltered :many
SELECT p.* FROM analytics.social_posts p
JOIN analytics.social_accounts a ON a.id = p.social_account_id
WHERE a.client_id = sqlc.arg(client_id) AND NOT a.is_competitor
  AND (cardinality(sqlc.arg(account_ids)::text[]) = 0 OR p.social_account_id = ANY(sqlc.arg(account_ids)::text[]))
  AND (cardinality(sqlc.arg(external_ids)::text[]) = 0 OR p.external_id = ANY(sqlc.arg(external_ids)::text[]))
  AND (sqlc.narg(from_ts)::timestamptz IS NULL OR p.published_at >= sqlc.narg(from_ts)::timestamptz)
  AND (sqlc.narg(to_ts)::timestamptz IS NULL OR p.published_at < sqlc.narg(to_ts)::timestamptz)
ORDER BY p.published_at DESC
LIMIT sqlc.arg(row_limit);
