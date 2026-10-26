-- name: GetPost :one
SELECT * FROM publishing.content_posts WHERE id = $1;

-- name: GetPostInClient :one
SELECT * FROM publishing.content_posts WHERE id = $1 AND client_id = $2;

-- name: InsertPost :one
INSERT INTO publishing.content_posts (id, client_id, title, body, status, scheduled_at, link_url, first_comment, labels, utm, created_by_id, format)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdatePostContent :one
UPDATE publishing.content_posts SET
    title = $2,
    body = $3,
    status = $4,
    scheduled_at = $5,
    link_url = $6,
    first_comment = $7,
    labels = $8,
    utm = $9,
    format = $10,
    approved_by_id = CASE WHEN sqlc.arg(clear_approval)::bool THEN NULL ELSE approved_by_id END,
    approved_at = CASE WHEN sqlc.arg(clear_approval)::bool THEN NULL ELSE approved_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdatePostStatus :one
UPDATE publishing.content_posts SET
    status = $2,
    scheduled_at = CASE WHEN sqlc.arg(set_schedule)::bool THEN sqlc.narg(scheduled_at)::timestamptz ELSE scheduled_at END,
    review_note = CASE WHEN sqlc.arg(set_review)::bool THEN sqlc.narg(review_note)::text ELSE review_note END,
    approved_by_id = CASE WHEN sqlc.arg(set_approval)::bool THEN sqlc.narg(approved_by_id)::text ELSE approved_by_id END,
    approved_at = CASE WHEN sqlc.arg(set_approval)::bool THEN sqlc.narg(approved_at)::timestamptz ELSE approved_at END,
    published_at = CASE WHEN sqlc.arg(set_published)::bool THEN sqlc.narg(published_at)::timestamptz ELSE published_at END,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdatePostScheduledAt :exec
UPDATE publishing.content_posts SET scheduled_at = sqlc.narg(scheduled_at)::timestamptz, updated_at = now() WHERE id = $1;

-- name: ClaimPost :execrows
UPDATE publishing.content_posts SET status = 'PUBLISHING', updated_at = now()
WHERE id = $1 AND status::text = ANY(sqlc.arg(statuses)::text[]);

-- name: ClaimDuePost :execrows
UPDATE publishing.content_posts SET status = 'PUBLISHING', updated_at = now()
WHERE id = $1 AND status = 'SCHEDULED' AND scheduled_at <= sqlc.arg(now_at)::timestamptz;

-- name: ListDuePosts :many
SELECT * FROM publishing.content_posts
WHERE status = 'SCHEDULED' AND scheduled_at <= sqlc.arg(now_at)::timestamptz
ORDER BY scheduled_at ASC
LIMIT sqlc.arg(max_rows);

-- name: ListPostsPublishing :many
SELECT * FROM publishing.content_posts WHERE id = ANY(sqlc.arg(ids)::text[]) AND status = 'PUBLISHING';

-- name: DeletePost :execrows
DELETE FROM publishing.content_posts WHERE id = $1;

-- name: ListPosts :many
SELECT * FROM publishing.content_posts WHERE client_id = $1 ORDER BY scheduled_at DESC, created_at DESC;

-- name: CountPostsByStatus :many
SELECT status, count(*)::int AS count FROM publishing.content_posts WHERE client_id = $1 GROUP BY status;

-- name: ListCalendarPosts :many
SELECT * FROM publishing.content_posts
WHERE client_id = $1 AND (
    (published_at >= sqlc.arg(from_at)::timestamptz AND published_at < sqlc.arg(to_at)::timestamptz)
    OR (published_at IS NULL AND scheduled_at >= sqlc.arg(from_at)::timestamptz AND scheduled_at < sqlc.arg(to_at)::timestamptz)
    OR (published_at IS NULL AND scheduled_at IS NULL AND created_at >= sqlc.arg(from_at)::timestamptz AND created_at < sqlc.arg(to_at)::timestamptz)
);

-- name: PublishingStats :one
SELECT
    count(*) FILTER (WHERE status = 'SCHEDULED' AND scheduled_at >= sqlc.arg(now_at)::timestamptz AND scheduled_at < sqlc.arg(in_7d)::timestamptz)::int AS scheduled_7d,
    count(*) FILTER (WHERE status = 'IN_REVIEW')::int AS in_review,
    count(*) FILTER (WHERE status = 'PUBLISHED' AND published_at >= sqlc.arg(ago_30d)::timestamptz)::int AS published_30d,
    count(*) FILTER (WHERE status = 'FAILED')::int AS failed,
    count(*) FILTER (WHERE status = 'DRAFT')::int AS drafts
