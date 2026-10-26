-- name: ListInvoices :many
SELECT i.id, i.kind, i.invoice_no, i.invoice_date, i.due_date, i.party_name, i.reference, i.account_id,
       COALESCE(a.name, '')::text AS account_name, i.currency, i.subtotal, i.tax_amount, i.total_amount, i.status,
       i.description, i.sales_invoice_id, i.client_id, i.deal_id, i.journal_entry_id, i.posted_at, i.created_by, i.created_at,
       COALESCE((SELECT SUM(pa.amount) FROM finance.payment_allocations pa
                   JOIN finance.payments p ON p.id = pa.payment_id
                  WHERE pa.invoice_id = i.id AND p.deleted_at IS NULL AND p.status = 'POSTED'), 0)::numeric AS allocated,
       count(*) OVER ()::int AS total
  FROM finance.invoices i
  LEFT JOIN finance.accounts a ON a.id = i.account_id
 WHERE i.deleted_at IS NULL AND i.kind = $1
   AND (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status))
   AND (sqlc.narg(search)::text IS NULL OR i.invoice_no ILIKE '%' || sqlc.narg(search) || '%'
        OR i.party_name ILIKE '%' || sqlc.narg(search) || '%' OR i.reference ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY i.invoice_date DESC, i.created_at DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetInvoice :one
SELECT i.id, i.kind, i.invoice_no, i.invoice_date, i.due_date, i.party_name, i.reference, i.account_id,
       COALESCE(a.name, '')::text AS account_name, i.currency, i.subtotal, i.tax_amount, i.total_amount, i.status,
       i.description, i.sales_invoice_id, i.client_id, i.deal_id, i.journal_entry_id, i.posted_at, i.created_by, i.created_at,
       COALESCE((SELECT SUM(pa.amount) FROM finance.payment_allocations pa
                   JOIN finance.payments p ON p.id = pa.payment_id
                  WHERE pa.invoice_id = i.id AND p.deleted_at IS NULL AND p.status = 'POSTED'), 0)::numeric AS allocated
  FROM finance.invoices i
  LEFT JOIN finance.accounts a ON a.id = i.account_id
 WHERE i.id = $1 AND i.deleted_at IS NULL;

-- name: GetInvoiceIDBySales :one
SELECT id FROM finance.invoices WHERE sales_invoice_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: InsertInvoice :exec
INSERT INTO finance.invoices (id, kind, invoice_no, invoice_date, due_date, party_name, reference, account_id, currency,
                              subtotal, tax_amount, total_amount, status, description, sales_invoice_id, client_id, deal_id,
                              posted_at, posted_by, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
        CASE WHEN $13::text = 'POSTED' THEN now() END,
        CASE WHEN $13::text = 'POSTED' THEN sqlc.narg(by)::text END,
        sqlc.narg(by), sqlc.narg(by));

-- name: UpdateInvoice :exec
UPDATE finance.invoices
   SET invoice_date = $2, due_date = $3, party_name = $4, reference = $5, account_id = $6, subtotal = $7,
       tax_amount = $8, total_amount = $9, description = $10, client_id = $11, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL AND status = 'DRAFT';

-- name: SoftDeleteInvoice :exec
UPDATE finance.invoices SET deleted_at = now(), deleted_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: MarkInvoicePosted :exec
UPDATE finance.invoices SET status = 'POSTED', posted_at = $2, posted_by = sqlc.narg(by), updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL AND status = 'DRAFT';

-- name: SetInvoiceJournal :exec
UPDATE finance.invoices SET journal_entry_id = $2, updated_at = now() WHERE id = $1;

-- name: ListPayments :many
SELECT p.id, p.kind, p.payment_no, p.payment_date, p.amount, p.method, p.cash_account_id,
       COALESCE(a.name, '')::text AS cash_account_name, p.reference_number, p.notes, p.status, p.party_name,
       p.sales_payment_id, p.journal_entry_id, p.void_reason, p.voided_at, p.created_by, p.created_at,
       count(*) OVER ()::int AS total
  FROM finance.payments p
  LEFT JOIN finance.accounts a ON a.id = p.cash_account_id
 WHERE p.deleted_at IS NULL AND p.kind = $1
   AND (sqlc.narg(search)::text IS NULL OR p.payment_no ILIKE '%' || sqlc.narg(search) || '%' OR p.party_name ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY p.payment_date DESC, p.created_at DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetPayment :one
SELECT p.id, p.kind, p.payment_no, p.payment_date, p.amount, p.method, p.cash_account_id,
       COALESCE(a.name, '')::text AS cash_account_name, p.reference_number, p.notes, p.status, p.party_name,
       p.sales_payment_id, p.journal_entry_id, p.void_reason, p.voided_at, p.created_by, p.created_at
  FROM finance.payments p
  LEFT JOIN finance.accounts a ON a.id = p.cash_account_id
 WHERE p.id = $1 AND p.deleted_at IS NULL;

-- name: GetPaymentIDBySales :one
SELECT id FROM finance.payments WHERE sales_payment_id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: PaymentAllocations :many
SELECT pa.payment_id, pa.invoice_id, i.invoice_no, pa.amount
  FROM finance.payment_allocations pa
  JOIN finance.invoices i ON i.id = pa.invoice_id
 WHERE pa.payment_id = ANY(sqlc.arg(payment_ids)::text[])
 ORDER BY i.invoice_no;

-- name: InsertPayment :exec
INSERT INTO finance.payments (id, kind, payment_no, payment_date, amount, method, cash_account_id, reference_number, notes,
                              status, party_name, sales_payment_id, posted_at, posted_by, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'POSTED', $10, $11, now(), sqlc.narg(by), sqlc.narg(by), sqlc.narg(by));

-- name: InsertAllocation :exec
INSERT INTO finance.payment_allocations (id, payment_id, invoice_id, amount) VALUES ($1, $2, $3, $4);

-- name: VoidPayment :exec
UPDATE finance.payments
   SET status = 'VOID', voided_at = now(), voided_by = sqlc.narg(by), void_reason = $2, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1;

-- name: SetPaymentJournal :exec
UPDATE finance.payments SET journal_entry_id = $2, updated_at = now() WHERE id = $1;

-- name: SubsidiaryParties :many
SELECT i.party_name,
       SUM(GREATEST(i.total_amount - COALESCE((
         SELECT SUM(pa.amount) FROM finance.payment_allocations pa
           JOIN finance.payments p ON p.id = pa.payment_id
          WHERE pa.invoice_id = i.id AND p.deleted_at IS NULL AND p.status = 'POSTED'), 0), 0))::numeric AS outstanding,
       count(*)::int AS invoice_count
  FROM finance.invoices i
 WHERE i.deleted_at IS NULL AND i.status = 'POSTED' AND i.kind = $1
 GROUP BY i.party_name
 ORDER BY i.party_name;

-- name: SubsidiaryInvoices :many
SELECT i.id, i.invoice_no, i.invoice_date, i.total_amount, i.description
  FROM finance.invoices i
 WHERE i.deleted_at IS NULL AND i.status = 'POSTED' AND i.kind = $1 AND i.party_name = $2
   AND (sqlc.narg(date_from)::date IS NULL OR i.invoice_date >= sqlc.narg(date_from))
   AND (sqlc.narg(date_to)::date IS NULL OR i.invoice_date <= sqlc.narg(date_to))
 ORDER BY i.invoice_date, i.invoice_no;

-- name: SubsidiaryAllocations :many
SELECT pa.id, p.payment_no, p.payment_date, pa.amount, i.invoice_no
  FROM finance.payment_allocations pa
  JOIN finance.payments p ON p.id = pa.payment_id
  JOIN finance.invoices i ON i.id = pa.invoice_id
 WHERE p.deleted_at IS NULL AND p.status = 'POSTED' AND i.deleted_at IS NULL AND i.status = 'POSTED'
   AND i.kind = $1 AND i.party_name = $2
   AND (sqlc.narg(date_from)::date IS NULL OR p.payment_date >= sqlc.narg(date_from))
   AND (sqlc.narg(date_to)::date IS NULL OR p.payment_date <= sqlc.narg(date_to))
 ORDER BY p.payment_date, p.payment_no;

-- name: SubsidiaryOpening :one
SELECT (COALESCE((SELECT SUM(i.total_amount) FROM finance.invoices i
          WHERE i.deleted_at IS NULL AND i.status = 'POSTED' AND i.kind = $1 AND i.party_name = $2
            AND i.invoice_date < sqlc.arg(before)::date), 0)
      - COALESCE((SELECT SUM(pa.amount) FROM finance.payment_allocations pa
          JOIN finance.payments p ON p.id = pa.payment_id
          JOIN finance.invoices i ON i.id = pa.invoice_id
          WHERE p.deleted_at IS NULL AND p.status = 'POSTED' AND i.deleted_at IS NULL AND i.status = 'POSTED'
            AND i.kind = $1 AND i.party_name = $2 AND p.payment_date < sqlc.arg(before)::date), 0))::numeric AS opening;
