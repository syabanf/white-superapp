-- name: DeleteClientPlans :exec
DELETE FROM content.plans WHERE client_id = $1;

-- name: DeleteClientAssets :exec
DELETE FROM content.assets WHERE client_id = $1;

-- name: DeleteClientFolders :exec
DELETE FROM content.asset_folders WHERE client_id = $1;

-- name: DeleteClientReports :exec
DELETE FROM content.reports WHERE client_id = $1;

-- name: DeleteOrphanReminders :exec
DELETE FROM content.reminders r WHERE NOT EXISTS (SELECT 1 FROM content.items i WHERE i.id = r.item_id);

-- name: ForgetItemAssignees :exec
UPDATE content.items SET
    copywriter_id = CASE WHEN copywriter_id = $1 THEN NULL ELSE copywriter_id END,
    designer_id = CASE WHEN designer_id = $1 THEN NULL ELSE designer_id END
WHERE copywriter_id = $1 OR designer_id = $1;

-- name: ForgetDesignAssignee :exec
UPDATE content.design_tasks SET designer_id = NULL WHERE designer_id = $1;
