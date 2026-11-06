-- name: GetInvoice :one
SELECT sqlc.embed(i), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       coalesce(q.quote_number, '')::text AS quote_number, coalesce(t.label, '')::text AS term_label,
       (SELECT coalesce(sum(p.amount), 0) FROM sales.payments p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL)::numeric AS paid_total
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
LEFT JOIN sales.quotations q ON q.id = i.quotation_id
LEFT JOIN sales.quotation_terms t ON t.id = i.term_id
WHERE i.id = $1;

-- name: ListInvoices :many
SELECT sqlc.embed(i), d.title AS deal_title, coalesce(a.name, l.org_name, d.title)::text AS client_name,
       coalesce(q.quote_number, '')::text AS quote_number, coalesce(t.label, '')::text AS term_label,
       (SELECT coalesce(sum(p.amount), 0) FROM sales.payments p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL)::numeric AS paid_total
FROM sales.invoices i
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
LEFT JOIN sales.quotations q ON q.id = i.quotation_id
LEFT JOIN sales.quotation_terms t ON t.id = i.term_id
WHERE d.deleted_at IS NULL
  AND (sqlc.narg('deal')::text IS NULL OR i.deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('status')::text IS NULL OR i.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::text IS NULL OR i.invoice_number ILIKE '%' || sqlc.narg('search') || '%' OR i.label ILIKE '%' || sqlc.narg('search') || '%'
       OR d.title ILIKE '%' || sqlc.narg('search') || '%' OR coalesce(a.name, l.org_name, '') ILIKE '%' || sqlc.narg('search') || '%')
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR coalesce(i.sent_at, i.created_at) >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR coalesce(i.sent_at, i.created_at) < sqlc.narg('to_at'))
  AND (sqlc.narg('client')::text IS NULL OR EXISTS (SELECT 1 FROM sales.onboardings o WHERE o.deal_id = i.deal_id AND o.marketing_client_id = sqlc.narg('client')))
ORDER BY i.created_at DESC, i.id
LIMIT sqlc.arg('lim');

-- name: UpsertInvoice :exec
INSERT INTO sales.invoices (id, deal_id, quotation_id, term_id, invoice_number, label, amount, due_date, status, sent_at,
    issued_by, cancelled_at, cancelled_by, cancel_reason, created_by, updated_by, created_at, updated_at, note, faktur_pajak_key, faktur_pajak_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (id) DO UPDATE SET label = EXCLUDED.label, amount = EXCLUDED.amount, due_date = EXCLUDED.due_date,
    status = EXCLUDED.status, sent_at = EXCLUDED.sent_at, issued_by = EXCLUDED.issued_by,
    cancelled_at = EXCLUDED.cancelled_at, cancelled_by = EXCLUDED.cancelled_by, cancel_reason = EXCLUDED.cancel_reason,
    updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at, note = EXCLUDED.note,
    faktur_pajak_key = EXCLUDED.faktur_pajak_key, faktur_pajak_name = EXCLUDED.faktur_pajak_name;

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

-- name: InsertFollowup :execrows
INSERT INTO sales.invoice_followups (id, invoice_id, stage, phone, message)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (invoice_id, stage) DO NOTHING;

-- name: ListFollowups :many
SELECT sqlc.embed(f), i.invoice_number, i.deal_id, coalesce(a.name, l.org_name, d.title)::text AS client_name
FROM sales.invoice_followups f
JOIN sales.invoices i ON i.id = f.invoice_id
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE (sqlc.narg('status')::text IS NULL OR f.status = sqlc.narg('status'))
ORDER BY f.created_at DESC, f.id
LIMIT sqlc.arg('lim');

-- name: GetFollowup :one
SELECT * FROM sales.invoice_followups WHERE id = $1;

-- name: UpdateFollowup :exec
UPDATE sales.invoice_followups
SET phone = $2, message = $3, status = $4, last_error = $5, approved_by = $6, approved_at = $7, sent_at = $8, message_id = $9
WHERE id = $1;

-- name: InsertPaymentProof :exec
INSERT INTO sales.payment_proofs (id, invoice_id, amount, paid_on, payer_name, note, file_key, file_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetPaymentProof :one
SELECT sqlc.embed(pp), i.invoice_number, i.deal_id, coalesce(a.name, l.org_name, d.title)::text AS client_name
FROM sales.payment_proofs pp
JOIN sales.invoices i ON i.id = pp.invoice_id
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE pp.id = $1;

-- name: ListPaymentProofs :many
SELECT sqlc.embed(pp), i.invoice_number, i.deal_id, coalesce(a.name, l.org_name, d.title)::text AS client_name
FROM sales.payment_proofs pp
JOIN sales.invoices i ON i.id = pp.invoice_id
JOIN sales.deals d ON d.id = i.deal_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE (sqlc.narg('status')::text IS NULL OR pp.status = sqlc.narg('status'))
  AND (sqlc.narg('invoice_ids')::text[] IS NULL OR pp.invoice_id = ANY(sqlc.narg('invoice_ids')::text[]))
ORDER BY pp.created_at DESC
LIMIT sqlc.arg('lim');

-- name: DecidePaymentProof :execrows
UPDATE sales.payment_proofs
SET status = $2, reject_reason = $3, payment_id = $4, decided_by = $5, decided_at = $6
WHERE id = $1 AND status = 'PENDING';
