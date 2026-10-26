-- name: ListPolicies :many
SELECT p.category, p.label, p.monthly_limit, p.per_claim_limit, p.expense_account_id, p.auto_approve, p.is_active
  FROM finance.reimbursement_policies p
 ORDER BY CASE p.category WHEN 'TRANSPORT' THEN 1 WHEN 'MAKAN' THEN 2 WHEN 'AKOMODASI' THEN 3 WHEN 'OPERASIONAL_PROYEK' THEN 4 ELSE 5 END;

-- name: GetPolicy :one
SELECT p.category, p.label, p.monthly_limit, p.per_claim_limit, p.expense_account_id, p.auto_approve, p.is_active
  FROM finance.reimbursement_policies p
 WHERE p.category = $1;

-- name: UpsertPolicy :exec
INSERT INTO finance.reimbursement_policies (category, label, monthly_limit, per_claim_limit, expense_account_id, auto_approve, is_active, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(by))
ON CONFLICT (category) DO UPDATE
   SET label = EXCLUDED.label, monthly_limit = EXCLUDED.monthly_limit, per_claim_limit = EXCLUDED.per_claim_limit,
       expense_account_id = EXCLUDED.expense_account_id, auto_approve = EXCLUDED.auto_approve, is_active = EXCLUDED.is_active,
       updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: InsertClaim :exec
