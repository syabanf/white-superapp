-- name: ListJobOpenings :many
SELECT sqlc.embed(j),
       (SELECT count(*) FROM recruitment.candidates c WHERE c.job_opening_id = j.id)::int AS candidate_count,
       (SELECT count(*) FROM recruitment.candidates c WHERE c.job_opening_id = j.id AND c.status = 'hired')::int AS hired_count
FROM recruitment.job_openings j
WHERE j.deleted_at IS NULL
  AND (sqlc.narg(status)::text IS NULL OR j.status = sqlc.narg(status))
  AND (sqlc.narg(search)::text IS NULL OR j.title ILIKE sqlc.narg(search) OR j.department ILIKE sqlc.narg(search))
ORDER BY j.created_at DESC, j.id;

-- name: GetJobOpening :one
SELECT sqlc.embed(j),
       (SELECT count(*) FROM recruitment.candidates c WHERE c.job_opening_id = j.id)::int AS candidate_count,
       (SELECT count(*) FROM recruitment.candidates c WHERE c.job_opening_id = j.id AND c.status = 'hired')::int AS hired_count
FROM recruitment.job_openings j
WHERE j.id = $1 AND j.deleted_at IS NULL;

-- name: GetJobOpeningBySlug :one
SELECT sqlc.embed(j), 0::int AS candidate_count, 0::int AS hired_count
FROM recruitment.job_openings j
WHERE j.slug = $1 AND j.deleted_at IS NULL;

-- name: ListPublicJobs :many
SELECT sqlc.embed(j), 0::int AS candidate_count, 0::int AS hired_count
FROM recruitment.job_openings j
WHERE j.deleted_at IS NULL AND j.status = 'published'
  AND (j.closing_date IS NULL OR j.closing_date >= @today::date)
  AND (sqlc.narg(search)::text IS NULL OR j.title ILIKE sqlc.narg(search) OR j.description ILIKE sqlc.narg(search))
  AND (sqlc.narg(department)::text IS NULL OR j.department = sqlc.narg(department))
ORDER BY j.published_at DESC NULLS LAST, j.title;

-- name: InsertJobOpening :exec
INSERT INTO recruitment.job_openings (
    id, title, slug, department_id, department, location, employment_type, work_mode, headcount,
    description, requirements, benefits, salary_min, salary_max, hiring_manager_user_id, status,
    published_at, closing_date, created_by, updated_by
) VALUES (
    @id, @title, @slug, sqlc.narg(department_id), @department, @location, @employment_type, @work_mode, @headcount,
    sqlc.narg(description), sqlc.narg(requirements), sqlc.narg(benefits), sqlc.narg(salary_min), sqlc.narg(salary_max),
    sqlc.narg(hiring_manager_user_id), @status, sqlc.narg(published_at), sqlc.narg(closing_date), sqlc.narg(created_by), sqlc.narg(created_by)
);

-- name: UpdateJobOpening :execrows
UPDATE recruitment.job_openings SET
    title = @title, slug = @slug, department_id = sqlc.narg(department_id), department = @department,
    location = @location, employment_type = @employment_type, work_mode = @work_mode, headcount = @headcount,
    description = sqlc.narg(description), requirements = sqlc.narg(requirements), benefits = sqlc.narg(benefits),
    salary_min = sqlc.narg(salary_min), salary_max = sqlc.narg(salary_max),
    hiring_manager_user_id = sqlc.narg(hiring_manager_user_id), status = @status,
    published_at = sqlc.narg(published_at), closing_date = sqlc.narg(closing_date),
    updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id AND deleted_at IS NULL;

-- name: DeleteJobOpening :execrows
UPDATE recruitment.job_openings SET deleted_at = now(), updated_by = sqlc.narg(updated_by), updated_at = now()
WHERE id = @id AND deleted_at IS NULL;
