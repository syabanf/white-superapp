-- name: AccountBalances :many
-- Posted lines per postable active account inside an optional window.
SELECT a.id, a.code, a.name, t.code AS type_code, t.normal_balance, a.is_contra, a.is_cash_bank,
       COALESCE(a.cash_flow_category, '')::text AS cash_flow_category, a.level,
       COALESCE(SUM(mov.debit), 0)::numeric AS debit,
       COALESCE(SUM(mov.credit), 0)::numeric AS credit,
       count(mov.line_id)::int AS movement_count
  FROM finance.accounts a
  JOIN finance.account_types t ON t.id = a.account_type_id
  LEFT JOIN (
    SELECT l.account_id, l.id AS line_id,
           CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE 0 END AS debit,
           CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE 0 END AS credit
      FROM finance.journal_entry_lines l
      JOIN finance.journal_entries e ON e.id = l.entry_id
     WHERE e.deleted_at IS NULL AND e.status = 'POSTED'
       AND (sqlc.narg(as_of)::date IS NULL OR e.entry_date <= sqlc.narg(as_of))
       AND (sqlc.narg(date_from)::date IS NULL OR e.entry_date >= sqlc.narg(date_from))
       AND (sqlc.narg(date_to)::date IS NULL OR e.entry_date <= sqlc.narg(date_to))
       -- A posted OPENING entry restates every balance: an as-of total starts
       -- at the latest one, and a date window skips one that restates
       -- earlier posted activity.
       AND (sqlc.narg(as_of)::date IS NULL OR e.entry_date >= COALESCE((
             SELECT max(o.entry_date) FROM finance.journal_entries o
              WHERE o.entry_type = 'OPENING' AND o.status = 'POSTED' AND o.deleted_at IS NULL
                AND o.entry_date <= sqlc.narg(as_of)), '-infinity'::date))
       AND (sqlc.narg(as_of)::date IS NOT NULL OR e.entry_type <> 'OPENING' OR NOT EXISTS (
             SELECT 1 FROM finance.journal_entries x
              WHERE x.status = 'POSTED' AND x.deleted_at IS NULL AND x.entry_date < e.entry_date))
  ) mov ON mov.account_id = a.id
 WHERE a.deleted_at IS NULL AND a.is_active AND a.is_postable
 GROUP BY a.id, a.code, a.name, t.code, t.normal_balance, a.is_contra, a.is_cash_bank, a.cash_flow_category, a.level
 ORDER BY a.code;

-- name: YearAccountTotals :many
SELECT a.id, a.code, a.name, t.code AS type_code, t.normal_balance, a.is_contra,
       COALESCE(SUM(CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE 0 END), 0)::numeric AS debit,
       COALESCE(SUM(CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE 0 END), 0)::numeric AS credit
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id AND e.deleted_at IS NULL AND e.status = 'POSTED'
  JOIN finance.fiscal_periods p ON p.id = e.fiscal_period_id
  JOIN finance.accounts a ON a.id = l.account_id AND a.deleted_at IS NULL AND a.is_postable
  JOIN finance.account_types t ON t.id = a.account_type_id
 WHERE p.fiscal_year_id = $1
 GROUP BY a.id, a.code, a.name, t.code, t.normal_balance, a.is_contra
 ORDER BY a.code;

-- name: AccountTotalsBefore :one
SELECT COALESCE(SUM(CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE 0 END), 0)::numeric AS debit,
       COALESCE(SUM(CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE 0 END), 0)::numeric AS credit
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id AND e.deleted_at IS NULL AND e.status = 'POSTED'
 WHERE l.account_id = $1 AND e.entry_date < sqlc.arg(before)::date
   -- Same OPENING anchor as AccountBalances (as of the day before).
   AND e.entry_date >= COALESCE((
         SELECT max(o.entry_date) FROM finance.journal_entries o
          WHERE o.entry_type = 'OPENING' AND o.status = 'POSTED' AND o.deleted_at IS NULL
            AND o.entry_date < sqlc.arg(before)::date), '-infinity'::date);

-- name: LedgerLines :many
SELECT l.id, e.id AS entry_id, e.entry_no, e.entry_date, e.entry_type, e.description, l.memo, l.entry_side, l.amount
  FROM finance.journal_entry_lines l
  JOIN finance.journal_entries e ON e.id = l.entry_id
 WHERE l.account_id = $1 AND e.deleted_at IS NULL AND e.status = 'POSTED'
   AND (sqlc.narg(date_from)::date IS NULL OR e.entry_date >= sqlc.narg(date_from))
   AND (sqlc.narg(date_to)::date IS NULL OR e.entry_date <= sqlc.narg(date_to))
   AND (e.entry_type <> 'OPENING' OR NOT EXISTS (
         SELECT 1 FROM finance.journal_entries x
          WHERE x.status = 'POSTED' AND x.deleted_at IS NULL AND x.entry_date < e.entry_date))
 ORDER BY e.entry_date, e.entry_no, l.sort_order, l.entry_side;

-- name: MonthlyPnL :many
WITH months AS (
  SELECT to_char(d, 'YYYY-MM') AS month,
         date_trunc('month', d)::date AS start_date,
         (date_trunc('month', d) + interval '1 month' - interval '1 day')::date AS end_date
    FROM generate_series(date_trunc('month', sqlc.arg(date_from)::date), date_trunc('month', sqlc.arg(date_to)::date), interval '1 month') AS d
)
SELECT m.month::text AS month,
       COALESCE(SUM(CASE WHEN t.code IN ('REVENUE', 'OTHER_INCOME') THEN
         CASE WHEN l.entry_side = 'CREDIT' THEN l.amount ELSE -l.amount END ELSE 0 END), 0)::numeric AS revenue,
       COALESCE(SUM(CASE WHEN t.code IN ('EXPENSE', 'COGS', 'OTHER_EXPENSE') THEN
         CASE WHEN l.entry_side = 'DEBIT' THEN l.amount ELSE -l.amount END ELSE 0 END), 0)::numeric AS expense
  FROM months m
  LEFT JOIN finance.journal_entries e
    ON e.deleted_at IS NULL AND e.status = 'POSTED'
   AND e.entry_date >= m.start_date AND e.entry_date <= LEAST(m.end_date, sqlc.arg(date_to)::date)
  LEFT JOIN finance.journal_entry_lines l ON l.entry_id = e.id
  LEFT JOIN finance.accounts a ON a.id = l.account_id
  LEFT JOIN finance.account_types t ON t.id = a.account_type_id
 GROUP BY m.month
 ORDER BY m.month;
