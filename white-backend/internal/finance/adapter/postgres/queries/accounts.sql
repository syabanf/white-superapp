-- name: ListAccountTypes :many
SELECT t.id, t.code, t.name, t.normal_balance, t.sort_order, t.is_active,
       (SELECT count(*) FROM finance.accounts a WHERE a.account_type_id = t.id AND a.deleted_at IS NULL)::int AS account_count
  FROM finance.account_types t
 WHERE sqlc.arg(include_inactive)::boolean OR t.is_active
 ORDER BY t.sort_order, t.code;

-- name: GetAccountType :one
SELECT t.id, t.code, t.name, t.normal_balance, t.sort_order, t.is_active,
       (SELECT count(*) FROM finance.accounts a WHERE a.account_type_id = t.id AND a.deleted_at IS NULL)::int AS account_count
  FROM finance.account_types t
 WHERE t.id = $1;

-- name: UpsertAccountType :exec
INSERT INTO finance.account_types (id, code, name, normal_balance, sort_order, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(by), sqlc.narg(by))
ON CONFLICT (id) DO UPDATE
   SET code = EXCLUDED.code, name = EXCLUDED.name, normal_balance = EXCLUDED.normal_balance,
       sort_order = EXCLUDED.sort_order, is_active = EXCLUDED.is_active,
       updated_by = EXCLUDED.updated_by, updated_at = now();

-- name: DeleteAccountType :exec
DELETE FROM finance.account_types WHERE id = $1;

