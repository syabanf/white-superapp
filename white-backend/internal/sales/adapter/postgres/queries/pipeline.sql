-- name: NextNumber :one
INSERT INTO sales.number_sequences (prefix, last_value) VALUES ($1, 1)
ON CONFLICT (prefix) DO UPDATE SET last_value = sales.number_sequences.last_value + 1
RETURNING last_value;

-- name: ListPipelines :many
SELECT * FROM sales.pipelines ORDER BY sort_order, name;

-- name: ListStages :many
SELECT * FROM sales.stages ORDER BY pipeline_id, sort_order, name;

-- name: UpsertPipeline :exec
INSERT INTO sales.pipelines (id, code, name, is_default, sort_order)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, is_default = EXCLUDED.is_default,
    sort_order = EXCLUDED.sort_order, updated_at = now();

-- name: ClearDefaultPipeline :exec
UPDATE sales.pipelines SET is_default = false WHERE id <> $1 AND is_default;

-- name: GetStage :one
SELECT * FROM sales.stages WHERE id = $1;

-- name: UpsertStage :exec
INSERT INTO sales.stages (id, pipeline_id, code, name, sort_order, probability, is_won, is_lost, stuck_threshold_days, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, sort_order = EXCLUDED.sort_order,
    probability = EXCLUDED.probability, is_won = EXCLUDED.is_won, is_lost = EXCLUDED.is_lost,
    stuck_threshold_days = EXCLUDED.stuck_threshold_days, is_active = EXCLUDED.is_active, updated_at = now();

-- name: ListLostReasons :many
SELECT * FROM sales.lost_reasons WHERE is_active ORDER BY sort_order, name;
