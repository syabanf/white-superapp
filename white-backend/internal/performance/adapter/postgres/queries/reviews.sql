-- name: ListReviewCycles :many
SELECT c.*,
       count(r.id)::int AS total_reviews,
       count(r.id) FILTER (WHERE r.status = 'final')::int AS final_reviews,
       count(r.id) FILTER (WHERE r.total_behavioral_score > 0)::int AS rated_reviews
FROM performance.review_cycles c
LEFT JOIN performance.performance_reviews r ON r.cycle_id = c.id
GROUP BY c.id
ORDER BY c.period_year DESC, c.period_quarter DESC;

-- name: GetReviewCycle :one
SELECT * FROM performance.review_cycles WHERE id = $1;

-- name: ReviewCycleExists :one
SELECT EXISTS (SELECT 1 FROM performance.review_cycles WHERE period_year = $1 AND period_quarter = $2);

-- name: InsertReviewCycle :exec
INSERT INTO performance.review_cycles (id, name, period_year, period_quarter, start_date, end_date, due_date, status, created_by, created_by_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: SetReviewCycleStatus :exec
UPDATE performance.review_cycles SET status = $2, updated_at = now() WHERE id = $1;

-- name: ListBehavioralStandards :many
SELECT * FROM performance.behavioral_standards WHERE is_active ORDER BY sort_order, created_at;

-- name: ListPerformanceCategories :many
SELECT * FROM performance.performance_categories ORDER BY min_score DESC;

-- name: InsertReview :exec
INSERT INTO performance.performance_reviews (id, cycle_id, employee_id, department_id, period_label, start_date, end_date, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'draft');

-- name: InsertReviewItem :exec
INSERT INTO performance.behavioral_review_items (id, review_id, value_name, competency, behavioral_standard,
  score_1_description, score_2_description, score_3_description, score_4_description, score_5_description, weight, item_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ListReviewsByCycle :many
SELECT r.*, c.name AS cycle_name, c.status AS cycle_status, c.due_date AS cycle_due_date,
       c.period_year AS cycle_year, c.period_quarter AS cycle_quarter,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id AND i.score IS NOT NULL)::int AS rated_items,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id)::int AS total_items
FROM performance.performance_reviews r
JOIN performance.review_cycles c ON c.id = r.cycle_id
WHERE r.cycle_id = $1
ORDER BY r.department_id NULLS LAST, r.employee_id;

-- name: ListOpenCycleReviews :many
SELECT r.*, c.name AS cycle_name, c.status AS cycle_status, c.due_date AS cycle_due_date,
       c.period_year AS cycle_year, c.period_quarter AS cycle_quarter,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id AND i.score IS NOT NULL)::int AS rated_items,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id)::int AS total_items
FROM performance.performance_reviews r
JOIN performance.review_cycles c ON c.id = r.cycle_id
WHERE c.status = 'open' AND r.status = 'draft';

-- name: GetReview :one
SELECT r.*, c.name AS cycle_name, c.status AS cycle_status, c.due_date AS cycle_due_date,
       c.period_year AS cycle_year, c.period_quarter AS cycle_quarter,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id AND i.score IS NOT NULL)::int AS rated_items,
       (SELECT count(*) FROM performance.behavioral_review_items i WHERE i.review_id = r.id)::int AS total_items
FROM performance.performance_reviews r
JOIN performance.review_cycles c ON c.id = r.cycle_id
WHERE r.id = $1;

-- name: UpdateReview :exec
UPDATE performance.performance_reviews SET
  total_work_result_score = $2, total_behavioral_score = $3, total_project_score = $4, grand_total_score = $5,
  category = $6, status = $7, self_assessment = $8, reviewer_id = $9, reviewer_name = $10, reviewer_notes = $11,
  manager_id = $12, manager_notes = $13, employee_sign_date = $14, reviewer_sign_date = $15,
  finalized_by = $16, finalized_at = $17, updated_at = now()
WHERE id = $1;

-- name: ListReviewItems :many
SELECT * FROM performance.behavioral_review_items WHERE review_id = $1 ORDER BY item_order, value_name;

-- name: UpdateReviewItem :exec
UPDATE performance.behavioral_review_items SET score = $2, weighted_score = $3, notes = $4, updated_at = now() WHERE id = $1;
