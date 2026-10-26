-- +goose Up
-- Same rows as 00003, with the creation stage joined once per candidate
-- instead of a correlated lookup.
CREATE OR REPLACE VIEW recruitment.candidate_stage_events AS
WITH moves AS (
    SELECT a.candidate_id, a.created_at, a.id,
           split_part(split_part(a.description, ': ', 2), ' → ', 1) AS from_label,
           split_part(a.description, ' → ', 2) AS to_label
    FROM recruitment.candidate_activities a
    WHERE a.activity_type = 'status_change'
),
labelled AS (
    SELECT m.candidate_id, m.created_at, m.id,
           CASE m.from_label WHEN 'Applied' THEN 'applied' WHEN 'Screening' THEN 'screening' WHEN 'Psikotes' THEN 'psikotes'
               WHEN 'Interview' THEN 'interview' WHEN 'Offer' THEN 'offer' WHEN 'Hired' THEN 'hired' WHEN 'Talent Pool' THEN 'talent_pool'
               WHEN 'Tolak' THEN 'rejected' WHEN 'Diarsipkan' THEN 'archived' END AS from_status,
           CASE m.to_label WHEN 'Applied' THEN 'applied' WHEN 'Screening' THEN 'screening' WHEN 'Psikotes' THEN 'psikotes'
               WHEN 'Interview' THEN 'interview' WHEN 'Offer' THEN 'offer' WHEN 'Hired' THEN 'hired' WHEN 'Talent Pool' THEN 'talent_pool'
               WHEN 'Tolak' THEN 'rejected' WHEN 'Diarsipkan' THEN 'archived' END AS to_status
    FROM moves m
),
first_move AS (
    SELECT DISTINCT ON (l.candidate_id) l.candidate_id, l.from_status
    FROM labelled l
    ORDER BY l.candidate_id, l.created_at, l.id
)
SELECT c.id AS candidate_id, COALESCE(f.from_status, c.status)::text AS status, c.created_at AS entered_at
FROM recruitment.candidates c
LEFT JOIN first_move f ON f.candidate_id = c.id
UNION ALL
SELECT l.candidate_id, l.to_status::text, l.created_at
FROM labelled l
WHERE l.to_status IS NOT NULL;

-- +goose Down
-- 00003's Down drops the view.
