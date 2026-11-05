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
SELECT a.id, a.kind, a.advance_no, a.advance_date, a.party_name, a.amount, a.cash_account_id, COALESCE(c.name, '')::text AS cash_account_name,
       a.advance_account_id, d.name AS advance_account_name, a.reference, a.notes, a.status, a.journal_entry_id, a.void_reason, a.created_by, a.source_payment_id,
       COALESCE((SELECT SUM(p.amount) FROM finance.advance_applications x
                   JOIN finance.payments p ON p.id = x.payment_id
                  WHERE x.advance_id = a.id AND p.status = 'POSTED' AND p.deleted_at IS NULL), 0)::numeric AS applied,
       COALESCE((SELECT SUM(r.amount) FROM finance.advance_refunds r WHERE r.advance_id = a.id AND r.status = 'POSTED'), 0)::numeric AS refunded
  FROM finance.advances a
  LEFT JOIN finance.accounts c ON c.id = a.cash_account_id
  JOIN finance.accounts d ON d.id = a.advance_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR a.id = sqlc.narg(id))
   AND (sqlc.narg(source_payment_id)::text IS NULL OR a.source_payment_id = sqlc.narg(source_payment_id))
   AND (sqlc.narg(sales_payment_id)::text IS NULL OR a.sales_payment_id = sqlc.narg(sales_payment_id))
   AND (sqlc.narg(kind)::text IS NULL OR a.kind = sqlc.narg(kind))
 ORDER BY a.advance_date DESC, a.advance_no DESC
 LIMIT 500;

-- name: LockAdvance :exec
SELECT id FROM finance.advances WHERE id = $1 FOR UPDATE;

-- name: InsertAdvance :exec
INSERT INTO finance.advances (id, kind, advance_no, advance_date, party_name, amount, cash_account_id, advance_account_id, reference, notes, status, journal_entry_id, created_by, source_payment_id, sales_payment_id)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(cash_account_id), $7, $8, $9, $10, sqlc.narg(journal_entry_id), sqlc.narg(by), sqlc.narg(source_payment_id), sqlc.narg(sales_payment_id));

-- name: SetAdvanceJournalByPayment :exec
UPDATE finance.advances SET journal_entry_id = $2 WHERE source_payment_id = $1;

-- name: InvoiceCredited :one
SELECT COALESCE(SUM(x.amount + COALESCE((SELECT SUM(j.amount) FROM finance.payment_adjustments j WHERE j.payment_id = p.id AND j.kind = 'EXCESS'), 0)), 0)::numeric
  FROM finance.payment_allocations x
  JOIN finance.payments p ON p.id = x.payment_id
 WHERE x.invoice_id = $1 AND p.method = 'credit_note' AND p.status = 'POSTED' AND p.deleted_at IS NULL;

-- name: VoidAdvance :exec
UPDATE finance.advances SET status = 'VOID', void_reason = $2 WHERE id = $1;

-- name: InsertAdvanceApplication :exec
INSERT INTO finance.advance_applications (payment_id, advance_id) VALUES ($1, $2);

-- name: DeletePostingLog :exec
DELETE FROM finance.posting_logs WHERE event_code = $1 AND document_id = $2;

-- name: PendingPostings :one
SELECT count(*)::int FROM finance.posting_logs
 WHERE status IN ('draft', 'skipped')
   AND (request->>'entryDate')::date >= sqlc.arg(date_from)::date
   AND (request->>'entryDate')::date <= sqlc.arg(date_to)::date;

-- name: SubledgerTotals :many
SELECT i.kind,
       COALESCE(SUM(i.total_amount - COALESCE(pa.allocated, 0)) FILTER (WHERE i.journal_entry_id IS NOT NULL), 0)::numeric AS outstanding,
       (count(*) FILTER (WHERE i.journal_entry_id IS NULL))::int AS unjournaled_count,
       COALESCE(SUM(i.total_amount) FILTER (WHERE i.journal_entry_id IS NULL), 0)::numeric AS unjournaled_amount
  FROM finance.invoices i
  LEFT JOIN LATERAL (
    SELECT SUM(a.amount) AS allocated FROM finance.payment_allocations a
      JOIN finance.payments p ON p.id = a.payment_id
     WHERE a.invoice_id = i.id AND p.status = 'POSTED' AND p.deleted_at IS NULL) pa ON true
 WHERE i.deleted_at IS NULL AND i.status = 'POSTED'
 GROUP BY i.kind;

-- name: ControlAdjustments :one
-- What sits on control accounts without an invoice: the latest opening balance
-- and posted recon journals, as debit minus credit.
WITH latest AS (
  SELECT MAX(entry_date) AS day FROM finance.journal_entries
   WHERE deleted_at IS NULL AND status = 'POSTED' AND entry_type = 'OPENING'
)
SELECT COALESCE(SUM(CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE -l.amount END) FILTER (WHERE e.entry_type = 'OPENING' AND e.entry_date = latest.day), 0)::numeric AS opening,
       COALESCE(SUM(CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE -l.amount END) FILTER (WHERE e.is_recon AND e.entry_type <> 'OPENING'), 0)::numeric AS recon
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
  CROSS JOIN latest
 WHERE e.deleted_at IS NULL AND e.status = 'POSTED' AND l.account_id = ANY(sqlc.arg(account_ids)::text[]);

