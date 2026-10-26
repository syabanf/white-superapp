-- name: MonthlyAccountMovements :many
-- Posted lines per month and account, skipping an OPENING entry that
-- restates earlier posted activity (as AccountBalances does for windows).
SELECT to_char(e.entry_date, 'YYYY-MM')::text AS month, a.id, a.code, a.name,
       t.code AS type_code, t.normal_balance, a.is_contra, a.is_cash_bank,
       COALESCE(a.cash_flow_category, '')::text AS cash_flow_category,
       COALESCE(p.id, '')::text AS parent_id, COALESCE(p.name, '')::text AS parent_name,
       COALESCE(p.parent_id, '')::text AS grandparent_id,
       SUM(CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE 0 END)::numeric AS debit,
       SUM(CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE 0 END)::numeric AS credit
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
  JOIN finance.accounts a ON a.id = l.account_id
  JOIN finance.account_types t ON t.id = a.account_type_id
  LEFT JOIN finance.accounts p ON p.id = a.parent_id
 WHERE e.deleted_at IS NULL AND e.status = 'POSTED'
   AND e.entry_date >= sqlc.arg(date_from)::date AND e.entry_date <= sqlc.arg(date_to)::date
   AND (e.entry_type <> 'OPENING' OR NOT EXISTS (
         SELECT 1 FROM finance.journal_entries x
          WHERE x.status = 'POSTED' AND x.deleted_at IS NULL AND x.entry_date < e.entry_date))
 GROUP BY 1, a.id, a.code, a.name, t.code, t.normal_balance, a.is_contra, a.is_cash_bank, a.cash_flow_category, p.id, p.name, p.parent_id
 ORDER BY 1, a.code;

-- name: AccountParents :many
-- Parent and grandparent of every live account, for group filters.
SELECT a.id, a.code, a.name, COALESCE(p.id, '')::text AS parent_id, COALESCE(p.name, '')::text AS parent_name,
       COALESCE(p.parent_id, '')::text AS grandparent_id
  FROM finance.accounts a
  LEFT JOIN finance.accounts p ON p.id = a.parent_id
 WHERE a.deleted_at IS NULL;

-- name: AnalyticsInvoices :many
-- Posted invoices issued on or before as_of with what was paid by then.
SELECT i.id, i.kind, i.invoice_no, i.invoice_date, i.due_date, i.party_name, i.subtotal, i.total_amount
  FROM finance.invoices i
 WHERE i.deleted_at IS NULL AND i.status = 'POSTED' AND i.invoice_date <= sqlc.arg(as_of)::date
 ORDER BY i.invoice_date, i.invoice_no;

-- name: AnalyticsAllocations :many
SELECT pa.invoice_id, p.payment_date, pa.amount
  FROM finance.payment_allocations pa
  JOIN finance.payments p ON p.id = pa.payment_id
 WHERE p.deleted_at IS NULL AND p.status = 'POSTED' AND p.payment_date <= sqlc.arg(as_of)::date;

-- name: AnalyticsClaims :many
-- Claims dated in the window, plus every claim still waiting.
SELECT c.id, c.claim_no, c.employee_name, c.department_name, c.category, c.transaction_date, c.amount, c.status,
       c.submitted_at, c.manager_approved_at, c.finance_approved_at, c.paid_at, c.rejected_at
  FROM finance.reimbursement_claims c
 WHERE (sqlc.narg(department)::text IS NULL OR c.department_name = sqlc.narg(department))
   AND ((c.transaction_date >= sqlc.arg(date_from)::date AND c.transaction_date <= sqlc.arg(date_to)::date)
        OR c.status IN ('SUBMITTED', 'APPROVED_MANAGER', 'APPROVED_FINANCE'))
 ORDER BY c.transaction_date, c.claim_no;

-- name: ClaimDepartments :many
SELECT DISTINCT department_name::text FROM finance.reimbursement_claims WHERE department_name <> '' ORDER BY 1;
