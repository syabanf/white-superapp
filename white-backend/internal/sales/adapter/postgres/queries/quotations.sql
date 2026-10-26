-- name: ListCatalog :many
SELECT * FROM sales.service_catalog WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY sort_order, name;

-- name: GetCatalogItem :one
SELECT * FROM sales.service_catalog WHERE id = $1;

-- name: ListCatalogTiers :many
SELECT * FROM sales.catalog_tiers ORDER BY catalog_id, CASE tier WHEN 'basic' THEN 1 WHEN 'standard' THEN 2 ELSE 3 END;

-- name: UpsertCatalogItem :exec
INSERT INTO sales.service_catalog (id, code, name, service_line, unit, default_price, description, is_retainer,
    is_default_for_line, is_active, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, service_line = EXCLUDED.service_line,
    unit = EXCLUDED.unit, default_price = EXCLUDED.default_price, description = EXCLUDED.description,
    is_retainer = EXCLUDED.is_retainer, is_default_for_line = EXCLUDED.is_default_for_line,
    is_active = EXCLUDED.is_active, sort_order = EXCLUDED.sort_order, updated_at = now();

-- name: DeleteCatalogTiers :exec
DELETE FROM sales.catalog_tiers WHERE catalog_id = $1;

-- name: InsertCatalogTier :exec
INSERT INTO sales.catalog_tiers (catalog_id, tier, price, description) VALUES ($1, $2, $3, $4);

-- name: ListRetainerDiscounts :many
SELECT * FROM sales.retainer_discounts ORDER BY min_months;

-- name: DeleteRetainerDiscounts :exec
DELETE FROM sales.retainer_discounts;

-- name: InsertRetainerDiscount :exec
INSERT INTO sales.retainer_discounts (min_months, discount_percent) VALUES ($1, $2);

-- name: GetQuotation :one
SELECT sqlc.embed(q), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       coalesce(d.owner_user_id, '')::text AS owner_user_id
FROM sales.quotations q
JOIN sales.deals d ON d.id = q.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE q.id = $1 AND q.deleted_at IS NULL;

-- name: ListQuotations :many
SELECT sqlc.embed(q), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       coalesce(d.owner_user_id, '')::text AS owner_user_id
FROM sales.quotations q
JOIN sales.deals d ON d.id = q.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE q.deleted_at IS NULL AND d.deleted_at IS NULL
  AND (sqlc.narg('deal')::text IS NULL OR q.deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('status')::text IS NULL OR q.status = sqlc.narg('status'))
ORDER BY q.created_at DESC, q.version DESC
LIMIT sqlc.arg('lim');

-- name: ListQuotationItems :many
SELECT * FROM sales.quotation_items WHERE quotation_id = ANY(sqlc.arg('ids')::text[]) ORDER BY quotation_id, sort_order;

-- name: ListQuotationTerms :many
SELECT * FROM sales.quotation_terms WHERE quotation_id = ANY(sqlc.arg('ids')::text[]) ORDER BY quotation_id, sort_order;

-- name: UpsertQuotation :exec
INSERT INTO sales.quotations (id, deal_id, quote_number, version, parent_quotation_id, status, use_ppn, ppn_percent,
    subtotal, discount_percent, discount_nominal, ppn_nominal, total, notes, valid_until, approval_status,
    approval_request_id, generated_by, sent_at, decided_at, superseded_at, expiry_reminded_at,
    created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, use_ppn = EXCLUDED.use_ppn, ppn_percent = EXCLUDED.ppn_percent,
    subtotal = EXCLUDED.subtotal, discount_percent = EXCLUDED.discount_percent, discount_nominal = EXCLUDED.discount_nominal,
    ppn_nominal = EXCLUDED.ppn_nominal, total = EXCLUDED.total, notes = EXCLUDED.notes, valid_until = EXCLUDED.valid_until,
    approval_status = EXCLUDED.approval_status, approval_request_id = EXCLUDED.approval_request_id,
    sent_at = EXCLUDED.sent_at, decided_at = EXCLUDED.decided_at, superseded_at = EXCLUDED.superseded_at,
    expiry_reminded_at = EXCLUDED.expiry_reminded_at, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteQuotationItems :exec
DELETE FROM sales.quotation_items WHERE quotation_id = $1;

-- name: DeleteQuotationTerms :exec
DELETE FROM sales.quotation_terms WHERE quotation_id = $1;

-- name: InsertQuotationItem :exec
INSERT INTO sales.quotation_items (id, quotation_id, item_type, catalog_id, description, scope, unit, qty, unit_price, line_total, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: InsertQuotationTerm :exec
INSERT INTO sales.quotation_terms (id, quotation_id, label, percent, due_date, sort_order)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: SetQuotationApproval :exec
UPDATE sales.quotations SET approval_status = $2, approval_request_id = $3, updated_at = now() WHERE id = $1;

-- name: DeleteQuotation :exec
UPDATE sales.quotations SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: LatestQuotationID :one
SELECT id FROM sales.quotations WHERE deal_id = $1 AND deleted_at IS NULL AND status <> 'superseded'
ORDER BY (status = 'diterima') DESC, created_at DESC, version DESC LIMIT 1;

-- name: ExpiringQuotationIDs :many
SELECT q.id FROM sales.quotations q JOIN sales.deals d ON d.id = q.deal_id
WHERE q.deleted_at IS NULL AND d.deleted_at IS NULL AND q.status = 'terkirim' AND q.expiry_reminded_at IS NULL
  AND q.valid_until IS NOT NULL AND q.valid_until <= $1;

-- name: MarkQuotationReminded :exec
UPDATE sales.quotations SET expiry_reminded_at = $2 WHERE id = $1;

-- name: ListApprovalRules :many
SELECT * FROM sales.approval_rules WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY level, min_discount_percent;

-- name: UpsertApprovalRule :exec
INSERT INTO sales.approval_rules (id, name, level, min_discount_percent, approver_role, approver_user_id, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, level = EXCLUDED.level, min_discount_percent = EXCLUDED.min_discount_percent,
    approver_role = EXCLUDED.approver_role, approver_user_id = EXCLUDED.approver_user_id, is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteApprovalRule :exec
DELETE FROM sales.approval_rules WHERE id = $1;

-- name: InsertApprovalRequest :exec
INSERT INTO sales.approval_requests (id, object, subject_id, status, current_level, discount_percent, amount, note, requested_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: InsertApprovalStep :exec
INSERT INTO sales.approval_steps (id, request_id, level, approver_role, approver_user_id, status)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetApprovalRequest :one
SELECT r.*, coalesce(q.quote_number, '')::text AS quote_number, coalesce(q.deal_id, '')::text AS deal_id,
       coalesce(d.title, '')::text AS deal_title, coalesce(a.name, l.org_name, d.title, '')::text AS client_name
FROM sales.approval_requests r
LEFT JOIN sales.quotations q ON q.id = r.subject_id
LEFT JOIN sales.deals d ON d.id = q.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE r.id = $1;

-- name: LockApprovalRequest :exec
SELECT id FROM sales.approval_requests WHERE id = $1 FOR UPDATE;

-- name: ListApprovalRequests :many
SELECT r.*, coalesce(q.quote_number, '')::text AS quote_number, coalesce(q.deal_id, '')::text AS deal_id,
       coalesce(d.title, '')::text AS deal_title, coalesce(a.name, l.org_name, d.title, '')::text AS client_name
FROM sales.approval_requests r
JOIN sales.quotations q ON q.id = r.subject_id AND q.deleted_at IS NULL
JOIN sales.deals d ON d.id = q.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE (sqlc.narg('status')::text IS NULL OR r.status = sqlc.narg('status'))
ORDER BY (r.status = 'pending') DESC, r.created_at DESC, r.id
LIMIT sqlc.arg('lim');

-- name: ListApprovalSteps :many
SELECT * FROM sales.approval_steps WHERE request_id = ANY(sqlc.arg('ids')::text[]) ORDER BY request_id, level;

-- name: UpdateApprovalRequest :exec
UPDATE sales.approval_requests SET status = $2, current_level = $3, resolved_by = $4, resolved_at = $5 WHERE id = $1;

-- name: UpdateApprovalStep :exec
UPDATE sales.approval_steps SET status = $2, decided_by = $3, decided_at = $4, comment = $5 WHERE id = $1;

-- name: CancelPendingApprovals :exec
UPDATE sales.approval_requests SET status = 'cancelled', resolved_at = $2 WHERE subject_id = $1 AND status = 'pending';
