-- name: InsertCrmEvent :exec
INSERT INTO sales.crm_events (id, event_type, subject_type, subject_id, payload, actor_user_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CountSubjectEvents :many
SELECT event_type, coalesce(payload->>'sub_type', '')::text AS sub_type, count(*)::int AS n
FROM sales.crm_events
WHERE subject_id = ANY(sqlc.arg('ids')::text[])
  AND (sqlc.narg('since')::timestamptz IS NULL OR created_at >= sqlc.narg('since'))
GROUP BY event_type, coalesce(payload->>'sub_type', '');

-- name: ListWorkflowRules :many
SELECT * FROM sales.workflow_rules
WHERE (sqlc.narg('object')::text IS NULL OR object = sqlc.narg('object'))
  AND (sqlc.narg('trigger')::text IS NULL OR trigger_type = sqlc.narg('trigger'))
  AND (NOT sqlc.arg('active_only')::bool OR is_active)
ORDER BY created_at, id;

-- name: GetWorkflowRule :one
SELECT * FROM sales.workflow_rules WHERE id = $1;

-- name: UpsertWorkflowRule :exec
INSERT INTO sales.workflow_rules (id, name, object, trigger_type, trigger_config, conditions, actions, run_once_per_record, is_active, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, object = EXCLUDED.object, trigger_type = EXCLUDED.trigger_type,
    trigger_config = EXCLUDED.trigger_config, conditions = EXCLUDED.conditions, actions = EXCLUDED.actions,
    run_once_per_record = EXCLUDED.run_once_per_record, is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteWorkflowRule :exec
DELETE FROM sales.workflow_rules WHERE id = $1;

-- name: MarkRuleRun :exec
UPDATE sales.workflow_rules SET run_count = run_count + 1, last_run_at = $2 WHERE id = $1;

-- name: HasRuleRun :one
SELECT EXISTS (SELECT 1 FROM sales.workflow_runs WHERE rule_id = $1 AND subject_id = $2 AND status <> 'skipped'
    AND (sqlc.narg('since')::timestamptz IS NULL OR created_at >= sqlc.narg('since')));

-- name: CountRuleRuns :one
SELECT count(*)::int FROM sales.workflow_runs WHERE rule_id = $1;

-- name: InsertWorkflowRun :exec
INSERT INTO sales.workflow_runs (id, rule_id, event_id, subject_type, subject_id, status, actions_result, error, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateWorkflowRun :exec
UPDATE sales.workflow_runs SET status = $2, actions_result = $3, error = $4 WHERE id = $1;

-- name: ListWorkflowRuns :many
SELECT * FROM sales.workflow_runs WHERE rule_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2;

-- name: InsertScheduledAction :exec
INSERT INTO sales.scheduled_actions (id, rule_id, run_id, subject_type, subject_id, actions, context, run_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: DueScheduledActions :many
UPDATE sales.scheduled_actions SET attempts = attempts + 1
WHERE id IN (SELECT s.id FROM sales.scheduled_actions s WHERE s.status = 'pending' AND s.run_at <= $1 ORDER BY s.run_at LIMIT $2 FOR UPDATE SKIP LOCKED)
RETURNING *;

-- name: FinishScheduledAction :exec
UPDATE sales.scheduled_actions SET status = $2, last_error = $3 WHERE id = $1;

-- name: ListScoringRules :many
SELECT * FROM sales.scoring_rules WHERE (NOT sqlc.arg('active_only')::bool OR is_active) ORDER BY created_at, id;

-- name: UpsertScoringRule :exec
INSERT INTO sales.scoring_rules (id, name, kind, field, operator, value, event_type, window_days, max_count, points, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, kind = EXCLUDED.kind, field = EXCLUDED.field, operator = EXCLUDED.operator,
    value = EXCLUDED.value, event_type = EXCLUDED.event_type, window_days = EXCLUDED.window_days, max_count = EXCLUDED.max_count,
    points = EXCLUDED.points, is_active = EXCLUDED.is_active, updated_at = now();

-- name: DeleteScoringRule :exec
DELETE FROM sales.scoring_rules WHERE id = $1;

-- name: GetForm :one
SELECT * FROM sales.forms WHERE id = $1;

-- name: GetFormBySlug :one
SELECT * FROM sales.forms WHERE slug = $1;

-- name: ListForms :many
SELECT * FROM sales.forms ORDER BY created_at DESC;

-- name: UpsertForm :exec
INSERT INTO sales.forms (id, slug, name, description, fields, submit_label, success_message, redirect_url, default_source,
    lead_temperature, notify_user_ids, is_active, created_by, updated_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
ON CONFLICT (id) DO UPDATE SET slug = EXCLUDED.slug, name = EXCLUDED.name, description = EXCLUDED.description,
    fields = EXCLUDED.fields, submit_label = EXCLUDED.submit_label, success_message = EXCLUDED.success_message,
    redirect_url = EXCLUDED.redirect_url, default_source = EXCLUDED.default_source, lead_temperature = EXCLUDED.lead_temperature,
    notify_user_ids = EXCLUDED.notify_user_ids, is_active = EXCLUDED.is_active, updated_by = EXCLUDED.updated_by,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteForm :exec
DELETE FROM sales.forms WHERE id = $1;

-- name: InsertFormSubmission :exec
INSERT INTO sales.form_submissions (id, form_id, lead_id, payload, utm, ip_hash, status, reason, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: IncrementFormSubmissions :exec
UPDATE sales.forms SET submission_count = submission_count + 1 WHERE id = $1;

-- name: ListFormSubmissions :many
SELECT * FROM sales.form_submissions WHERE form_id = $1 ORDER BY created_at DESC LIMIT $2;