INSERT INTO finance.reimbursement_claims (
  id, claim_no, employee_id, employee_user_id, employee_name, department_name, manager_employee_id, manager_user_id,
  category, transaction_date, amount, description, client_id, project, status, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateClaim :exec
UPDATE finance.reimbursement_claims
   SET employee_name = $2, department_name = $3, manager_employee_id = $4, manager_user_id = $5,
       category = $6, transaction_date = $7, amount = $8, description = $9, client_id = $10, project = $11,
       status = $12, over_limit = $13, limit_note = $14, submitted_at = $15,
       manager_approved_at = $16, manager_approved_by = $17, finance_approved_at = $18, finance_approved_by = $19,
       rejected_at = $20, rejected_by = $21, rejected_stage = $22, reject_reason = $23, cancelled_at = $24,
       paid_at = $25, paid_by = $26, payment_batch_id = $27, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1;

-- name: GetClaim :one
SELECT * FROM finance.reimbursement_claims WHERE id = $1;

-- name: ListClaims :many
SELECT c.*, count(*) OVER ()::int AS total, (SUM(c.amount) OVER ())::numeric AS total_amount
  FROM finance.reimbursement_claims c
 WHERE (sqlc.narg(statuses)::text[] IS NULL OR c.status = ANY(sqlc.narg(statuses)::text[]))
   AND (sqlc.narg(category)::text IS NULL OR c.category = sqlc.narg(category))
   AND (sqlc.narg(employee_id)::text IS NULL OR c.employee_id = sqlc.narg(employee_id))
   AND (sqlc.narg(employee_user_id)::text IS NULL OR c.employee_user_id = sqlc.narg(employee_user_id))
   AND (sqlc.narg(manager_user_id)::text IS NULL OR c.manager_user_id = sqlc.narg(manager_user_id))
   AND (sqlc.narg(department)::text IS NULL OR c.department_name = sqlc.narg(department))
   AND (sqlc.narg(month_from)::date IS NULL OR c.transaction_date >= sqlc.narg(month_from))
   AND (sqlc.narg(month_to)::date IS NULL OR c.transaction_date <= sqlc.narg(month_to))
   AND (sqlc.narg(submitted_before)::timestamptz IS NULL OR c.submitted_at < sqlc.narg(submitted_before))
   AND (sqlc.narg(reminded_before)::timestamptz IS NULL OR c.last_reminded_at IS NULL OR c.last_reminded_at < sqlc.narg(reminded_before))
   AND (sqlc.narg(search)::text IS NULL OR c.claim_no ILIKE '%' || sqlc.narg(search) || '%'
        OR c.employee_name ILIKE '%' || sqlc.narg(search) || '%' OR c.description ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY c.transaction_date DESC, c.created_at DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: MonthToDate :one
SELECT COALESCE(SUM(amount), 0)::numeric
  FROM finance.reimbursement_claims
 WHERE employee_id = $1 AND category = $2 AND id <> $3
   AND status IN ('SUBMITTED', 'APPROVED_MANAGER', 'APPROVED_FINANCE', 'PAID')
   AND transaction_date >= sqlc.arg(date_from)::date AND transaction_date <= sqlc.arg(date_to)::date;

-- name: ClaimReceipts :many
SELECT * FROM finance.reimbursement_receipts WHERE claim_id = ANY(sqlc.arg(claim_ids)::text[]) ORDER BY created_at;

-- name: ClaimEvents :many
SELECT * FROM finance.reimbursement_events WHERE claim_id = $1 ORDER BY created_at, id;

-- name: InsertReceipt :exec
INSERT INTO finance.reimbursement_receipts (id, claim_id, storage_key, filename, content_type, size_bytes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteReceipt :exec
DELETE FROM finance.reimbursement_receipts WHERE id = $1;

-- name: GetReceipt :one
SELECT * FROM finance.reimbursement_receipts WHERE id = $1;

-- name: InsertClaimEvent :exec
INSERT INTO finance.reimbursement_events (id, claim_id, action, actor_id, actor_name, note) VALUES ($1, $2, $3, $4, $5, $6);

-- name: MarkReminded :exec
UPDATE finance.reimbursement_claims SET last_reminded_at = sqlc.arg(at) WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: InsertBatch :exec
INSERT INTO finance.reimbursement_batches (id, batch_no, paid_at, cash_account_id, total_amount, claim_count, notes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateBatchJournal :exec
UPDATE finance.reimbursement_batches SET journal_entry_id = $2, journal_status = $3, journal_reason = $4 WHERE id = $1;

-- name: ListBatches :many
SELECT b.*, COALESCE(a.name, '')::text AS cash_account_name
  FROM finance.reimbursement_batches b
  LEFT JOIN finance.accounts a ON a.id = b.cash_account_id
 ORDER BY b.created_at DESC
 LIMIT $1;

-- name: MonthlySummary :many
SELECT c.employee_id, max(c.employee_name)::text AS employee_name, max(c.department_name)::text AS department_name, c.category,
       count(*)::int AS claim_count,
       COALESCE(SUM(c.amount) FILTER (WHERE c.status IN ('SUBMITTED', 'APPROVED_MANAGER', 'APPROVED_FINANCE', 'PAID')), 0)::numeric AS submitted_amount,
       COALESCE(SUM(c.amount) FILTER (WHERE c.status IN ('APPROVED_FINANCE', 'PAID')), 0)::numeric AS approved_amount,
       COALESCE(SUM(c.amount) FILTER (WHERE c.status = 'PAID'), 0)::numeric AS paid_amount
  FROM finance.reimbursement_claims c
 WHERE c.transaction_date >= sqlc.arg(date_from)::date AND c.transaction_date <= sqlc.arg(date_to)::date
   AND c.status NOT IN ('DRAFT', 'CANCELLED')
   AND (sqlc.narg(employee_id)::text IS NULL OR c.employee_id = sqlc.narg(employee_id))
 GROUP BY c.employee_id, c.category
 ORDER BY employee_name, c.category;

-- name: DraftClaimKeysOfUser :many
SELECT r.storage_key
  FROM finance.reimbursement_receipts r
  JOIN finance.reimbursement_claims c ON c.id = r.claim_id
 WHERE c.employee_user_id = $1 AND c.status = 'DRAFT';

-- name: DeleteDraftClaimsOfUser :exec
DELETE FROM finance.reimbursement_claims WHERE employee_user_id = $1 AND status = 'DRAFT';

-- name: ClearManagerUser :exec
UPDATE finance.reimbursement_claims SET manager_user_id = NULL WHERE manager_user_id = $1;

-- name: CountPendingClaims :one
SELECT count(*)::int FROM finance.reimbursement_claims WHERE status IN ('SUBMITTED', 'APPROVED_MANAGER', 'APPROVED_FINANCE');

-- name: InsertInsight :exec
INSERT INTO finance.financial_insights (id, date_from, date_to, language, model, content, metrics, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListInsights :many
SELECT * FROM finance.financial_insights ORDER BY created_at DESC LIMIT $1;
