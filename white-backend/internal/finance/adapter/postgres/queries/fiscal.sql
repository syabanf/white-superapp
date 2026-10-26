-- name: ListFiscalYears :many
SELECT y.id, y.code, y.name, y.start_date, y.end_date, y.is_active, y.created_at, y.updated_at
  FROM finance.fiscal_years y
 WHERE y.deleted_at IS NULL
   AND (sqlc.narg(is_active)::boolean IS NULL OR y.is_active = sqlc.narg(is_active))
   AND (sqlc.narg(search)::text IS NULL OR y.code ILIKE '%' || sqlc.narg(search) || '%' OR y.name ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY y.start_date DESC, y.code;

-- name: GetFiscalYear :one
SELECT y.id, y.code, y.name, y.start_date, y.end_date, y.is_active, y.created_at, y.updated_at
  FROM finance.fiscal_years y
 WHERE y.id = $1 AND y.deleted_at IS NULL;

-- name: ListPeriods :many
SELECT p.id, p.fiscal_year_id, p.period_no, p.name, p.start_date, p.end_date, p.status,
       y.code AS year_code, y.name AS year_name, y.is_active AS year_active,
       (SELECT count(*) FROM finance.journal_entries e WHERE e.fiscal_period_id = p.id AND e.deleted_at IS NULL AND e.status = 'POSTED')::int AS posted_count,
       (SELECT count(*) FROM finance.journal_entries e WHERE e.fiscal_period_id = p.id AND e.deleted_at IS NULL AND e.status = 'DRAFT')::int AS draft_count
  FROM finance.fiscal_periods p
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id AND y.deleted_at IS NULL
 WHERE (sqlc.narg(fiscal_year_ids)::text[] IS NULL OR p.fiscal_year_id = ANY(sqlc.narg(fiscal_year_ids)::text[]))
   AND (sqlc.narg(active_only)::boolean IS NULL OR y.is_active)
   AND (sqlc.narg(status)::text IS NULL OR p.status = sqlc.narg(status))
   AND (sqlc.narg(search)::text IS NULL OR p.name ILIKE '%' || sqlc.narg(search) || '%' OR y.code ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY y.start_date DESC, p.period_no;

-- name: GetPeriod :one
SELECT p.id, p.fiscal_year_id, p.period_no, p.name, p.start_date, p.end_date, p.status,
       y.code AS year_code, y.name AS year_name, y.is_active AS year_active,
       (SELECT count(*) FROM finance.journal_entries e WHERE e.fiscal_period_id = p.id AND e.deleted_at IS NULL AND e.status = 'POSTED')::int AS posted_count,
       (SELECT count(*) FROM finance.journal_entries e WHERE e.fiscal_period_id = p.id AND e.deleted_at IS NULL AND e.status = 'DRAFT')::int AS draft_count
  FROM finance.fiscal_periods p
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id AND y.deleted_at IS NULL
 WHERE p.id = $1;

-- name: OpenPeriodFor :one
SELECT p.id, p.fiscal_year_id, p.period_no, p.name, p.start_date, p.end_date, p.status,
       y.code AS year_code, y.name AS year_name, y.is_active AS year_active
  FROM finance.fiscal_periods p
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id AND y.deleted_at IS NULL
 WHERE y.is_active AND p.status = 'OPEN' AND p.start_date <= sqlc.arg(day)::date AND p.end_date >= sqlc.arg(day)::date
 ORDER BY p.period_no
 LIMIT 1;

-- name: CoveringPeriod :one
SELECT p.id, p.fiscal_year_id, p.period_no, p.name, p.start_date, p.end_date, p.status,
       y.code AS year_code, y.name AS year_name, y.is_active AS year_active
  FROM finance.fiscal_periods p
  JOIN finance.fiscal_years y ON y.id = p.fiscal_year_id AND y.deleted_at IS NULL
 WHERE y.is_active AND p.start_date <= sqlc.arg(day)::date AND p.end_date >= sqlc.arg(day)::date
 ORDER BY p.period_no
 LIMIT 1;

-- name: PreviousOpenPeriods :many
SELECT p.id, p.fiscal_year_id, p.period_no, p.name, p.start_date, p.end_date, p.status
  FROM finance.fiscal_periods p
 WHERE p.fiscal_year_id = $1 AND p.period_no < $2 AND p.status = 'OPEN'
 ORDER BY p.period_no;

-- name: SetPeriodStatus :exec
UPDATE finance.fiscal_periods SET status = sqlc.arg(status), updated_at = now() WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: InsertFiscalYear :exec
INSERT INTO finance.fiscal_years (id, code, name, start_date, end_date, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateFiscalYear :exec
UPDATE finance.fiscal_years
   SET code = $2, name = $3, start_date = $4, end_date = $5, is_active = $6, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteFiscalYear :exec
UPDATE finance.fiscal_years
   SET deleted_at = now(), deleted_by = sqlc.narg(by), is_active = false, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteYearPeriods :exec
DELETE FROM finance.fiscal_periods WHERE fiscal_year_id = $1;

-- name: UpsertPeriod :exec
INSERT INTO finance.fiscal_periods (id, fiscal_year_id, period_no, name, start_date, end_date, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (fiscal_year_id, period_no) DO UPDATE
   SET name = EXCLUDED.name, start_date = EXCLUDED.start_date, end_date = EXCLUDED.end_date,
       status = EXCLUDED.status, updated_at = now();

-- name: DeleteUnusedPeriods :exec
DELETE FROM finance.fiscal_periods p
 WHERE p.fiscal_year_id = $1
   AND p.period_no <> ALL(sqlc.arg(keep)::int[])
   AND NOT EXISTS (SELECT 1 FROM finance.journal_entries e WHERE e.fiscal_period_id = p.id AND e.deleted_at IS NULL);

-- name: UsedPeriodNos :many
SELECT DISTINCT p.period_no
  FROM finance.fiscal_periods p
  JOIN finance.journal_entries e ON e.fiscal_period_id = p.id AND e.deleted_at IS NULL
 WHERE p.fiscal_year_id = $1
 ORDER BY p.period_no;

-- name: CountYearEntries :one
SELECT count(*)::int
  FROM finance.journal_entries e
  JOIN finance.fiscal_periods p ON p.id = e.fiscal_period_id
 WHERE p.fiscal_year_id = $1 AND e.deleted_at IS NULL;

-- name: PriorYearWithOpenPeriods :one
SELECT y.id, y.code, y.name, y.end_date, count(p.id)::int AS open_count
  FROM finance.fiscal_years y
  JOIN finance.fiscal_periods p ON p.fiscal_year_id = y.id AND p.status = 'OPEN'
 WHERE y.deleted_at IS NULL AND y.end_date < sqlc.arg(before)::date AND y.id <> sqlc.arg(exclude_id)::text
 GROUP BY y.id, y.code, y.name, y.end_date
 ORDER BY y.end_date DESC
 LIMIT 1;

-- name: PriorFiscalYear :one
SELECT y.id, y.code, y.name, y.end_date,
       (SELECT count(*) FROM finance.fiscal_periods p WHERE p.fiscal_year_id = y.id AND p.status = 'OPEN')::int AS open_count
  FROM finance.fiscal_years y
 WHERE y.deleted_at IS NULL AND y.end_date < sqlc.arg(before)::date
 ORDER BY y.end_date DESC
 LIMIT 1;

-- name: CoveringFiscalYearStart :one
SELECT start_date FROM finance.fiscal_years
 WHERE deleted_at IS NULL AND start_date <= sqlc.arg(day)::date AND end_date >= sqlc.arg(day)::date
 ORDER BY start_date DESC
 LIMIT 1;

-- name: DraftEntries :many
SELECT id, entry_no, entry_date, description
  FROM finance.journal_entries
 WHERE fiscal_period_id = $1 AND deleted_at IS NULL AND status = 'DRAFT'
 ORDER BY entry_date, entry_no
 LIMIT $2;