FROM publishing.content_posts
WHERE client_id = $1;

-- name: DeletePostsByClient :execrows
DELETE FROM publishing.content_posts WHERE client_id = $1;

-- name: NullPostCreatedBy :exec
UPDATE publishing.content_posts SET created_by_id = NULL WHERE created_by_id = $1;

-- name: NullPostApprovedBy :exec
UPDATE publishing.content_posts SET approved_by_id = NULL WHERE approved_by_id = $1;

-- name: ListTargets :many
SELECT * FROM publishing.post_targets WHERE post_id = $1 ORDER BY platform ASC, id ASC;

-- name: ListTargetsForPosts :many
SELECT post_id, platform, status FROM publishing.post_targets WHERE post_id = ANY(sqlc.arg(post_ids)::text[]) ORDER BY post_id, id;

-- name: UpsertTarget :exec
INSERT INTO publishing.post_targets (id, post_id, social_account_id, platform, body_override, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (post_id, social_account_id) DO UPDATE
SET platform = EXCLUDED.platform, body_override = EXCLUDED.body_override, status = EXCLUDED.status;

-- name: DeleteTargetsNotIn :exec
DELETE FROM publishing.post_targets WHERE post_id = $1 AND NOT (social_account_id = ANY(sqlc.arg(keep)::text[]));

-- name: SetTargetsStatusExceptPublished :exec
UPDATE publishing.post_targets SET status = $2 WHERE post_id = $1 AND status <> 'PUBLISHED';

-- name: SetTargetsStatus :exec
UPDATE publishing.post_targets SET status = sqlc.arg(status) WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: UpdateTargetResult :exec
UPDATE publishing.post_targets
SET status = $2, external_id = $3, permalink = $4, error = $5, attempts = $6, published_at = $7
WHERE id = $1;

-- name: ListPublishedTargets :many
SELECT t.post_id, t.social_account_id, t.platform, t.external_id, t.published_at, p.title, p.body
FROM publishing.post_targets t
JOIN publishing.content_posts p ON p.id = t.post_id
WHERE p.client_id = $1 AND t.status = 'PUBLISHED' AND t.published_at >= sqlc.arg(from_at)::timestamptz AND t.published_at < sqlc.arg(to_at)::timestamptz;

-- name: CountFailedTargets :one
SELECT count(*)::int
FROM publishing.post_targets t
JOIN publishing.content_posts p ON p.id = t.post_id
WHERE p.client_id = $1 AND t.status = 'FAILED' AND (
    (p.scheduled_at >= sqlc.arg(from_at)::timestamptz AND p.scheduled_at < sqlc.arg(to_at)::timestamptz)
    OR (p.scheduled_at IS NULL AND p.created_at >= sqlc.arg(from_at)::timestamptz AND p.created_at < sqlc.arg(to_at)::timestamptz)
);

-- name: InsertComment :one
INSERT INTO publishing.post_comments (id, post_id, user_id, body) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListComments :many
SELECT * FROM publishing.post_comments WHERE post_id = $1 ORDER BY created_at ASC;

-- name: NullCommentUser :exec
UPDATE publishing.post_comments SET user_id = NULL WHERE user_id = $1;

-- name: InsertActivity :exec
INSERT INTO publishing.post_activities (id, post_id, user_id, type, meta) VALUES ($1, $2, $3, $4, $5);

-- name: ListActivities :many
SELECT * FROM publishing.post_activities WHERE post_id = $1 ORDER BY created_at DESC;

-- name: NullActivityUser :exec
UPDATE publishing.post_activities SET user_id = NULL WHERE user_id = $1;

-- name: InsertSyncJob :exec
INSERT INTO publishing.sync_jobs (id, client_id, kind, status, started_at, finished_at, message, error)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListSyncJobs :many
SELECT * FROM publishing.sync_jobs WHERE client_id = $1 ORDER BY started_at DESC LIMIT $2;

-- name: DeleteSyncJobsBefore :execrows
DELETE FROM publishing.sync_jobs WHERE started_at < $1;

-- name: DeleteSyncJobsByClient :execrows
DELETE FROM publishing.sync_jobs WHERE client_id = $1;
