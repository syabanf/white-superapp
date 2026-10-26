-- name: GetDeal :one
SELECT sqlc.embed(d), s.code AS stage_code, s.name AS stage_name, s.is_won, s.is_lost, s.probability,
       coalesce(l.org_name, '')::text AS org_name, coalesce(l.pic_name, '')::text AS pic_name,
       coalesce(l.pic_phone, '')::text AS pic_phone, coalesce(l.pic_email, '')::text AS pic_email,
       coalesce(a.name, '')::text AS account_name
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE d.id = $1 AND d.deleted_at IS NULL;

-- name: ListDeals :many
SELECT sqlc.embed(d), s.code AS stage_code, s.name AS stage_name, s.is_won, s.is_lost, s.probability,
       coalesce(l.org_name, '')::text AS org_name, coalesce(l.pic_name, '')::text AS pic_name,
       coalesce(l.pic_phone, '')::text AS pic_phone, coalesce(l.pic_email, '')::text AS pic_email,
       coalesce(a.name, '')::text AS account_name
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE d.deleted_at IS NULL
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('stage')::text IS NULL OR d.stage_id = sqlc.narg('stage'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('lead')::text IS NULL OR d.lead_id = sqlc.narg('lead'))
  AND (sqlc.narg('account')::text IS NULL OR d.account_id = sqlc.narg('account'))
  AND (sqlc.narg('status')::text IS NULL
       OR (sqlc.narg('status') = 'open' AND NOT s.is_won AND NOT s.is_lost)
       OR (sqlc.narg('status') = 'won' AND s.is_won)
       OR (sqlc.narg('status') = 'lost' AND s.is_lost))
  AND (sqlc.narg('q')::text IS NULL OR d.title ILIKE '%' || sqlc.narg('q') || '%' OR l.org_name ILIKE '%' || sqlc.narg('q') || '%' OR a.name ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('industry')::text IS NULL OR coalesce(nullif(a.industry, ''), 'Lainnya') = sqlc.narg('industry'))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
  AND (sqlc.narg('lost_reason')::text IS NULL OR coalesce(d.lost_reason_id, 'none') = sqlc.narg('lost_reason'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR (CASE WHEN sqlc.arg('by_closed')::bool THEN d.closed_at ELSE d.created_at END) >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR (CASE WHEN sqlc.arg('by_closed')::bool THEN d.closed_at ELSE d.created_at END) < sqlc.narg('to_at'))
ORDER BY d.updated_at DESC, d.id DESC
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: CountDeals :one
SELECT count(*)
FROM sales.deals d
JOIN sales.stages s ON s.id = d.stage_id
LEFT JOIN sales.leads l ON l.id = d.lead_id
LEFT JOIN sales.accounts a ON a.id = d.account_id
WHERE d.deleted_at IS NULL
  AND (sqlc.narg('pipeline')::text IS NULL OR d.pipeline_id = sqlc.narg('pipeline'))
  AND (sqlc.narg('stage')::text IS NULL OR d.stage_id = sqlc.narg('stage'))
  AND (sqlc.narg('owner')::text IS NULL OR d.owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('lead')::text IS NULL OR d.lead_id = sqlc.narg('lead'))
  AND (sqlc.narg('account')::text IS NULL OR d.account_id = sqlc.narg('account'))
  AND (sqlc.narg('status')::text IS NULL
       OR (sqlc.narg('status') = 'open' AND NOT s.is_won AND NOT s.is_lost)
       OR (sqlc.narg('status') = 'won' AND s.is_won)
       OR (sqlc.narg('status') = 'lost' AND s.is_lost))
  AND (sqlc.narg('q')::text IS NULL OR d.title ILIKE '%' || sqlc.narg('q') || '%' OR l.org_name ILIKE '%' || sqlc.narg('q') || '%' OR a.name ILIKE '%' || sqlc.narg('q') || '%')
  AND (sqlc.narg('line')::text IS NULL OR sqlc.narg('line')::text = ANY(d.service_lines))
  AND (sqlc.narg('industry')::text IS NULL OR coalesce(nullif(a.industry, ''), 'Lainnya') = sqlc.narg('industry'))
  AND (sqlc.narg('source')::text IS NULL OR l.source = sqlc.narg('source'))
  AND (sqlc.narg('lost_reason')::text IS NULL OR coalesce(d.lost_reason_id, 'none') = sqlc.narg('lost_reason'))
  AND (sqlc.narg('from_at')::timestamptz IS NULL OR (CASE WHEN sqlc.arg('by_closed')::bool THEN d.closed_at ELSE d.created_at END) >= sqlc.narg('from_at'))
  AND (sqlc.narg('to_at')::timestamptz IS NULL OR (CASE WHEN sqlc.arg('by_closed')::bool THEN d.closed_at ELSE d.created_at END) < sqlc.narg('to_at'));

-- name: UpsertDeal :exec
INSERT INTO sales.deals (id, lead_id, account_id, title, service_lines, pipeline_id, stage_id, value_estimate, value_final,
    owner_user_id, lost_reason_id, expected_close_date, notes, forecast_category, entered_stage_at, closed_at, custom,
    created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (id) DO UPDATE SET lead_id = EXCLUDED.lead_id, account_id = EXCLUDED.account_id, title = EXCLUDED.title,
    service_lines = EXCLUDED.service_lines, pipeline_id = EXCLUDED.pipeline_id, stage_id = EXCLUDED.stage_id,
    value_estimate = EXCLUDED.value_estimate, value_final = EXCLUDED.value_final, owner_user_id = EXCLUDED.owner_user_id,
    lost_reason_id = EXCLUDED.lost_reason_id, expected_close_date = EXCLUDED.expected_close_date, notes = EXCLUDED.notes,
    forecast_category = EXCLUDED.forecast_category, entered_stage_at = EXCLUDED.entered_stage_at,
    closed_at = EXCLUDED.closed_at, custom = EXCLUDED.custom, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteDeal :exec
UPDATE sales.deals SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: ListDealMembers :many
SELECT * FROM sales.deal_members WHERE deal_id = $1 ORDER BY split_percent DESC, user_id;

-- name: DeleteDealMembers :exec
DELETE FROM sales.deal_members WHERE deal_id = $1;

-- name: InsertDealMember :exec
INSERT INTO sales.deal_members (deal_id, user_id, role, split_percent) VALUES ($1, $2, $3, $4);

-- name: InsertStageHistory :exec
INSERT INTO sales.deal_stage_history (id, deal_id, stage_id, entered_at, moved_by) VALUES ($1, $2, $3, $4, $5);

-- name: ListStageHistory :many
SELECT h.id, h.deal_id, h.stage_id, s.name AS stage_name, h.entered_at, h.moved_by
FROM sales.deal_stage_history h JOIN sales.stages s ON s.id = h.stage_id
WHERE h.deal_id = $1 ORDER BY h.entered_at, h.id;

-- name: ForecastDeals :many
SELECT d.owner_user_id, coalesce(d.value_final, d.value_estimate, 0)::numeric AS value, s.probability,
       (CASE WHEN s.is_won THEN 'closed_won' WHEN s.is_lost THEN 'closed_lost' ELSE d.forecast_category END)::text AS category
FROM sales.deals d JOIN sales.stages s ON s.id = d.stage_id
WHERE d.deleted_at IS NULL
  AND ((s.is_won AND d.closed_at >= sqlc.arg('from_at') AND d.closed_at < sqlc.arg('to_at'))
       OR (NOT s.is_won AND NOT s.is_lost
           AND coalesce(d.expected_close_date, (d.created_at AT TIME ZONE 'Asia/Jakarta')::date) >= sqlc.arg('from_day')::date
           AND coalesce(d.expected_close_date, (d.created_at AT TIME ZONE 'Asia/Jakarta')::date) < sqlc.arg('to_day')::date));

-- name: StaleDealIDs :many
SELECT d.id FROM sales.deals d JOIN sales.stages s ON s.id = d.stage_id
WHERE d.deleted_at IS NULL AND NOT s.is_won AND NOT s.is_lost AND d.updated_at < $1 LIMIT 200;

-- name: GetActivity :one
SELECT * FROM sales.activities WHERE id = $1 AND deleted_at IS NULL;

-- name: ListActivities :many
SELECT * FROM sales.activities
WHERE deleted_at IS NULL
  AND (sqlc.narg('lead')::text IS NULL OR lead_id = sqlc.narg('lead'))
  AND (sqlc.narg('deal')::text IS NULL OR deal_id = sqlc.narg('deal'))
  AND (sqlc.narg('owner')::text IS NULL OR owner_user_id = sqlc.narg('owner'))
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (NOT sqlc.arg('open_only')::bool OR status IN ('open', 'in_progress'))
  AND (sqlc.narg('due_before')::timestamptz IS NULL OR due_at < sqlc.narg('due_before'))
ORDER BY (status IN ('open', 'in_progress')) DESC, due_at NULLS LAST, created_at DESC
LIMIT sqlc.arg('lim');

-- name: UpsertActivity :exec
INSERT INTO sales.activities (id, lead_id, deal_id, subject_type, subject_id, activity_type, title, notes, priority, status,
    due_at, done_at, reminder_at, reminder_sent_at, recurrence, parent_task_id, owner_user_id, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
ON CONFLICT (id) DO UPDATE SET lead_id = EXCLUDED.lead_id, deal_id = EXCLUDED.deal_id, subject_type = EXCLUDED.subject_type,
    subject_id = EXCLUDED.subject_id, activity_type = EXCLUDED.activity_type, title = EXCLUDED.title, notes = EXCLUDED.notes,
    priority = EXCLUDED.priority, status = EXCLUDED.status, due_at = EXCLUDED.due_at, done_at = EXCLUDED.done_at,
    reminder_at = EXCLUDED.reminder_at, reminder_sent_at = EXCLUDED.reminder_sent_at, recurrence = EXCLUDED.recurrence,
    owner_user_id = EXCLUDED.owner_user_id, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at;

-- name: DeleteActivity :exec
UPDATE sales.activities SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: DueReminders :many
SELECT * FROM sales.activities
WHERE deleted_at IS NULL AND reminder_sent_at IS NULL AND reminder_at IS NOT NULL AND reminder_at <= $1
  AND status IN ('open', 'in_progress')
ORDER BY reminder_at LIMIT $2;

-- name: ClaimReminder :execrows
UPDATE sales.activities SET reminder_sent_at = $2 WHERE id = $1 AND reminder_sent_at IS NULL;

-- name: TasksDueBetween :many
SELECT id FROM sales.activities
WHERE deleted_at IS NULL AND status IN ('open', 'in_progress') AND due_at >= $1 AND due_at < $2;
