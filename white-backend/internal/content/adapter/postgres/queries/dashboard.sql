-- name: DashboardItems :many
-- Items planned in the window that match the dashboard filters, with the
-- first file of the newest approved design as the thumbnail.
SELECT i.id, i.plan_id, i.planned_date, i.platform, i.format, i.pillar, i.title, i.status, i.revision_round,
    i.copywriter_id, i.designer_id, i.copy_due, i.design_due, i.review_due,
    i.created_at, i.copy_done_at, i.design_done_at, i.approved_at, i.published_at, i.permalink, i.external_ids,
    coalesce((SELECT f.id FROM content.design_tasks t
        JOIN content.design_files f ON f.task_id = t.id AND f.version = t.current_version
        WHERE t.item_id = i.id AND t.status = 'APPROVED' AND t.deleted_at IS NULL
        ORDER BY t.created_at DESC, f.sort_order LIMIT 1), '')::text AS thumbnail_file_id
FROM content.items i
WHERE i.client_id = @client_id AND i.deleted_at IS NULL
  AND i.planned_date BETWEEN @from_day::date AND @to_day::date
  AND (sqlc.narg(platform)::text IS NULL OR i.platform::text = sqlc.narg(platform))
  AND (sqlc.narg(format)::text IS NULL OR i.format::text = sqlc.narg(format))
  AND (sqlc.narg(pillar)::text IS NULL OR i.pillar = sqlc.narg(pillar))
  AND (sqlc.narg(plan_id)::text IS NULL OR i.plan_id = sqlc.narg(plan_id))
  AND (sqlc.narg(assignee_id)::text IS NULL OR i.copywriter_id = sqlc.narg(assignee_id) OR i.designer_id = sqlc.narg(assignee_id))
ORDER BY i.planned_date, i.id;

-- name: DashboardDesignTasks :many
-- Design tasks of items planned in the window (same filters), with the
-- first approval and how many change requests came before it.
SELECT t.id, t.item_id, t.designer_id, t.status, t.current_version, t.created_at,
    fb.approved_at, coalesce(fb.changes_requested, 0)::int AS changes_requested
FROM content.design_tasks t
JOIN content.items i ON i.id = t.item_id AND i.deleted_at IS NULL
LEFT JOIN (
    SELECT task_id, min(created_at) FILTER (WHERE decision = 'APPROVED') AS approved_at,
        count(*) FILTER (WHERE decision = 'CHANGES_REQUESTED') AS changes_requested
    FROM content.design_feedback GROUP BY task_id
) fb ON fb.task_id = t.id
WHERE t.client_id = @client_id AND t.deleted_at IS NULL
  AND i.planned_date BETWEEN @from_day::date AND @to_day::date
  AND (sqlc.narg(platform)::text IS NULL OR i.platform::text = sqlc.narg(platform))
  AND (sqlc.narg(format)::text IS NULL OR i.format::text = sqlc.narg(format))
  AND (sqlc.narg(pillar)::text IS NULL OR i.pillar = sqlc.narg(pillar))
  AND (sqlc.narg(plan_id)::text IS NULL OR i.plan_id = sqlc.narg(plan_id))
  AND (sqlc.narg(assignee_id)::text IS NULL OR i.copywriter_id = sqlc.narg(assignee_id) OR i.designer_id = sqlc.narg(assignee_id) OR t.designer_id = sqlc.narg(assignee_id))
ORDER BY t.created_at, t.id;

-- name: DashboardAssetVersions :many
-- Asset uploads in the window (instants, inclusive days in agency time are
-- resolved by the caller).
SELECT v.asset_id, a.name, a.kind, a.current_version, v.version, v.created_at
FROM content.asset_versions v
JOIN content.assets a ON a.id = v.asset_id AND a.deleted_at IS NULL
WHERE a.client_id = @client_id AND v.created_at >= @from_at AND v.created_at < @to_at
ORDER BY v.created_at, v.id;

-- name: ClientPillars :many
SELECT DISTINCT pillar FROM content.items
WHERE client_id = $1 AND deleted_at IS NULL AND pillar <> ''
ORDER BY pillar;

-- name: ListItemMetrics :many
SELECT * FROM content.item_metrics WHERE item_id = ANY(@item_ids::text[]);

-- name: UpsertItemMetrics :exec
INSERT INTO content.item_metrics (item_id, likes, comments, shares, saves, reach, views, synced_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (item_id) DO UPDATE SET
    likes = EXCLUDED.likes, comments = EXCLUDED.comments, shares = EXCLUDED.shares,
    saves = EXCLUDED.saves, reach = EXCLUDED.reach, views = EXCLUDED.views, synced_at = now();
