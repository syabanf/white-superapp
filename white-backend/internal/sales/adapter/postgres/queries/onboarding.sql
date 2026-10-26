-- name: ListOnboardingTemplates :many
SELECT * FROM sales.onboarding_templates ORDER BY is_default DESC, name;

-- name: UpsertOnboardingTemplate :exec
INSERT INTO sales.onboarding_templates (id, name, is_default, items, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, is_default = EXCLUDED.is_default, items = EXCLUDED.items,
    updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: ClearDefaultOnboardingTemplate :exec
UPDATE sales.onboarding_templates SET is_default = false WHERE id <> $1 AND is_default;

-- name: GetOnboarding :one
SELECT * FROM sales.onboardings WHERE id = $1;

-- name: GetOnboardingByDeal :one
SELECT * FROM sales.onboardings WHERE deal_id = $1;

-- name: GetOnboardingByToken :one
SELECT * FROM sales.onboardings WHERE portal_token = $1;

-- name: ListOnboardings :many
SELECT * FROM sales.onboardings WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')) ORDER BY created_at DESC LIMIT 200;

-- name: InsertOnboarding :execrows
INSERT INTO sales.onboardings (id, deal_id, template_id, client_name, status, portal_token, brief, progress, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (deal_id) DO NOTHING;

-- name: UpdateOnboarding :exec
UPDATE sales.onboardings SET client_name = $2, status = $3, brief = $4, brief_submitted_at = $5, marketing_client_id = $6,
    progress = $7, completed_at = $8, updated_at = $9
WHERE id = $1;

-- name: ListOnboardingItems :many
SELECT * FROM sales.onboarding_items WHERE onboarding_id = ANY(sqlc.arg('ids')::text[]) ORDER BY onboarding_id, sort_order;

-- name: GetOnboardingItem :one
SELECT * FROM sales.onboarding_items WHERE id = $1;

-- name: UpsertOnboardingItem :exec
INSERT INTO sales.onboarding_items (id, onboarding_id, key, kind, title, description, owner_user_id, due_date, status, done_at, done_by, reminded_at, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, description = EXCLUDED.description, owner_user_id = EXCLUDED.owner_user_id,
    due_date = EXCLUDED.due_date, status = EXCLUDED.status, done_at = EXCLUDED.done_at, done_by = EXCLUDED.done_by,
    reminded_at = EXCLUDED.reminded_at;

-- name: InsertOnboardingAsset :exec
INSERT INTO sales.onboarding_assets (id, onboarding_id, file_name, content_type, size_bytes, storage_key, uploader_ip, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetOnboardingAsset :one
SELECT * FROM sales.onboarding_assets WHERE id = $1;

-- name: ListOnboardingAssets :many
SELECT * FROM sales.onboarding_assets WHERE onboarding_id = ANY(sqlc.arg('ids')::text[]) ORDER BY created_at;

-- name: DueOnboardingItems :many
SELECT i.*, o.client_name
FROM sales.onboarding_items i JOIN sales.onboardings o ON o.id = i.onboarding_id
WHERE o.status = 'aktif' AND i.status = 'todo' AND i.due_date IS NOT NULL AND i.due_date <= $1
  AND (i.reminded_at IS NULL OR i.reminded_at < now() - interval '20 hours')
ORDER BY i.due_date LIMIT 200;

-- name: MarkOnboardingItemReminded :exec
UPDATE sales.onboarding_items SET reminded_at = $2 WHERE id = $1;
