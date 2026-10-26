-- name: GetInvoice :one
SELECT sqlc.embed(i), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       (SELECT coalesce(sum(p.amount), 0) FROM sales.payments p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL)::numeric AS paid_total
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE i.id = $1;

-- name: ListInvoices :many
SELECT sqlc.embed(i), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       (SELECT coalesce(sum(p.amount), 0) FROM sales.payments p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL)::numeric AS paid_total
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE d.deleted_at IS NULL
  AND (sqlc.narg('deal')::text IS NULL OR i.deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('status')::text IS NULL OR i.status = sqlc.narg('status'))
ORDER BY i.created_at DESC, i.id
LIMIT sqlc.arg('lim');

-- name: UpsertInvoice :exec
INSERT INTO sales.invoices (id, deal_id, quotation_id, term_id, invoice_number, label, amount, due_date, status, sent_at,
    issued_by, cancelled_at, cancelled_by, cancel_reason, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, amount = EXCLUDED.amount, due_date = EXCLUDED.due_date,
    status = EXCLUDED.status, sent_at = EXCLUDED.sent_at, issued_by = EXCLUDED.issued_by,
    cancelled_at = EXCLUDED.cancelled_at, cancelled_by = EXCLUDED.cancelled_by, cancel_reason = EXCLUDED.cancel_reason,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: GetPayment :one
SELECT * FROM sales.payments WHERE id = $1 AND deleted_at IS NULL;

-- name: ListPayments :many
SELECT * FROM sales.payments WHERE deal_id = $1 AND deleted_at IS NULL ORDER BY paid_on, created_at;

-- name: InsertPayment :exec
INSERT INTO sales.payments (id, deal_id, invoice_id, amount, method, paid_on, note, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: DeletePayment :exec
UPDATE sales.payments SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: PaidTotal :one
SELECT coalesce(sum(amount), 0)::numeric FROM sales.payments WHERE deal_id = $1 AND deleted_at IS NULL;

-- name: InvoicePaidTotal :one
SELECT coalesce(sum(amount), 0)::numeric FROM sales.payments WHERE invoice_id = $1 AND deleted_at IS NULL;

-- name: ListTargets :many
SELECT * FROM sales.targets WHERE period_month = $1 ORDER BY user_id NULLS FIRST;

-- name: DeleteTargets :exec
DELETE FROM sales.targets WHERE period_month = $1;

-- name: InsertTarget :exec
INSERT INTO sales.targets (id, user_id, period_month, target_value, target_deals, updated_by) VALUES ($1, $2, $3, $4, $5, $6);
