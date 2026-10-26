-- name: GetLead :one
SELECT * FROM sales.leads WHERE id = $1 AND deleted_at IS NULL;

-- name: ListLeads :many
SELECT * FROM sales.leads
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('temperature')::text IS NULL OR temperature = sqlc.narg('temperature'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('owner')::text IS NULL OR owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('q')::text IS NULL OR org_name ILIKE '%' || sqlc.narg('q') || '%' OR pic_name ILIKE '%' || sqlc.narg('q') || '%'
       OR pic_phone ILIKE '%' || sqlc.narg('q') || '%' OR pic_email ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR created_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR created_at < sqlc.narg('to_at'))
  AND (sqlc.narg('score_min')::int IS NULL OR score >= sqlc.narg('score_min'))
  AND (sqlc.narg('score_max')::int IS NULL OR score <= sqlc.narg('score_max'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountLeads :one
SELECT count(*) FROM sales.leads
WHERE deleted_at IS NULL
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('temperature')::text IS NULL OR temperature = sqlc.narg('temperature'))
  AND (sqlc.narg('source')::text IS NULL OR source = sqlc.narg('source'))
  AND (sqlc.narg('owner')::text IS NULL OR owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('q')::text IS NULL OR org_name ILIKE '%' || sqlc.narg('q') || '%' OR pic_name ILIKE '%' || sqlc.narg('q') || '%'
       OR pic_phone ILIKE '%' || sqlc.narg('q') || '%' OR pic_email ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR created_at >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR created_at < sqlc.narg('to_at'))
  AND (sqlc.narg('score_min')::int IS NULL OR score >= sqlc.narg('score_min'))
  AND (sqlc.narg('score_max')::int IS NULL OR score <= sqlc.narg('score_max'));

-- name: UpsertLead :exec
INSERT INTO sales.leads (id, org_name, org_type, pic_name, pic_title, pic_phone, pic_email, city, source, temperature, status,
    notes, owner_user_id, account_id, contact_id, custom, utm_source, utm_medium, utm_campaign, utm_term, utm_content,
    landing_page, referrer, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
ON CONFLICT (id) DO UPDATE SET org_name = EXCLUDED.org_name, org_type = EXCLUDED.org_type, pic_name = EXCLUDED.pic_name,
    pic_title = EXCLUDED.pic_title, pic_phone = EXCLUDED.pic_phone, pic_email = EXCLUDED.pic_email, city = EXCLUDED.city,
    source = EXCLUDED.source, temperature = EXCLUDED.temperature, status = EXCLUDED.status, notes = EXCLUDED.notes,
    owner_user_id = EXCLUDED.owner_user_id, account_id = EXCLUDED.account_id, contact_id = EXCLUDED.contact_id,
    custom = EXCLUDED.custom, utm_source = EXCLUDED.utm_source, utm_medium = EXCLUDED.utm_medium,
    utm_campaign = EXCLUDED.utm_campaign, utm_term = EXCLUDED.utm_term, utm_content = EXCLUDED.utm_content,
    landing_page = EXCLUDED.landing_page, referrer = EXCLUDED.referrer, updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteLead :exec
UPDATE sales.leads SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: FindLeadByContact :one
SELECT * FROM sales.leads
WHERE deleted_at IS NULL
  AND ((sqlc.narg('phone')::text IS NOT NULL AND pic_phone = sqlc.narg('phone'))
       OR (sqlc.narg('email')::text IS NOT NULL AND lower(pic_email) = lower(sqlc.narg('email'))))
ORDER BY created_at DESC LIMIT 1;

-- name: LeadKeys :many
SELECT coalesce(pic_phone, '')::text AS phone, org_name FROM sales.leads WHERE deleted_at IS NULL;

-- name: UpdateLeadScore :exec
UPDATE sales.leads SET score = $2, score_breakdown = $3, score_updated_at = $4 WHERE id = $1;

-- name: ListLeadIDs :many
SELECT id FROM sales.leads WHERE deleted_at IS NULL ORDER BY created_at;

-- name: StaleLeadIDs :many
SELECT id FROM sales.leads WHERE deleted_at IS NULL AND status IN ('baru', 'dihubungi', 'qualified') AND updated_at < $1 LIMIT 200;

-- name: GetAccount :one
SELECT * FROM sales.accounts WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAccounts :many
SELECT * FROM sales.accounts
WHERE deleted_at IS NULL AND (sqlc.narg('q')::text IS NULL OR name ILIKE '%' || sqlc.narg('q') || '%')
ORDER BY name LIMIT sqlc.arg('lim');

-- name: UpsertAccount :exec
INSERT INTO sales.accounts (id, name, account_type, industry, address, city, phone, email, website, npwp, notes,
    owner_user_id, custom, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, account_type = EXCLUDED.account_type, industry = EXCLUDED.industry,
    address = EXCLUDED.address, city = EXCLUDED.city, phone = EXCLUDED.phone, email = EXCLUDED.email,
    website = EXCLUDED.website, npwp = EXCLUDED.npwp, notes = EXCLUDED.notes, owner_user_id = EXCLUDED.owner_user_id,
    custom = EXCLUDED.custom, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteAccount :exec
UPDATE sales.accounts SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: GetContact :one
SELECT * FROM sales.contacts WHERE id = $1 AND deleted_at IS NULL;

-- name: ListContacts :many
SELECT * FROM sales.contacts
WHERE deleted_at IS NULL
  AND (sqlc.narg('account')::text IS NULL OR account_id = sqlc.narg('account'))
  AND (sqlc.narg('q')::text IS NULL OR name ILIKE '%' || sqlc.narg('q') || '%' OR phone ILIKE '%' || sqlc.narg('q') || '%' OR email ILIKE '%' || sqlc.narg('q') || '%')
ORDER BY is_primary DESC, name LIMIT sqlc.arg('lim');

-- name: UpsertContact :exec
INSERT INTO sales.contacts (id, account_id, name, title, phone, email, is_primary, notes, custom, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (id) DO UPDATE SET account_id = EXCLUDED.account_id, name = EXCLUDED.name, title = EXCLUDED.title,
    phone = EXCLUDED.phone, email = EXCLUDED.email, is_primary = EXCLUDED.is_primary, notes = EXCLUDED.notes,
    custom = EXCLUDED.custom, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteContact :exec
UPDATE sales.contacts SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCustomFields :many
SELECT * FROM sales.custom_fields WHERE (sqlc.narg('object')::text IS NULL OR object = sqlc.narg('object')) ORDER BY object, sort_order, label;

-- name: UpsertCustomField :exec
INSERT INTO sales.custom_fields (id, object, key, label, field_type, options, is_required, show_in_list, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, field_type = EXCLUDED.field_type, options = EXCLUDED.options,
    is_required = EXCLUDED.is_required, show_in_list = EXCLUDED.show_in_list, sort_order = EXCLUDED.sort_order,
    is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteCustomField :exec
DELETE FROM sales.custom_fields WHERE id = $1;
