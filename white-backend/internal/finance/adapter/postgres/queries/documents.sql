-- name: ListVendors :many
SELECT * FROM finance.vendors
 WHERE (sqlc.narg(active)::boolean IS NULL OR is_active = sqlc.narg(active))
   AND (sqlc.narg(q)::text IS NULL OR name ILIKE '%' || sqlc.narg(q) || '%')
 ORDER BY is_active DESC, name;

-- name: GetVendor :one
SELECT * FROM finance.vendors WHERE id = $1;

-- name: UpsertVendor :exec
INSERT INTO finance.vendors (id, name, pic_name, phone, email, address, npwp, bank_name, bank_account_no, bank_account_name, payment_terms_days, notes, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, sqlc.narg(by), sqlc.narg(by))
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, pic_name = EXCLUDED.pic_name, phone = EXCLUDED.phone, email = EXCLUDED.email,
  address = EXCLUDED.address, npwp = EXCLUDED.npwp, bank_name = EXCLUDED.bank_name, bank_account_no = EXCLUDED.bank_account_no,
  bank_account_name = EXCLUDED.bank_account_name, payment_terms_days = EXCLUDED.payment_terms_days, notes = EXCLUDED.notes,
  is_active = EXCLUDED.is_active, updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: ListDocuments :many
SELECT *, count(*) OVER ()::int AS total
  FROM finance.documents
 WHERE kind = $1
   AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
   AND (sqlc.narg(q)::text IS NULL OR number ILIKE '%' || sqlc.narg(q) || '%' OR title ILIKE '%' || sqlc.narg(q) || '%' OR party_name ILIKE '%' || sqlc.narg(q) || '%')
 ORDER BY issue_date DESC, number DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetDocument :one
SELECT * FROM finance.documents WHERE id = $1;

-- name: InsertDocument :exec
INSERT INTO finance.documents (id, kind, side, number, status, vendor_id, party_name, pic_name, phone, email, address, title, issue_date, due_date,
  external_number, notes, items, discount_percent, taxes, charges, subtotal, discount_amount, charges_amount, tax_amount, withholding_amount, total_amount, account_id, source_id, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, sqlc.narg(vendor_id), $6, $7, $8, $9, $10, $11, $12, sqlc.narg(due_date), $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24,
  sqlc.narg(account_id), sqlc.narg(source_id), sqlc.narg(by), sqlc.narg(by));

-- name: UpdateDocument :exec
UPDATE finance.documents
   SET vendor_id = sqlc.narg(vendor_id), party_name = $2, pic_name = $3, phone = $4, email = $5, address = $6, title = $7, issue_date = $8, due_date = sqlc.narg(due_date),
       external_number = $9, notes = $10, items = $11, discount_percent = $12, taxes = $13, charges = $14, subtotal = $15, discount_amount = $16, charges_amount = $17, tax_amount = $18, withholding_amount = $19, total_amount = $20,
       account_id = sqlc.narg(account_id), status = $21, ledger_invoice_id = sqlc.narg(ledger_invoice_id), letterhead = sqlc.narg(letterhead), reject_reason = $22,
       decided_by = sqlc.narg(decided_by), decided_at = sqlc.narg(decided_at), updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1;

-- name: DeleteDocument :exec
DELETE FROM finance.documents WHERE id = $1;