-- name: InsertPaymentAdjustment :exec
INSERT INTO finance.payment_adjustments (id, payment_id, kind, account_id, tax_code, tax_name, tax_rate, report_code, tax_base, amount, memo, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: PaymentAdjustments :many
SELECT x.payment_id, x.kind, x.account_id, a.name AS account_name, x.tax_code, x.tax_name, x.tax_rate, x.report_code, x.tax_base, x.amount, x.memo
  FROM finance.payment_adjustments x
  JOIN finance.accounts a ON a.id = x.account_id
 WHERE x.payment_id = ANY(sqlc.arg(payment_ids)::text[])
 ORDER BY x.payment_id, x.sort_order;

-- name: InsertAdvanceRefund :exec
INSERT INTO finance.advance_refunds (id, advance_id, refund_no, refund_date, amount, cash_account_id, reason, status, journal_entry_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, sqlc.narg(by));

-- name: ListAdvanceRefunds :many
SELECT r.id, r.advance_id, r.refund_no, r.refund_date, r.amount, r.cash_account_id, c.name AS cash_account_name, r.reason, r.status, r.journal_entry_id, r.void_reason, r.created_by
  FROM finance.advance_refunds r
  JOIN finance.accounts c ON c.id = r.cash_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR r.id = sqlc.narg(id))
   AND (sqlc.narg(advance_ids)::text[] IS NULL OR r.advance_id = ANY(sqlc.narg(advance_ids)::text[]))
 ORDER BY r.refund_date, r.refund_no;

-- name: VoidAdvanceRefund :exec
UPDATE finance.advance_refunds SET status = 'VOID', void_reason = $2 WHERE id = $1;

-- name: ListDeferrals :many
SELECT d.id, d.kind, d.name, d.party_name, d.reference, d.deferred_account_id, x.name AS deferred_account_name, d.target_account_id, t.name AS target_account_name,
       d.tag_id, d.start_date, d.months, d.posted_months, d.amount, d.is_active, d.last_entry_id, d.last_error, d.created_by
  FROM finance.deferrals d
  JOIN finance.accounts x ON x.id = d.deferred_account_id
  JOIN finance.accounts t ON t.id = d.target_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR d.id = sqlc.narg(id))
   AND (sqlc.arg(active_only)::bool = false OR d.is_active)
 ORDER BY d.start_date DESC, d.name
 LIMIT 500;

-- name: InsertDeferral :exec
INSERT INTO finance.deferrals (id, kind, name, party_name, reference, deferred_account_id, target_account_id, tag_id, start_date, months, amount, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(tag_id), $8, $9, $10, sqlc.narg(by));

-- name: UpdateDeferralRun :exec
UPDATE finance.deferrals SET posted_months = $2, last_entry_id = sqlc.narg(last_entry_id), last_error = $3, is_active = $4 WHERE id = $1;

-- name: SetDeferralActive :exec
UPDATE finance.deferrals SET is_active = $2 WHERE id = $1;

-- name: DeleteDeferral :exec
DELETE FROM finance.deferrals WHERE id = $1;

-- name: ListTaxRemittances :many
SELECT r.id, r.remit_no, r.period, r.remit_date, r.account_id, a.name AS account_name, r.cash_account_id, c.name AS cash_account_name, r.amount, r.ntpn, r.notes, r.status, r.journal_entry_id, r.void_reason, r.created_by
  FROM finance.tax_remittances r
  JOIN finance.accounts a ON a.id = r.account_id
  JOIN finance.accounts c ON c.id = r.cash_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR r.id = sqlc.narg(id))
 ORDER BY r.remit_date DESC, r.remit_no DESC
 LIMIT 500;

-- name: InsertTaxRemittance :exec
INSERT INTO finance.tax_remittances (id, remit_no, period, remit_date, account_id, cash_account_id, amount, ntpn, notes, status, journal_entry_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, sqlc.narg(by));

-- name: VoidTaxRemittance :exec
UPDATE finance.tax_remittances SET status = 'VOID', void_reason = $2 WHERE id = $1;

-- name: SeparateApprover :one
SELECT COALESCE((SELECT separate_approver FROM finance.control_settings), false)::bool;

-- name: SetSeparateApprover :exec
INSERT INTO finance.control_settings (id, separate_approver, updated_by) VALUES (true, $1, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET separate_approver = EXCLUDED.separate_approver, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: InsertAuditEntry :exec
INSERT INTO finance.audit_trail (id, actor_id, actor_name, action, doc_id, doc_no, result, message, request, response)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: ListAuditEntries :many
SELECT id, at, actor_id, actor_name, action, doc_id, doc_no, result, message, request, response
  FROM finance.audit_trail
 WHERE (sqlc.narg(search)::text IS NULL OR actor_name ILIKE '%' || sqlc.narg(search) || '%' OR action ILIKE '%' || sqlc.narg(search) || '%' OR doc_no ILIKE '%' || sqlc.narg(search) || '%' OR doc_id = sqlc.narg(search))
   AND (sqlc.narg(result)::text IS NULL OR result = sqlc.narg(result))
   AND (sqlc.narg(date_from)::date IS NULL OR at >= sqlc.narg(date_from)::date)
   AND (sqlc.narg(date_to)::date IS NULL OR at < sqlc.narg(date_to)::date + 1)
 ORDER BY at DESC, id DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountAuditEntries :one
SELECT count(*)::int
  FROM finance.audit_trail
 WHERE (sqlc.narg(search)::text IS NULL OR actor_name ILIKE '%' || sqlc.narg(search) || '%' OR action ILIKE '%' || sqlc.narg(search) || '%' OR doc_no ILIKE '%' || sqlc.narg(search) || '%' OR doc_id = sqlc.narg(search))
   AND (sqlc.narg(result)::text IS NULL OR result = sqlc.narg(result))
   AND (sqlc.narg(date_from)::date IS NULL OR at >= sqlc.narg(date_from)::date)
   AND (sqlc.narg(date_to)::date IS NULL OR at < sqlc.narg(date_to)::date + 1);
