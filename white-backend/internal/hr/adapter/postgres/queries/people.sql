-- name: ListDepartments :many
SELECT * FROM hr.departments
WHERE deleted_at IS NULL AND (sqlc.arg(include_inactive)::boolean OR is_active)
ORDER BY name;

-- name: GetDepartment :one
SELECT * FROM hr.departments WHERE id = $1 AND deleted_at IS NULL;

-- name: InsertDepartment :one
INSERT INTO hr.departments (id, code, name, parent_id, head_employee_id, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING *;

-- name: UpdateDepartment :one
UPDATE hr.departments
SET code = $2, name = $3, parent_id = $4, head_employee_id = $5, is_active = $6, updated_by = $7, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteDepartment :execrows
UPDATE hr.departments SET deleted_at = now(), is_active = false, updated_by = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: CountDepartmentEmployees :one
SELECT count(*) FROM hr.employees WHERE department_id = $1 AND deleted_at IS NULL AND is_active;

-- name: ListPositions :many
SELECT sqlc.embed(p), COALESCE(d.name, '')::text AS department_name
FROM hr.positions p
LEFT JOIN hr.departments d ON d.id = p.department_id
WHERE p.deleted_at IS NULL
  AND (sqlc.arg(include_inactive)::boolean OR p.is_active)
  AND (sqlc.arg(department_id)::text = '' OR p.department_id = sqlc.arg(department_id)::text)
ORDER BY d.name NULLS LAST, p.title;

-- name: GetPosition :one
SELECT sqlc.embed(p), COALESCE(d.name, '')::text AS department_name
FROM hr.positions p
LEFT JOIN hr.departments d ON d.id = p.department_id
WHERE p.id = $1 AND p.deleted_at IS NULL;

-- name: InsertPosition :exec
INSERT INTO hr.positions (id, title, department_id, level, salary_min, salary_max, is_active, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8);

-- name: UpdatePosition :execrows
UPDATE hr.positions
SET title = $2, department_id = $3, level = $4, salary_min = $5, salary_max = $6, is_active = $7, updated_by = $8, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeletePosition :execrows
UPDATE hr.positions SET deleted_at = now(), is_active = false, updated_by = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: ListEmploymentStatuses :many
SELECT * FROM hr.employment_statuses ORDER BY sort_order, name;

-- name: UpsertEmploymentStatus :one
INSERT INTO hr.employment_statuses (code, name, is_active, sort_order)
VALUES ($1, $2, $3, $4)
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, sort_order = EXCLUDED.sort_order, updated_at = now()
RETURNING *;

-- name: DeleteEmploymentStatus :execrows
DELETE FROM hr.employment_statuses
WHERE code = $1 AND NOT EXISTS (SELECT 1 FROM hr.employees e WHERE e.employment_status = $1 AND e.deleted_at IS NULL);

-- name: ListEmployees :many
SELECT sqlc.embed(e),
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(p.title, '')::text AS position_name,
       COALESCE(m.full_name, '')::text AS manager_name,
       m.user_id AS manager_user_id,
       COALESCE(s.name, e.employment_status)::text AS employment_status_name,
       count(*) OVER () AS total
FROM hr.employees e
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.employees m ON m.id = e.manager_id
LEFT JOIN hr.employment_statuses s ON s.code = e.employment_status
WHERE e.deleted_at IS NULL
  AND (sqlc.arg(include_inactive)::boolean OR e.is_active)
  AND (sqlc.arg(department_id)::text = '' OR e.department_id = sqlc.arg(department_id)::text)
  AND (sqlc.arg(manager_id)::text = '' OR e.manager_id = sqlc.arg(manager_id)::text)
  AND (cardinality(sqlc.arg(ids)::text[]) = 0 OR e.id = ANY (sqlc.arg(ids)::text[]))
  AND (sqlc.arg(search)::text = '' OR e.full_name ILIKE '%' || sqlc.arg(search)::text || '%'
       OR e.employee_no ILIKE '%' || sqlc.arg(search)::text || '%' OR e.email ILIKE '%' || sqlc.arg(search)::text || '%')
  AND (sqlc.arg(position_id)::text = '' OR e.position_id = sqlc.arg(position_id)::text)
  AND (sqlc.arg(level)::text = '' OR p.level = sqlc.arg(level)::text)
  AND (sqlc.arg(gender)::text = '' OR e.gender = sqlc.arg(gender)::text)
  AND (sqlc.arg(employment_status)::text = '' OR e.employment_status = sqlc.arg(employment_status)::text)
ORDER BY e.full_name
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: GetEmployee :one
SELECT sqlc.embed(e),
       COALESCE(d.name, '')::text AS department_name,
       COALESCE(p.title, '')::text AS position_name,
       COALESCE(m.full_name, '')::text AS manager_name,
       m.user_id AS manager_user_id,
       COALESCE(s.name, e.employment_status)::text AS employment_status_name
FROM hr.employees e
LEFT JOIN hr.departments d ON d.id = e.department_id
LEFT JOIN hr.positions p ON p.id = e.position_id
LEFT JOIN hr.employees m ON m.id = e.manager_id
LEFT JOIN hr.employment_statuses s ON s.code = e.employment_status
WHERE e.id = $1 AND e.deleted_at IS NULL;

-- name: GetEmployeeIDByUser :one
SELECT id FROM hr.employees WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListEmployeeNos :many
SELECT employee_no FROM hr.employees WHERE employee_no LIKE sqlc.arg(prefix)::text || '%';

-- name: InsertEmployee :exec
INSERT INTO hr.employees (
    id, user_id, employee_no, full_name, email, phone, nik, npwp, bpjs_tk_no, bpjs_kes_no, birth_place, birth_date,
    gender, marital_status, religion, address, department_id, position_id, manager_id, employment_status, is_active,
    join_date, end_date, probation_end_date, bank_name, bank_account_no, bank_account_name,
    emergency_contact_name, emergency_contact_phone, emergency_contact_relation, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21,
        $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $31);

-- name: UpdateEmployee :execrows
UPDATE hr.employees SET
    employee_no = $2, full_name = $3, email = $4, phone = $5, nik = $6, npwp = $7, bpjs_tk_no = $8, bpjs_kes_no = $9,
    birth_place = $10, birth_date = $11, gender = $12, marital_status = $13, religion = $14, address = $15,
    department_id = $16, position_id = $17, manager_id = $18, employment_status = $19, is_active = $20,
    join_date = $21, end_date = $22, probation_end_date = $23, bank_name = $24, bank_account_no = $25,
    bank_account_name = $26, emergency_contact_name = $27, emergency_contact_phone = $28,
    emergency_contact_relation = $29, updated_by = $30, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL;

-- name: SetEmployeeStatus :exec
UPDATE hr.employees SET employment_status = $2, is_active = $3, end_date = COALESCE($4, end_date), updated_by = $5, updated_at = now()
WHERE id = $1;

-- name: SoftDeleteEmployee :execrows
UPDATE hr.employees SET deleted_at = now(), is_active = false, user_id = NULL, updated_by = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: SetEmployeeUser :execrows
UPDATE hr.employees SET user_id = $2, updated_by = $3, updated_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: UnlinkUser :execrows
UPDATE hr.employees SET user_id = NULL, updated_at = now() WHERE user_id = $1;

-- name: ListLinkedUserIDsInDepartment :many
SELECT e.user_id::text
FROM hr.employees e
JOIN hr.departments d ON d.id = e.department_id
WHERE lower(d.code) = lower(sqlc.arg(code)::text) AND e.user_id IS NOT NULL AND e.is_active AND e.deleted_at IS NULL;

-- name: InsertHistory :exec
INSERT INTO hr.employment_history (id, employee_id, change_type, notes, prev_employment_status, new_employment_status,
    prev_department_id, new_department_id, prev_position_id, new_position_id, created_by, created_at)
VALUES (sqlc.arg(id), sqlc.arg(employee_id), sqlc.arg(change_type), sqlc.arg(notes), sqlc.narg(prev_employment_status),
        sqlc.narg(new_employment_status), sqlc.narg(prev_department_id), sqlc.narg(new_department_id), sqlc.narg(prev_position_id),
        sqlc.narg(new_position_id), sqlc.narg(created_by), COALESCE(sqlc.narg(created_at)::timestamptz, now()));

-- name: ListHistory :many
SELECT sqlc.embed(h),
       COALESCE(pd.name, '')::text AS prev_department,
       COALESCE(nd.name, '')::text AS new_department,
       COALESCE(pp.title, '')::text AS prev_position,
       COALESCE(np.title, '')::text AS new_position
FROM hr.employment_history h
LEFT JOIN hr.departments pd ON pd.id = h.prev_department_id
LEFT JOIN hr.departments nd ON nd.id = h.new_department_id
LEFT JOIN hr.positions pp ON pp.id = h.prev_position_id
LEFT JOIN hr.positions np ON np.id = h.new_position_id
WHERE h.employee_id = $1
ORDER BY h.created_at DESC;

-- name: InsertDocument :exec
INSERT INTO hr.employee_documents (id, employee_id, doc_type, name, storage_key, content_type, size_bytes, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: ListDocuments :many
SELECT * FROM hr.employee_documents WHERE employee_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: GetDocument :one
SELECT * FROM hr.employee_documents WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteDocument :execrows
UPDATE hr.employee_documents SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: ClaimReminder :execrows
INSERT INTO hr.reminder_sends (kind, ref, day) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;

-- name: InsertRecruitmentHire :execrows
INSERT INTO hr.recruitment_hires (candidate_id, employee_id) VALUES ($1, $2) ON CONFLICT (candidate_id) DO NOTHING;