-- name: ListAccounts :many
SELECT a.id, a.code, a.name, a.parent_id, a.account_type_id, t.code AS type_code, t.name AS type_name,
       t.normal_balance, a.level, a.is_postable, a.is_contra, a.is_cash_bank, a.cash_flow_category,
       a.description, a.is_active, a.created_at, a.updated_at
  FROM finance.accounts a
  JOIN finance.account_types t ON t.id = a.account_type_id
 WHERE a.deleted_at IS NULL
   AND (sqlc.narg(account_type_id)::text IS NULL OR a.account_type_id = sqlc.narg(account_type_id))
   AND (sqlc.narg(is_postable)::boolean IS NULL OR a.is_postable = sqlc.narg(is_postable))
   AND (sqlc.narg(is_cash_bank)::boolean IS NULL OR a.is_cash_bank = sqlc.narg(is_cash_bank))
   AND (sqlc.narg(is_active)::boolean IS NULL OR a.is_active = sqlc.narg(is_active))
   AND (sqlc.narg(cash_flow_category)::text IS NULL OR a.cash_flow_category = sqlc.narg(cash_flow_category))
   AND (sqlc.narg(search)::text IS NULL OR a.code ILIKE '%' || sqlc.narg(search) || '%' OR a.name ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY a.code;

-- name: GetAccount :one
SELECT a.id, a.code, a.name, a.parent_id, a.account_type_id, t.code AS type_code, t.name AS type_name,
       t.normal_balance, a.level, a.is_postable, a.is_contra, a.is_cash_bank, a.cash_flow_category,
       a.description, a.is_active, a.created_at, a.updated_at
  FROM finance.accounts a
  JOIN finance.account_types t ON t.id = a.account_type_id
 WHERE a.id = $1 AND a.deleted_at IS NULL;

-- name: InsertAccount :exec
INSERT INTO finance.accounts (id, code, name, parent_id, account_type_id, level, is_postable, is_contra, is_cash_bank,
                              cash_flow_category, description, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8, $9, $10, $11, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateAccount :exec
UPDATE finance.accounts
   SET code = $2, name = $3, parent_id = $4, account_type_id = $5, level = $6, is_contra = $7, is_cash_bank = $8,
       cash_flow_category = $9, description = $10, is_active = $11, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteAccount :exec
UPDATE finance.accounts
   SET deleted_at = now(), deleted_by = sqlc.narg(by), is_active = false, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: CountChildren :one
SELECT count(*)::int FROM finance.accounts WHERE parent_id = $1 AND deleted_at IS NULL;

-- name: RecomputePostableChain :exec
-- A leaf (no live children) is postable; its ancestors are headers.
WITH RECURSIVE chain AS (
  SELECT a.id, a.parent_id, 1 AS depth FROM finance.accounts a WHERE a.id = $1
  UNION ALL
  SELECT p.id, p.parent_id, c.depth + 1 FROM finance.accounts p JOIN chain c ON p.id = c.parent_id WHERE c.depth < 10
)
UPDATE finance.accounts a
   SET is_postable = NOT EXISTS (SELECT 1 FROM finance.accounts child WHERE child.parent_id = a.id AND child.deleted_at IS NULL),
       updated_at = now()
 WHERE a.id IN (SELECT id FROM chain);

-- name: RecomputeAllPostable :exec
UPDATE finance.accounts a
   SET is_postable = NOT EXISTS (SELECT 1 FROM finance.accounts child WHERE child.parent_id = a.id AND child.deleted_at IS NULL)
 WHERE a.deleted_at IS NULL;

-- name: AccountChecks :many
SELECT id, is_postable, (deleted_at IS NOT NULL)::boolean AS deleted FROM finance.accounts WHERE id = ANY(sqlc.arg(ids)::text[]);

-- name: ListMappings :many
SELECT m.id, m.event_code, m.name, m.description, m.module, m.is_active, m.updated_at
  FROM finance.journal_mappings m
 WHERE m.deleted_at IS NULL
   AND (sqlc.narg(module)::text IS NULL OR m.module = sqlc.narg(module))
   AND (sqlc.narg(is_active)::boolean IS NULL OR m.is_active = sqlc.narg(is_active))
   AND (sqlc.narg(search)::text IS NULL OR m.event_code ILIKE '%' || sqlc.narg(search) || '%' OR m.name ILIKE '%' || sqlc.narg(search) || '%')
 ORDER BY m.module, m.event_code;

-- name: GetMapping :one
SELECT m.id, m.event_code, m.name, m.description, m.module, m.is_active, m.updated_at
  FROM finance.journal_mappings m
 WHERE m.id = $1 AND m.deleted_at IS NULL;

-- name: GetActiveMapping :one
SELECT m.id, m.event_code, m.name, m.description, m.module, m.is_active, m.updated_at
  FROM finance.journal_mappings m
 WHERE m.event_code = $1 AND m.deleted_at IS NULL AND m.is_active;

-- name: MappingLines :many
SELECT l.id, l.mapping_id, l.entry_side, l.line_role, l.account_id, l.amount_source, l.sort_order, l.is_required,
       COALESCE(a.code, '')::text AS account_code, COALESCE(a.name, '')::text AS account_name
  FROM finance.journal_mapping_lines l
  LEFT JOIN finance.accounts a ON a.id = l.account_id AND a.deleted_at IS NULL
 WHERE l.mapping_id = ANY(sqlc.arg(mapping_ids)::text[])
 ORDER BY l.sort_order, l.entry_side;

-- name: InsertMapping :exec
INSERT INTO finance.journal_mappings (id, event_code, name, description, module, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, sqlc.narg(by), sqlc.narg(by));

-- name: UpdateMapping :exec
UPDATE finance.journal_mappings
   SET event_code = $2, name = $3, description = $4, module = $5, is_active = $6, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;

-- name: DeleteMappingLines :exec
DELETE FROM finance.journal_mapping_lines WHERE mapping_id = $1;

-- name: InsertMappingLine :exec
INSERT INTO finance.journal_mapping_lines (id, mapping_id, entry_side, line_role, account_id, amount_source, sort_order, is_required)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: SoftDeleteMapping :exec
UPDATE finance.journal_mappings
   SET deleted_at = now(), deleted_by = sqlc.narg(by), is_active = false, updated_by = sqlc.narg(by), updated_at = now()
 WHERE id = $1 AND deleted_at IS NULL;
