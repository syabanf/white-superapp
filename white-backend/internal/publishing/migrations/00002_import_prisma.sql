-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Rows keep their cuid ids. Prisma stored `timestamp(3)` in UTC, so every
-- instant is read AT TIME ZONE 'UTC'.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."ContentPost"') IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO publishing.media_assets (id, client_id, kind, url, thumbnail_url, filename, mime_type, size_bytes,
                                         width, height, duration_sec, alt_text, tags, uploaded_by_id, created_at)
    SELECT id, "clientId", kind::text::publishing.media_kind, url, "thumbnailUrl", filename, "mimeType", "sizeBytes",
           width, height, "durationSec", "altText", COALESCE(tags, '{}'), "uploadedById", "createdAt" AT TIME ZONE 'UTC'
    FROM public."MediaAsset"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO publishing.content_posts (id, client_id, title, body, status, scheduled_at, published_at, link_url,
                                          first_comment, labels, utm, review_note, created_by_id, approved_by_id,
                                          approved_at, created_at, updated_at)
    SELECT id, "clientId", title, body, status::text::publishing.post_status,
           "scheduledAt" AT TIME ZONE 'UTC', "publishedAt" AT TIME ZONE 'UTC', "linkUrl",
           "firstComment", COALESCE(labels, '{}'), utm, "reviewNote", "createdById", "approvedById",
           "approvedAt" AT TIME ZONE 'UTC', "createdAt" AT TIME ZONE 'UTC', "updatedAt" AT TIME ZONE 'UTC'
    FROM public."ContentPost"
    ON CONFLICT (id) DO NOTHING;

    IF to_regclass('public."SocialAccount"') IS NOT NULL THEN
        INSERT INTO publishing.post_targets (id, post_id, social_account_id, platform, body_override, status,
                                             external_id, permalink, error, attempts, published_at)
        SELECT t.id, t."postId", t."socialAccountId", COALESCE(a.platform::text, 'INSTAGRAM'), t."bodyOverride",
               t.status::text::publishing.post_status, t."externalId", t.permalink, t.error, t.attempts,
               t."publishedAt" AT TIME ZONE 'UTC'
        FROM public."PostTarget" t
        LEFT JOIN public."SocialAccount" a ON a.id = t."socialAccountId"
        ON CONFLICT (id) DO NOTHING;
    ELSE
        INSERT INTO publishing.post_targets (id, post_id, social_account_id, platform, body_override, status,
                                             external_id, permalink, error, attempts, published_at)
        SELECT t.id, t."postId", t."socialAccountId", 'INSTAGRAM', t."bodyOverride",
               t.status::text::publishing.post_status, t."externalId", t.permalink, t.error, t.attempts,
               t."publishedAt" AT TIME ZONE 'UTC'
        FROM public."PostTarget" t
        ON CONFLICT (id) DO NOTHING;
    END IF;

    INSERT INTO publishing.post_media (post_id, asset_id, sort_order)
    SELECT "postId", "assetId", "order"
    FROM public."PostMedia"
    ON CONFLICT DO NOTHING;

    INSERT INTO publishing.post_comments (id, post_id, user_id, body, created_at)
    SELECT id, "postId", "userId", body, "createdAt" AT TIME ZONE 'UTC'
    FROM public."PostComment"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO publishing.post_activities (id, post_id, user_id, type, meta, created_at)
    SELECT id, "postId", "userId", type, meta, "createdAt" AT TIME ZONE 'UTC'
    FROM public."PostActivity"
    ON CONFLICT (id) DO NOTHING;

    IF to_regclass('public."SyncJob"') IS NOT NULL THEN
        INSERT INTO publishing.sync_jobs (id, client_id, kind, status, started_at, finished_at, message, error)
        SELECT id, "clientId", kind, status::text::publishing.job_status, "startedAt" AT TIME ZONE 'UTC',
               "finishedAt" AT TIME ZONE 'UTC', message, error
        FROM public."SyncJob"
        WHERE kind = 'PUBLISH'
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
