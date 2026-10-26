-- name: NextSequence :one
INSERT INTO finance.document_sequences (prefix, last_no) VALUES ($1, 1)
ON CONFLICT (prefix) DO UPDATE SET last_no = finance.document_sequences.last_no + 1
RETURNING last_no;

-- name: InsertJournalEntry :exec
INSERT INTO finance.journal_entries (
  id, entry_no, entry_date, description, fiscal_period_id, status, is_recon, entry_type,
  source_module, source_event_code, source_document_type, source_document_id, source_fiscal_year_id,
  posted_at, posted_by, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
        CASE WHEN $6::text = 'POSTED' THEN now() END,
        CASE WHEN $6::text = 'POSTED' THEN sqlc.narg(by)::text END,
        sqlc.narg(by), sqlc.narg(by));

-- name: UpdateJournalHeader :exec
UPDATE finance.journal_entries
   SET entry_date = $2, description = $3, fiscal_period_id = $4, status = $5, is_recon = $6,
       posted_at = CASE WHEN $5::text = 'POSTED' THEN COALESCE(posted_at, now()) END,
       posted_by = CASE WHEN $5::text = 'POSTED' THEN COALESCE(posted_by, sqlc.narg(by)::text) END,
       updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteJournalLines :exec
DELETE FROM finance.journal_entry_lines WHERE entry_id = $1;

-- name: InsertJournalLine :exec
INSERT INTO finance.journal_entry_lines (id, entry_id, account_id, entry_side, amount, memo, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: MarkJournalPosted :exec
UPDATE finance.journal_entries
   SET status = 'POSTED', posted_at = now(), posted_by = sqlc.narg(by), updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL AND status = 'DRAFT';

-- name: SoftDeleteJournalEntry :exec
UPDATE finance.journal_entries
   SET deleted_at = now(), deleted_by = sqlc.narg(by), updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: ListJournalEntries :many
SELECT e.id, e.entry_no, e.entry_date, e.description, e.fiscal_period_id, p.name AS period_name, y.code AS year_code,
       e.entry_type, e.status, e.is_recon, e.posted_at, e.posted_by, e.source_module, e.source_event_code,
       e.source_document_type, e.source_document_id, e.source_fiscal_year_id, e.created_by, e.created_at, e.updated_at,
       count(*) OVER ()::int AS total
  FROM finance.journal_entries e
  JOIN finance.fiscal_periods p ON p.id = e.fiscal_period_id
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id
 WHERE e.deleted_at IS NULL
   AND (sqlc.narg(status)::text IS NULL OR e.status = sqlc.narg(status))
   AND (sqlc.narg(entry_type)::text IS NULL OR e.entry_type = sqlc.narg(entry_type))
   AND (sqlc.narg(date_from)::date IS NULL OR e.entry_date >= sqlc.narg(date_from))
   AND (sqlc.narg(date_to)::date IS NULL OR e.entry_date <= sqlc.narg(date_to))
   AND (sqlc.narg(search)::text IS NULL OR e.entry_no ILIKE '%' || sqlc.narg(search) || '%' OR e.description ILIKE '%' || sqlc.narg(search) || '%')
   AND (sqlc.narg(source_document_type)::text IS NULL OR e.source_document_type = sqlc.narg(source_document_type))
   AND (sqlc.narg(account_id)::text IS NULL OR EXISTS (
         SELECT 1 FROM finance.journal_entry_lines l WHERE l.entry_id = e.id AND l.account_id = sqlc.narg(account_id)))
 ORDER BY e.entry_date DESC, e.entry_no DESC
 LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetJournalEntry :one
SELECT e.id, e.entry_no, e.entry_date, e.description, e.fiscal_period_id, p.name AS period_name, y.code AS year_code,
       e.entry_type, e.status, e.is_recon, e.posted_at, e.posted_by, e.source_module, e.source_event_code,
       e.source_document_type, e.source_document_id, e.source_fiscal_year_id, e.created_by, e.created_at, e.updated_at
  FROM finance.journal_entries e
  JOIN finance.fiscal_periods p ON p.id = e.fiscal_period_id
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id
 WHERE e.id = $1 AND e.deleted_at IS NULL;

-- name: FindEntryBySource :one
SELECT e.id
  FROM finance.journal_entries e
 WHERE e.deleted_at IS NULL AND e.source_event_code = $1 AND e.source_document_id = $2
 LIMIT 1;

-- name: OpeningEntryID :one
SELECT e.id
  FROM finance.journal_entries e
 WHERE e.deleted_at IS NULL AND e.entry_type = 'OPENING' AND e.source_fiscal_year_id = $1
 LIMIT 1;

-- name: JournalLines :many
SELECT l.id, l.entry_id, l.account_id, l.entry_side, l.amount, l.memo, l.sort_order,
       COALESCE(a.code, '')::text AS account_code, COALESCE(a.name, '')::text AS account_name
  FROM finance.journal_entry_lines l
  LEFT JOIN finance.accounts a ON a.id = l.account_id
 WHERE l.entry_id = ANY(sqlc.arg(entry_ids)::text[])
 ORDER BY l.sort_order, l.entry_side;

-- name: CountEntries :one
SELECT count(*) FILTER (WHERE status = 'POSTED' AND entry_date >= sqlc.arg(date_from)::date AND entry_date <= sqlc.arg(date_to)::date)::int AS posted,
       count(*) FILTER (WHERE status = 'DRAFT')::int AS drafts
  FROM finance.journal_entries
 WHERE deleted_at IS NULL;

-- name: UpsertPostingLog :one
INSERT INTO finance.posting_logs (id, event_code, document_type, document_id, document_label, status, reason, entry_id, request)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (event_code, document_id) DO UPDATE
   SET document_type = EXCLUDED.document_type, document_label = EXCLUDED.document_label, status = EXCLUDED.status,
       reason = EXCLUDED.reason, entry_id = EXCLUDED.entry_id, request = EXCLUDED.request,
       attempts = finance.posting_logs.attempts + 1, updated_at = now()
RETURNING *;

-- name: GetPostingLog :one
SELECT * FROM finance.posting_logs WHERE id = $1;

-- name: ListPostingLogs :many
SELECT * FROM finance.posting_logs
 WHERE (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status))
   AND (sqlc.narg(event_code)::text IS NULL OR event_code = sqlc.narg(event_code))
 ORDER BY updated_at DESC
 LIMIT $1;
