-- name: ListFeedbackCategories :many
SELECT * FROM performance.feedback_categories WHERE is_active ORDER BY display_order;

-- name: ListFeedbackCriteria :many
SELECT cr.* FROM performance.feedback_criteria cr
JOIN performance.feedback_categories c ON c.id = cr.category_id
WHERE c.is_active
ORDER BY c.display_order, cr.display_order;

-- name: ListFeedbackCycles :many
SELECT c.*,
  (SELECT count(*) FROM performance.feedback_assignments a WHERE a.cycle_id = c.id)::int AS assignments_count,
  (SELECT count(*) FROM performance.feedback_assignments a WHERE a.cycle_id = c.id AND a.status IN ('submitted', 'approved'))::int AS submitted_count,
  (SELECT count(*) FROM performance.feedback_summaries s WHERE s.cycle_id = c.id)::int AS summaries_count
FROM performance.feedback_cycles c
WHERE sqlc.arg(status)::text = '' OR c.status = sqlc.arg(status)::text
ORDER BY c.created_at DESC;

-- name: GetFeedbackCycle :one
SELECT c.*,
  (SELECT count(*) FROM performance.feedback_assignments a WHERE a.cycle_id = c.id)::int AS assignments_count,
  (SELECT count(*) FROM performance.feedback_assignments a WHERE a.cycle_id = c.id AND a.status IN ('submitted', 'approved'))::int AS submitted_count,
  (SELECT count(*) FROM performance.feedback_summaries s WHERE s.cycle_id = c.id)::int AS summaries_count
FROM performance.feedback_cycles c WHERE c.id = $1;

-- name: InsertFeedbackCycle :exec
INSERT INTO performance.feedback_cycles (id, name, period_label, start_date, end_date, status, kpi_weight, feedback_weight,
  is_anonymous, allow_self_assessment, require_manager_review, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12);

-- name: UpdateFeedbackCycle :exec
UPDATE performance.feedback_cycles SET name = $2, period_label = $3, start_date = $4, end_date = $5, status = $6,
  kpi_weight = $7, feedback_weight = $8, is_anonymous = $9, allow_self_assessment = $10, require_manager_review = $11,
  updated_by = $12, updated_at = now()
WHERE id = $1;

-- name: DeleteFeedbackCycle :exec
DELETE FROM performance.feedback_cycles WHERE id = $1;

-- name: InsertAssignment :execrows
INSERT INTO performance.feedback_assignments (id, cycle_id, employee_id, reviewer_id, relationship_type, status, due_date, created_by)
VALUES ($1, $2, $3, $4, $5, 'pending', $6, $7)
ON CONFLICT (cycle_id, employee_id, reviewer_id) DO NOTHING;

-- name: ListAssignments :many
SELECT a.*, c.name AS cycle_name, c.status AS cycle_status
FROM performance.feedback_assignments a
JOIN performance.feedback_cycles c ON c.id = a.cycle_id
WHERE (sqlc.arg(cycle_id)::text = '' OR a.cycle_id = sqlc.arg(cycle_id)::text)
  AND (sqlc.arg(employee_id)::text = '' OR a.employee_id = sqlc.arg(employee_id)::text)
  AND (sqlc.arg(reviewer_id)::text = '' OR a.reviewer_id = sqlc.arg(reviewer_id)::text)
  AND (sqlc.arg(status)::text = '' OR a.status = sqlc.arg(status)::text)
ORDER BY a.due_date NULLS LAST, a.created_at DESC
LIMIT 1000;

-- name: ListRemindableAssignments :many
SELECT a.*, c.name AS cycle_name, c.status AS cycle_status
FROM performance.feedback_assignments a
JOIN performance.feedback_cycles c ON c.id = a.cycle_id
WHERE c.status = 'active' AND a.status IN ('pending', 'in_progress', 'rejected')
ORDER BY a.reviewer_id, a.due_date NULLS LAST;

-- name: GetAssignment :one
SELECT a.*, c.name AS cycle_name, c.status AS cycle_status
FROM performance.feedback_assignments a
JOIN performance.feedback_cycles c ON c.id = a.cycle_id
WHERE a.id = $1;

-- name: UpdateAssignment :exec
UPDATE performance.feedback_assignments SET relationship_type = $2, status = $3, due_date = $4, started_at = $5,
  submitted_at = $6, reminder_count = $7, last_reminder_at = $8, manager_comments = $9, rejection_reason = $10,
  approved_by = $11, approved_at = $12, updated_at = now()
WHERE id = $1;

-- name: DeleteAssignment :exec
DELETE FROM performance.feedback_assignments WHERE id = $1;

-- name: UpsertResponse :exec
INSERT INTO performance.feedback_responses (id, assignment_id, criteria_id, rating, comments)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (assignment_id, criteria_id) DO UPDATE SET rating = EXCLUDED.rating, comments = EXCLUDED.comments, updated_at = now();

-- name: ListResponses :many
SELECT * FROM performance.feedback_responses WHERE assignment_id = $1 ORDER BY created_at;

-- name: CycleRatings :many
SELECT a.employee_id, a.status AS assignment_status, a.relationship_type, c.code AS category_code, c.name AS category_name,
       cr.name AS criteria_name, r.rating, r.comments
FROM performance.feedback_responses r
JOIN performance.feedback_assignments a ON a.id = r.assignment_id
JOIN performance.feedback_criteria cr ON cr.id = r.criteria_id
JOIN performance.feedback_categories c ON c.id = cr.category_id
WHERE a.cycle_id = $1
ORDER BY a.employee_id;

-- name: UpsertFeedbackSummary :exec
INSERT INTO performance.feedback_summaries (id, cycle_id, employee_id, leadership_score, communication_score, collaboration_score,
  accountability_score, problem_solving_score, overall_360_score, kpi_score, final_score, final_grade, strengths, weaknesses,
  responses_count, generated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, now())
ON CONFLICT (cycle_id, employee_id) DO UPDATE SET
  leadership_score = EXCLUDED.leadership_score, communication_score = EXCLUDED.communication_score,
  collaboration_score = EXCLUDED.collaboration_score, accountability_score = EXCLUDED.accountability_score,
  problem_solving_score = EXCLUDED.problem_solving_score, overall_360_score = EXCLUDED.overall_360_score,
  kpi_score = EXCLUDED.kpi_score, final_score = EXCLUDED.final_score, final_grade = EXCLUDED.final_grade,
  strengths = EXCLUDED.strengths, weaknesses = EXCLUDED.weaknesses, responses_count = EXCLUDED.responses_count,
  generated_at = now(), updated_at = now()
WHERE NOT performance.feedback_summaries.is_locked;

-- name: ListFeedbackSummaries :many
SELECT * FROM performance.feedback_summaries
WHERE (sqlc.arg(cycle_id)::text = '' OR cycle_id = sqlc.arg(cycle_id)::text)
  AND (sqlc.arg(employee_id)::text = '' OR employee_id = sqlc.arg(employee_id)::text)
ORDER BY final_score DESC NULLS LAST;

-- name: GetFeedbackSummary :one
SELECT * FROM performance.feedback_summaries WHERE id = $1;

-- name: UpdateFeedbackSummary :exec
UPDATE performance.feedback_summaries SET strengths = $2, weaknesses = $3, manager_comments = $4, is_locked = $5, updated_at = now()
WHERE id = $1;
