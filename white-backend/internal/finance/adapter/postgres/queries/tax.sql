-- name: ListTaxCodes :many
SELECT * FROM finance.tax_codes
 WHERE (sqlc.narg(active)::boolean IS NULL OR is_active = sqlc.narg(active))
 ORDER BY is_active DESC, kind DESC, code;

-- name: GetTaxCode :one
SELECT * FROM finance.tax_codes WHERE id = $1;

-- name: GetTaxCodeByCode :one
SELECT * FROM finance.tax_codes WHERE code = $1;

-- name: UpsertTaxCode :exec
INSERT INTO finance.tax_codes (id, code, name, kind, scope, rate, base_numerator, base_denominator, price_included, account_sale_id, account_purchase_id, report_code, valid_from, valid_to, is_default_sale, is_default_purchase, is_active, notes, brackets, cash_basis, transition_account_id, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, sqlc.narg(account_sale_id), sqlc.narg(account_purchase_id), $10, sqlc.narg(valid_from), sqlc.narg(valid_to), $11, $12, $13, $14, $15, $16, sqlc.narg(transition_account_id), sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, kind = EXCLUDED.kind, scope = EXCLUDED.scope, rate = EXCLUDED.rate, base_numerator = EXCLUDED.base_numerator,
  base_denominator = EXCLUDED.base_denominator, price_included = EXCLUDED.price_included, account_sale_id = EXCLUDED.account_sale_id, account_purchase_id = EXCLUDED.account_purchase_id,
  report_code = EXCLUDED.report_code, valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to, is_default_sale = EXCLUDED.is_default_sale, is_default_purchase = EXCLUDED.is_default_purchase,
  is_active = EXCLUDED.is_active, notes = EXCLUDED.notes, brackets = EXCLUDED.brackets, cash_basis = EXCLUDED.cash_basis, transition_account_id = EXCLUDED.transition_account_id,
  updated_by = EXCLUDED.updated_by, updated_at = now();

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
INSERT INTO finance.invoice_postings (id, invoice_id, sort_order, account_id, entry_side, amount, memo, tax_code, tax_name, tax_kind, tax_base, tax_id, tax_rate, report_code, cash_basis, final_account_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, sqlc.narg(final_account_id));

-- name: DeleteInvoicePostings :exec
DELETE FROM finance.invoice_postings WHERE invoice_id = $1;

-- name: ListInvoicePostings :many
SELECT * FROM finance.invoice_postings WHERE invoice_id = $1 ORDER BY sort_order;

-- name: DocumentCountByInvoice :one
SELECT count(*)::int FROM finance.documents WHERE ledger_invoice_id = $1;

-- name: InsertTaxMove :exec
INSERT INTO finance.tax_moves (id, kind, source_type, source_id, invoice_id, tax_code, tax_name, tax_kind, tax_rate, report_code, move_date, base, amount)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (source_type, source_id, tax_code) DO NOTHING;

-- name: TaxRegister :many
SELECT kind, tax_code, tax_name, tax_kind, count(DISTINCT invoice_id)::int AS invoice_count, sum(base)::numeric AS base, sum(amount)::numeric AS amount
  FROM finance.tax_moves
 WHERE move_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
 GROUP BY kind, tax_code, tax_name, tax_kind
 ORDER BY kind, tax_kind DESC, tax_code;

-- name: TaxExportRows :many
SELECT m.move_date, m.tax_code, m.tax_name, m.tax_rate, m.report_code, m.base, m.amount, i.invoice_no, i.reference, i.party_name, i.subtotal, i.description,
       coalesce(p.npwp, '')::text AS npwp, coalesce(p.address, '')::text AS address
  FROM finance.tax_moves m
  JOIN finance.invoices i ON i.id = m.invoice_id
  LEFT JOIN finance.party_tax_profiles p ON p.side = CASE WHEN m.kind = 'AR' THEN 'B2B' ELSE 'SUPPLIER' END AND lower(p.name) = lower(i.party_name)
 WHERE m.kind = $1 AND m.tax_kind = $2 AND m.source_type IN ('INVOICE', 'CASH', 'PAYMENT')
   AND NOT EXISTS (SELECT 1 FROM finance.tax_moves v WHERE v.source_id = m.source_id AND v.tax_code = m.tax_code AND v.source_type IN ('CASH_VOID', 'PAYMENT_VOID'))
   AND m.move_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
 ORDER BY m.move_date, i.invoice_no;

-- name: ListFiscalPositions :many
SELECT * FROM finance.fiscal_positions ORDER BY is_active DESC, name;

-- name: GetFiscalPosition :one
SELECT * FROM finance.fiscal_positions WHERE id = $1;

-- name: UpsertFiscalPosition :exec
INSERT INTO finance.fiscal_positions (id, name, rules, is_active, notes, updated_by) VALUES ($1, $2, $3, $4, $5, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, rules = EXCLUDED.rules, is_active = EXCLUDED.is_active, notes = EXCLUDED.notes, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: ListPartyProfiles :many
SELECT * FROM finance.party_tax_profiles ORDER BY side, name;

-- name: GetPartyProfile :one
SELECT * FROM finance.party_tax_profiles WHERE side = $1 AND lower(name) = lower($2);

-- name: UpsertPartyProfile :exec
INSERT INTO finance.party_tax_profiles (id, side, name, npwp, address, is_pkp, fiscal_position_id, notes, updated_by) VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(fiscal_position_id), $7, sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET side = EXCLUDED.side, name = EXCLUDED.name, npwp = EXCLUDED.npwp, address = EXCLUDED.address, is_pkp = EXCLUDED.is_pkp, fiscal_position_id = EXCLUDED.fiscal_position_id,
  notes = EXCLUDED.notes, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: UntaggedTaxTotals :many
SELECT i.kind, count(*)::int AS invoice_count, sum(i.subtotal)::numeric AS base, sum(i.tax_amount)::numeric AS amount
  FROM finance.invoices i
 WHERE i.status = 'POSTED' AND i.deleted_at IS NULL AND i.tax_amount > 0
   AND i.invoice_date BETWEEN sqlc.arg(date_from)::date AND sqlc.arg(date_to)::date
   AND NOT EXISTS (SELECT 1 FROM finance.invoice_postings p WHERE p.invoice_id = i.id AND p.tax_kind = 'VAT')
 GROUP BY i.kind
 ORDER BY i.kind;

-- name: TaxCodeInUse :one
SELECT EXISTS (SELECT 1 FROM finance.invoice_postings ip WHERE ip.tax_id = sqlc.arg(id)::text)
    OR EXISTS (SELECT 1 FROM finance.tax_moves tm WHERE tm.tax_code = sqlc.arg(code)::text)
    OR EXISTS (SELECT 1 FROM finance.documents WHERE taxes @> jsonb_build_array(jsonb_build_object('taxId', sqlc.arg(id)::text)))
    OR EXISTS (SELECT 1 FROM finance.fiscal_positions
                WHERE rules @> jsonb_build_array(jsonb_build_object('fromTaxId', sqlc.arg(id)::text))
                   OR rules @> jsonb_build_array(jsonb_build_object('toTaxId', sqlc.arg(id)::text)));

-- name: DeleteTaxCode :exec
DELETE FROM finance.tax_codes WHERE id = $1;

-- name: ChargeTypeInUse :one
SELECT EXISTS (SELECT 1 FROM finance.documents WHERE charges @> jsonb_build_array(jsonb_build_object('chargeId', sqlc.arg(id)::text)));

-- name: DeleteChargeType :exec
DELETE FROM finance.charge_types WHERE id = $1;

-- name: FiscalPositionInUse :one
SELECT EXISTS (SELECT 1 FROM finance.party_tax_profiles WHERE fiscal_position_id = $1);

-- name: DeleteFiscalPosition :exec
DELETE FROM finance.fiscal_positions WHERE id = $1;

-- name: DeletePartyProfile :exec
DELETE FROM finance.party_tax_profiles WHERE id = $1;
