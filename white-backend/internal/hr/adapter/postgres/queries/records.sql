-- name: ListContracts :many
SELECT sqlc.embed(c), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS employee_department
FROM hr.employment_contracts c
JOIN hr.employees e ON e.id = c.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE (sqlc.arg(employee_id)::text = '' OR c.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(status)::text = '' OR c.status = sqlc.arg(status)::text)
  AND (sqlc.arg(contract_type)::text = '' OR c.contract_type = sqlc.arg(contract_type)::text)
  AND (sqlc.arg(search)::text = '' OR e.full_name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR c.contract_number ILIKE '%' || sqlc.arg(search)::text || '%')
ORDER BY c.end_date NULLS LAST, c.created_at DESC
LIMIT 500;

-- name: GetContract :one
SELECT sqlc.embed(c), e.full_name, e.employee_no, COALESCE(d.name, '')::text AS employee_department
FROM hr.employment_contracts c
JOIN hr.employees e ON e.id = c.employee_id
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE c.id = $1;

-- name: CountContractsOfType :one
SELECT count(*) FROM hr.employment_contracts
WHERE contract_type = $1 AND extract(year FROM created_at)::int = sqlc.arg(year)::int;

-- name: InsertContract :exec
INSERT INTO hr.employment_contracts (id, contract_number, employee_id, contract_type, status, start_date, end_date,
    probation_end_date, parent_contract_id, sequence, position_title, department_name, work_location, base_salary,
    notes, created_by, updated_by)
VALUES ($1, $2, $3, $4, 'draft', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $15);

-- name: UpdateDraftContract :execrows
UPDATE hr.employment_contracts SET contract_type = $2, start_date = $3, end_date = $4, probation_end_date = $5,
    position_title = $6, department_name = $7, work_location = $8, base_salary = $9, notes = $10,
    updated_by = $11, updated_at = now()
WHERE id = $1 AND status = 'draft';

-- name: SetContractStatus :execrows
UPDATE hr.employment_contracts SET status = sqlc.arg(status),
    end_date = COALESCE(sqlc.narg(end_date), end_date),
    signed_at = COALESCE(sqlc.narg(signed_at), signed_at),
    compensation_amount = COALESCE(sqlc.narg(compensation_amount), compensation_amount),
    terminated_reason = COALESCE(sqlc.narg(terminated_reason), terminated_reason),
    updated_by = sqlc.arg(actor), updated_at = now()
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status);

-- name: ActiveContractNumber :one
SELECT contract_number FROM hr.employment_contracts WHERE employee_id = $1 AND status = 'active' AND id <> $2;

-- name: PkwtChain :many
SELECT start_date, end_date FROM hr.employment_contracts
WHERE employee_id = $1 AND contract_type = 'pkwt' AND status IN ('active', 'ended');

-- name: DeleteDraftContract :execrows
DELETE FROM hr.employment_contracts WHERE id = $1 AND status = 'draft';

-- name: ActiveEmployeesWithoutContract :many
SELECT e.id, e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name
FROM hr.employees e
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE e.is_active AND e.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM hr.employment_contracts c WHERE c.employee_id = e.id AND c.status = 'active')
ORDER BY e.full_name;

-- name: ProbationsEnding :many
SELECT e.id, e.full_name, e.employee_no, COALESCE(d.name, '')::text AS department_name, e.probation_end_date
FROM hr.employees e
LEFT JOIN hr.departments d ON d.id = e.department_id
WHERE e.is_active AND e.deleted_at IS NULL AND e.probation_end_date IS NOT NULL
  AND e.probation_end_date BETWEEN sqlc.arg(date_from) AND sqlc.arg(date_to)
ORDER BY e.probation_end_date;

-- name: ActiveBirthdays :many
SELECT e.id, e.full_name, e.birth_date FROM hr.employees e
WHERE e.is_active AND e.deleted_at IS NULL AND e.birth_date IS NOT NULL
ORDER BY e.full_name;

-- name: ListAnnouncements :many
SELECT sqlc.embed(a),
       COALESCE((SELECT array_agg(ad.department_id) FROM hr.announcement_departments ad WHERE ad.announcement_id = a.id), '{}')::text[] AS department_ids,
       (SELECT count(*) FROM hr.announcement_reads r WHERE r.announcement_id = a.id)::int AS read_count
