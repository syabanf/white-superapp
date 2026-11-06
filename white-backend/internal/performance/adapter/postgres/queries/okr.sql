-- name: ListObjectives :many
SELECT * FROM performance.objectives
WHERE period = $1
ORDER BY CASE level WHEN 'company' THEN 0 WHEN 'department' THEN 1 ELSE 2 END, created_at;

-- name: GetObjective :one
SELECT * FROM performance.objectives WHERE id = $1;

-- name: UpsertObjective :exec
INSERT INTO performance.objectives (id, period, level, department_id, employee_id, parent_id, title, description, status, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET parent_id = EXCLUDED.parent_id, title = EXCLUDED.title, description = EXCLUDED.description,
    status = EXCLUDED.status, updated_at = now();

-- name: DeleteObjective :execrows
DELETE FROM performance.objectives WHERE id = $1;

-- name: ListKeyResults :many
SELECT kr.* FROM performance.key_results kr
JOIN performance.objectives o ON o.id = kr.objective_id
WHERE o.period = $1
ORDER BY kr.objective_id, kr.sort_order, kr.title;

-- name: GetKeyResult :one
SELECT * FROM performance.key_results WHERE id = $1;

-- name: UpsertKeyResult :exec
INSERT INTO performance.key_results (id, objective_id, title, metric, start_value, target_value, current_value, unit, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title, metric = EXCLUDED.metric, start_value = EXCLUDED.start_value,
    target_value = EXCLUDED.target_value, unit = EXCLUDED.unit, sort_order = EXCLUDED.sort_order;

-- name: DeleteKeyResult :execrows
DELETE FROM performance.key_results WHERE id = $1;

-- name: SetKeyResultValue :exec
UPDATE performance.key_results SET current_value = $2 WHERE id = $1;

-- name: InsertCheckin :exec
INSERT INTO performance.okr_checkins (id, key_result_id, value, confidence, note, created_by)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListCheckins :many
SELECT * FROM performance.okr_checkins WHERE key_result_id = $1 ORDER BY created_at DESC LIMIT 50;

-- name: LatestConfidence :many
SELECT DISTINCT ON (c.key_result_id) c.key_result_id, c.confidence
FROM performance.okr_checkins c
JOIN performance.key_results kr ON kr.id = c.key_result_id
JOIN performance.objectives o ON o.id = kr.objective_id
WHERE o.period = $1
ORDER BY c.key_result_id, c.created_at DESC;
