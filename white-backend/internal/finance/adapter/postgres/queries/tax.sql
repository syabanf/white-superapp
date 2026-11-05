-- name: ListTaxCodes :many
SELECT * FROM finance.tax_codes
 WHERE (sqlc.narg(active)::boolean IS NULL OR is_active = sqlc.narg(active))
 ORDER BY is_active DESC, kind DESC, code;

-- name: GetTaxCode :one
SELECT * FROM finance.tax_codes WHERE id = $1;

-- name: UpsertTaxCode :exec
INSERT INTO finance.tax_codes (id, code, name, kind, scope, rate, base_numerator, base_denominator, price_included, account_sale_id, account_purchase_id, report_code, valid_from, valid_to, is_default_sale, is_default_purchase, is_active, notes, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, sqlc.narg(account_sale_id), sqlc.narg(account_purchase_id), $10, sqlc.narg(valid_from), sqlc.narg(valid_to), $11, $12, $13, $14, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, kind = EXCLUDED.kind, scope = EXCLUDED.scope, rate = EXCLUDED.rate, base_numerator = EXCLUDED.base_numerator,
  base_denominator = EXCLUDED.base_denominator, price_included = EXCLUDED.price_included, account_sale_id = EXCLUDED.account_sale_id, account_purchase_id = EXCLUDED.account_purchase_id,
  report_code = EXCLUDED.report_code, valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to, is_default_sale = EXCLUDED.is_default_sale, is_default_purchase = EXCLUDED.is_default_purchase,
  is_active = EXCLUDED.is_active, notes = EXCLUDED.notes, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: ClearDefaultSaleTax :exec
UPDATE finance.tax_codes SET is_default_sale = false WHERE id <> $1;

-- name: ClearDefaultPurchaseTax :exec
UPDATE finance.tax_codes SET is_default_purchase = false WHERE id <> $1;

-- name: ListChargeTypes :many
SELECT * FROM finance.charge_types
 WHERE (sqlc.narg(active)::boolean IS NULL OR is_active = sqlc.narg(active))
 ORDER BY is_active DESC, code;

-- name: GetChargeType :one
SELECT * FROM finance.charge_types WHERE id = $1;

-- name: UpsertChargeType :exec
INSERT INTO finance.charge_types (id, code, name, scope, computation, value, taxable, is_deduction, account_sale_id, account_purchase_id, is_active, notes, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, sqlc.narg(account_sale_id), sqlc.narg(account_purchase_id), $9, $10, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, scope = EXCLUDED.scope, computation = EXCLUDED.computation, value = EXCLUDED.value, taxable = EXCLUDED.taxable,
  is_deduction = EXCLUDED.is_deduction, account_sale_id = EXCLUDED.account_sale_id, account_purchase_id = EXCLUDED.account_purchase_id, is_active = EXCLUDED.is_active, notes = EXCLUDED.notes,
  updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: InsertInvoicePosting :exec
INSERT INTO finance.invoice_postings (id, invoice_id, sort_order, account_id, entry_side, amount, memo, tax_code, tax_name, tax_kind, tax_base)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: ListInvoicePostings :many
SELECT * FROM finance.invoice_postings WHERE invoice_id = $1 ORDER BY sort_order;

-- name: DocumentCountByInvoice :one
SELECT count(*)::int FROM finance.documents WHERE ledger_invoice_id = $1;

-- name: TaxRegister :many
SELECT i.kind, p.tax_code, p.tax_name, p.tax_kind, count(DISTINCT i.id)::int AS invoice_count, sum(p.tax_base)::numeric AS base, sum(p.amount)::numeric AS amount
  FROM finance.invoice_postings p
  JOIN finance.invoices i ON i.id = p.invoice_id
 WHERE p.tax_code <> '' AND i.status = 'POSTED' AND i.deleted_at IS NULL
   AND i.invoice_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
 GROUP BY i.kind, p.tax_code, p.tax_name, p.tax_kind
 ORDER BY i.kind, p.tax_kind DESC, p.tax_code;

-- name: UntaggedTaxTotals :many
SELECT i.kind, count(*)::int AS invoice_count, sum(i.subtotal)::numeric AS base, sum(i.tax_amount)::numeric AS amount
  FROM finance.invoices i
 WHERE i.status = 'POSTED' AND i.deleted_at IS NULL AND i.tax_amount > 0
   AND i.invoice_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
   AND NOT EXISTS (SELECT 1 FROM finance.invoice_postings p WHERE p.invoice_id = i.id AND p.tax_kind = 'VAT')
 GROUP BY i.kind
 ORDER BY i.kind;