FROM hr.announcements a
WHERE a.deleted_at IS NULL AND (sqlc.arg(status)::text = '' OR a.status = sqlc.arg(status)::text)
ORDER BY a.is_pinned DESC, COALESCE(a.publish_at, a.created_at) DESC;

-- name: GetAnnouncement :one
SELECT sqlc.embed(a),
       COALESCE((SELECT array_agg(ad.department_id) FROM hr.announcement_departments ad WHERE ad.announcement_id = a.id), '{}')::text[] AS department_ids,
       (SELECT count(*) FROM hr.announcement_reads r WHERE r.announcement_id = a.id)::int AS read_count
FROM hr.announcements a
WHERE a.id = $1 AND a.deleted_at IS NULL;

-- name: AnnouncementFeed :many
SELECT sqlc.embed(a), (r.employee_id IS NOT NULL)::boolean AS is_read
FROM hr.announcements a
JOIN hr.employees e ON e.id = sqlc.arg(employee_id)::text
LEFT JOIN hr.announcement_reads r ON r.announcement_id = a.id AND r.employee_id = e.id
WHERE a.deleted_at IS NULL AND a.status = 'published'
  AND (a.publish_at IS NULL OR a.publish_at <= sqlc.arg(now)::timestamptz)
  AND (a.expires_at IS NULL OR a.expires_at > sqlc.arg(now)::timestamptz)
  AND (a.target_scope = 'global' OR EXISTS (
        SELECT 1 FROM hr.announcement_departments ad WHERE ad.announcement_id = a.id AND ad.department_id = e.department_id))
ORDER BY a.is_pinned DESC, COALESCE(a.publish_at, a.created_at) DESC
LIMIT 100;

-- name: InsertAnnouncement :exec
INSERT INTO hr.announcements (id, title, body, tags, status, is_pinned, target_scope, publish_at, expires_at, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10);

-- name: UpdateAnnouncement :execrows
UPDATE hr.announcements SET title = $2, body = $3, tags = $4, status = $5, is_pinned = $6, target_scope = $7,
    publish_at = $8, expires_at = $9, updated_by = $10, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ClearAnnouncementDepartments :exec
DELETE FROM hr.announcement_departments WHERE announcement_id = $1;

-- name: AddAnnouncementDepartment :exec
INSERT INTO hr.announcement_departments (announcement_id, department_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: SoftDeleteAnnouncement :execrows
UPDATE hr.announcements SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: MarkAnnouncementRead :exec
INSERT INTO hr.announcement_reads (announcement_id, employee_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListOnboardingTasks :many
SELECT sqlc.embed(t), COALESCE(c.full_name, '')::text AS completed_by_name
FROM hr.onboarding_tasks t
LEFT JOIN hr.employees c ON c.id = t.completed_by
WHERE t.employee_id = $1
ORDER BY t.is_completed, t.priority, t.due_date NULLS LAST, t.created_at;

-- name: GetOnboardingTask :one
SELECT sqlc.embed(t), COALESCE(c.full_name, '')::text AS completed_by_name
FROM hr.onboarding_tasks t
LEFT JOIN hr.employees c ON c.id = t.completed_by
WHERE t.id = $1;

-- name: InsertOnboardingTask :exec
INSERT INTO hr.onboarding_tasks (id, employee_id, task_name, category, priority, due_date, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateOnboardingTask :execrows
UPDATE hr.onboarding_tasks SET task_name = $2, due_date = $3, is_completed = $4,
    completed_at = CASE WHEN $4 AND NOT is_completed THEN now() WHEN NOT $4 THEN NULL ELSE completed_at END,
    completed_by = CASE WHEN $4 AND NOT is_completed THEN sqlc.narg(completed_by) WHEN NOT $4 THEN NULL ELSE completed_by END,
    notes = $5, updated_at = now()
WHERE id = $1;

-- name: DeleteOnboardingTask :execrows
DELETE FROM hr.onboarding_tasks WHERE id = $1;

-- name: GetOffboarding :one
SELECT * FROM hr.offboarding WHERE employee_id = $1;

-- name: InsertOffboarding :exec
INSERT INTO hr.offboarding (id, employee_id, resignation_type, resignation_date, last_working_day, reason, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7);

-- name: UpdateOffboarding :exec
UPDATE hr.offboarding SET status = $2, exit_interview_notes = $3, clearance_hr = $4, clearance_it = $5,
    clearance_finance = $6, clearance_manager = $7, notes = $8, completed_at = $9, updated_by = $10, updated_at = now()
WHERE employee_id = $1;
