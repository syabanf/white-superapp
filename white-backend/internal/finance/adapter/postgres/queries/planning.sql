-- name: ListProjectTags :many
SELECT * FROM finance.project_tags
 WHERE (NOT sqlc.arg(active_only)::boolean OR is_active)
 ORDER BY is_active DESC, client_name, name;

-- name: GetProjectTag :one
SELECT * FROM finance.project_tags WHERE id = $1;

-- name: UpsertProjectTag :exec
INSERT INTO finance.project_tags (id, name, client_name, is_active, notes, updated_by)
VALUES ($1, $2, $3, $4, $5, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, client_name = EXCLUDED.client_name, is_active = EXCLUDED.is_active,
  notes = EXCLUDED.notes, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: EnsureDealTag :one
INSERT INTO finance.project_tags (id, name, client_name, deal_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (deal_id) DO UPDATE SET updated_at = finance.project_tags.updated_at
RETURNING id;

-- name: ProjectTagInUse :one
SELECT EXISTS (SELECT 1 FROM finance.journal_entries j WHERE j.tag_id = sqlc.arg(id)::text AND j.deleted_at IS NULL)
    OR EXISTS (SELECT 1 FROM finance.invoices i WHERE i.tag_id = sqlc.arg(id)::text AND i.deleted_at IS NULL)
    OR EXISTS (SELECT 1 FROM finance.recurring_journals r WHERE r.tag_id = sqlc.arg(id)::text)
    OR EXISTS (SELECT 1 FROM finance.recurring_invoices v WHERE v.tag_id = sqlc.arg(id)::text);

-- name: DeleteProjectTag :exec
DELETE FROM finance.project_tags WHERE id = $1;

-- name: ProjectProfit :many
SELECT COALESCE(e.tag_id, '')::text AS tag_id, COALESCE(t.name, '')::text AS tag_name, COALESCE(t.client_name, '')::text AS client_name,
       COALESCE(SUM(CASE WHEN at.code IN ('REVENUE', 'OTHER_INCOME') THEN CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE -l.amount END END), 0)::numeric AS revenue,
       COALESCE(SUM(CASE WHEN at.code IN ('COGS', 'EXPENSE', 'OTHER_EXPENSE') THEN CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE -l.amount END END), 0)::numeric AS cost
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
  JOIN finance.accounts a ON a.id = l.account_id
  JOIN finance.account_types at ON at.id = a.account_type_id
  LEFT JOIN finance.project_tags t ON t.id = e.tag_id
 WHERE e.deleted_at IS NULL AND e.status = 'POSTED' AND e.entry_type <> 'OPENING'
   AND e.entry_date >= sqlc.arg(date_from)::date AND e.entry_date <= sqlc.arg(date_to)::date
   AND at.code IN ('REVENUE', 'OTHER_INCOME', 'COGS', 'EXPENSE', 'OTHER_EXPENSE')
   AND (sqlc.narg(client_name)::text IS NULL OR lower(t.client_name) = lower(sqlc.narg(client_name)))
 GROUP BY 1, 2, 3
 ORDER BY revenue DESC, tag_name;

-- name: ListRecurringInvoices :many
SELECT * FROM finance.recurring_invoices ORDER BY is_active DESC, next_run, name;

-- name: DueRecurringInvoices :many
SELECT * FROM finance.recurring_invoices WHERE is_active AND next_run <= sqlc.arg(today)::date ORDER BY next_run, id;

-- name: GetRecurringInvoice :one
SELECT * FROM finance.recurring_invoices WHERE id = $1;

-- name: UpsertRecurringInvoice :exec
INSERT INTO finance.recurring_invoices (id, kind, name, party_name, client_id, account_id, tag_id, description, subtotal, tax_amount, tax_ids,
  due_days, frequency, anchor_day, next_run, end_date, auto_post, is_active, last_error, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, sqlc.narg(account_id), sqlc.narg(tag_id), $6, $7, $8, $9, $10, $11, $12, $13, sqlc.narg(end_date), $14, $15, $16, sqlc.narg(by), sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET kind = EXCLUDED.kind, name = EXCLUDED.name, party_name = EXCLUDED.party_name, client_id = EXCLUDED.client_id,
  account_id = EXCLUDED.account_id, tag_id = EXCLUDED.tag_id, description = EXCLUDED.description, subtotal = EXCLUDED.subtotal, tax_amount = EXCLUDED.tax_amount,
  tax_ids = EXCLUDED.tax_ids, due_days = EXCLUDED.due_days, frequency = EXCLUDED.frequency, anchor_day = EXCLUDED.anchor_day, next_run = EXCLUDED.next_run,
  end_date = EXCLUDED.end_date, auto_post = EXCLUDED.auto_post, last_error = EXCLUDED.last_error, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: UpdateRecurringInvoiceRun :exec
UPDATE finance.recurring_invoices
   SET next_run = $2, is_active = $3, last_invoice_id = sqlc.narg(last_invoice_id), last_error = $4, updated_at = now()
 WHERE id = $1;

-- name: SetRecurringInvoiceActive :exec
UPDATE finance.recurring_invoices SET is_active = $2, updated_by = sqlc.narg(by), updated_at = now() WHERE id = $1;

-- name: DeleteRecurringInvoice :exec
DELETE FROM finance.recurring_invoices WHERE id = $1;

-- name: ListAdvances :many
SELECT a.id, a.kind, a.advance_no, a.advance_date, a.party_name, a.amount, a.cash_account_id, c.name AS cash_account_name,
       a.advance_account_id, d.name AS advance_account_name, a.reference, a.notes, a.status, a.journal_entry_id, a.void_reason, a.created_by,
       COALESCE((SELECT SUM(p.amount) FROM finance.advance_applications x
                   JOIN finance.payments p ON p.id = x.payment_id
                  WHERE x.advance_id = a.id AND p.status = 'POSTED' AND p.deleted_at IS NULL), 0)::numeric AS applied
  FROM finance.advances a
  JOIN finance.accounts c ON c.id = a.cash_account_id
  JOIN finance.accounts d ON d.id = a.advance_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR a.id = sqlc.narg(id))
   AND (sqlc.narg(kind)::text IS NULL OR a.kind = sqlc.narg(kind))
 ORDER BY a.advance_date DESC, a.advance_no DESC
 LIMIT 500;

-- name: LockAdvance :exec
SELECT id FROM finance.advances WHERE id = $1 FOR UPDATE;

-- name: InsertAdvance :exec
INSERT INTO finance.advances (id, kind, advance_no, advance_date, party_name, amount, cash_account_id, advance_account_id, reference, notes, status, journal_entry_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, sqlc.narg(journal_entry_id), sqlc.narg(by));

-- name: VoidAdvance :exec
UPDATE finance.advances SET status = 'VOID', void_reason = $2 WHERE id = $1;

-- name: InsertAdvanceApplication :exec
INSERT INTO finance.advance_applications (payment_id, advance_id) VALUES ($1, $2);
