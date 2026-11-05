-- name: InsertBankStatement :exec
INSERT INTO finance.bank_statements (id, account_id, filename, created_by) VALUES ($1, $2, $3, sqlc.narg(by));

-- name: SetBankStatementCount :exec
UPDATE finance.bank_statements SET line_count = $2 WHERE id = $1;

-- name: InsertStatementLine :execrows
INSERT INTO finance.bank_statement_lines (id, statement_id, account_id, line_date, description, sender_name, amount, dedupe_key)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (account_id, dedupe_key) DO NOTHING;

-- name: ListStatementLines :many
SELECT sqlc.embed(l), coalesce(e.entry_no, '')::text AS matched_entry_no, count(*) OVER ()::int AS total
  FROM finance.bank_statement_lines l
  LEFT JOIN finance.journal_entries e ON e.id = l.matched_entry_id
 WHERE l.account_id = $1
   AND (sqlc.narg(status)::text IS NULL OR l.status = sqlc.narg(status))
   AND (sqlc.narg(date_from)::date IS NULL OR l.line_date >= sqlc.narg(date_from))
   AND (sqlc.narg(date_to)::date IS NULL OR l.line_date <= sqlc.narg(date_to))
 ORDER BY l.line_date DESC, l.id
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: PendingStatementLines :many
SELECT * FROM finance.bank_statement_lines
 WHERE account_id = $1 AND status IN ('UNMATCHED', 'SUGGESTED')
 ORDER BY line_date, id;

-- name: GetStatementLine :one
SELECT * FROM finance.bank_statement_lines WHERE id = $1;

-- name: UpdateStatementLine :exec
UPDATE finance.bank_statement_lines
   SET status = $2, suggested_kind = $3, suggested_ref = $4, suggested_label = $5, suggested_score = $6, suggested_reason = $7,
       matched_line_id = $8, matched_entry_id = $9, note = $10, verified_by = sqlc.narg(verified_by), verified_at = sqlc.narg(verified_at)
 WHERE id = $1;

-- name: StatementLineCounts :many
SELECT status, count(*)::int AS n FROM finance.bank_statement_lines WHERE account_id = $1 GROUP BY status;

-- name: OpenJournalLines :many
SELECT l.id, e.id AS entry_id, e.entry_no, e.entry_date, e.entry_type, e.description, l.memo, l.entry_side, l.amount
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
 WHERE l.account_id = $1 AND l.entry_side = $2 AND l.amount = $3
   AND e.status = 'POSTED' AND e.deleted_at IS NULL
   AND e.entry_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
   AND NOT EXISTS (SELECT 1 FROM finance.bank_statement_lines b WHERE b.matched_line_id = l.id)
 ORDER BY e.entry_date, e.entry_no;

-- name: GetJournalLineForMatch :one
SELECT l.id, e.id AS entry_id, e.entry_no, e.entry_date, e.entry_type, e.description, l.memo, l.entry_side, l.amount, l.account_id, e.status
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
 WHERE l.id = $1 AND e.deleted_at IS NULL;

-- name: JournalLineOnAccount :one
SELECT id FROM finance.journal_entry_lines
 WHERE entry_id = $1 AND account_id = $2 AND entry_side = $3 AND amount = $4
 ORDER BY sort_order LIMIT 1;

-- name: ListRecurringJournals :many
SELECT * FROM finance.recurring_journals ORDER BY is_active DESC, next_run, name;

-- name: GetRecurringJournal :one
SELECT * FROM finance.recurring_journals WHERE id = $1;

-- name: DueRecurringJournals :many
SELECT * FROM finance.recurring_journals WHERE is_active AND next_run <= sqlc.arg(today)::date ORDER BY next_run, id;

-- name: InsertRecurringJournal :exec
INSERT INTO finance.recurring_journals (id, name, description, frequency, anchor_day, next_run, end_date, lines, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(end_date), $7, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateRecurringRun :exec
UPDATE finance.recurring_journals
   SET next_run = $2, is_active = $3, last_entry_id = sqlc.narg(last_entry_id), last_error = $4, updated_at = now()
 WHERE id = $1;

-- name: SetRecurringActive :exec
UPDATE finance.recurring_journals SET is_active = $2, updated_by = sqlc.narg(by), updated_at = now() WHERE id = $1;

-- name: DeleteRecurringJournal :exec
DELETE FROM finance.recurring_journals WHERE id = $1;

-- name: ListFixedAssets :many
SELECT f.*, x.name AS expense_account_name, a.name AS accum_account_name
  FROM finance.fixed_assets f
  JOIN finance.accounts x ON x.id = f.expense_account_id
  JOIN finance.accounts a ON a.id = f.accum_account_id
 WHERE (sqlc.narg(id)::text IS NULL OR f.id = sqlc.narg(id))
   AND (sqlc.narg(active)::boolean IS NULL OR f.is_active = sqlc.narg(active))
 ORDER BY f.is_active DESC, f.acquired_on DESC, f.name;

-- name: InsertFixedAsset :exec
INSERT INTO finance.fixed_assets (id, name, acquired_on, cost, salvage, life_months, expense_account_id, accum_account_id, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateFixedAssetRun :exec
UPDATE finance.fixed_assets
   SET depreciated_months = $2, is_active = $3, last_entry_id = sqlc.narg(last_entry_id), last_error = $4, updated_at = now()
 WHERE id = $1;

-- name: SetFixedAssetActive :exec
UPDATE finance.fixed_assets SET is_active = $2, updated_by = sqlc.narg(by), updated_at = now() WHERE id = $1;

-- name: DeleteFixedAsset :exec
DELETE FROM finance.fixed_assets WHERE id = $1;

-- name: ReceiptPrintsOutside :many
SELECT r.id, r.claim_id, c.claim_no, r.filename, r.content_hash, r.image_hash
  FROM finance.reimbursement_receipts r
  JOIN finance.reimbursement_claims c ON c.id = r.claim_id
 WHERE r.claim_id <> $1 AND c.status NOT IN ('CANCELLED', 'REJECTED')
   AND (r.content_hash <> '' OR r.image_hash IS NOT NULL);

-- name: ListBudgets :many
SELECT period_month, account_id, amount FROM finance.budgets
 WHERE period_month BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
 ORDER BY period_month, account_id;

-- name: UpsertBudget :exec
INSERT INTO finance.budgets (period_month, account_id, amount, updated_by) VALUES ($1, $2, $3, sqlc.narg(by))
ON CONFLICT (period_month, account_id) DO UPDATE SET amount = EXCLUDED.amount, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: DeleteBudget :exec
DELETE FROM finance.budgets WHERE period_month = $1 AND account_id = $2;

-- name: ClientEconomics :many
SELECT i.party_name, count(*)::int AS invoice_count, sum(i.total_amount)::numeric AS invoiced,
       coalesce(sum(pay.paid), 0)::numeric AS collected,
       coalesce(sum(pay.paid_days), 0)::float8 AS paid_days
  FROM finance.invoices i
  LEFT JOIN LATERAL (
    SELECT sum(pa.amount) AS paid, sum(pa.amount * (p.payment_date - i.invoice_date)) AS paid_days
      FROM finance.payment_allocations pa
      JOIN finance.payments p ON p.id = pa.payment_id
     WHERE pa.invoice_id = i.id AND p.deleted_at IS NULL AND p.status = 'POSTED'
  ) pay ON true
 WHERE i.kind = 'AR' AND i.status = 'POSTED' AND i.deleted_at IS NULL
   AND i.invoice_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
 GROUP BY i.party_name
 ORDER BY invoiced DESC, i.party_name;
