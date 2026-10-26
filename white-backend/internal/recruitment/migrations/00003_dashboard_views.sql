-- +goose Up
-- Read models for the recruitment dashboard.

-- Every stage a candidate entered, derived from the audited status_change
-- activities ("Tahap diubah: <from> → <to>", labels of domain.StatusLabel).
-- The first row is the stage the candidate was created in: the "from" of
-- the first move, or the current status when it never moved.
CREATE VIEW recruitment.candidate_stage_events AS
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
)
SELECT c.id AS candidate_id,
       COALESCE((SELECT l.from_status FROM labelled l WHERE l.candidate_id = c.id ORDER BY l.created_at, l.id LIMIT 1), c.status)::text AS status,
       c.created_at AS entered_at
FROM recruitment.candidates c
UNION ALL
SELECT l.candidate_id, l.to_status::text, l.created_at
FROM labelled l
WHERE l.to_status IS NOT NULL;

-- Each stage stay with the moment the candidate left it (NULL while there).
CREATE VIEW recruitment.candidate_stage_spans AS
SELECT e.candidate_id, e.status, e.entered_at,
       lead(e.entered_at) OVER (PARTITION BY e.candidate_id ORDER BY e.entered_at) AS left_at
FROM recruitment.candidate_stage_events e;

-- When an opening was taking applications: from publication until it was
-- closed (the earlier of the close and its closing date) or its closing
-- date passed. Drafts and deleted openings are left out.
CREATE VIEW recruitment.opening_windows AS
SELECT j.id, j.title, j.department, j.status, j.headcount, j.closing_date, j.salary_min, j.salary_max,
       COALESCE(j.published_at, j.created_at) AS opened_at,
       CASE
           WHEN j.status = 'closed' THEN LEAST(j.updated_at, COALESCE((j.closing_date + 1)::timestamp AT TIME ZONE 'Asia/Jakarta', j.updated_at))
           WHEN j.closing_date IS NOT NULL THEN (j.closing_date + 1)::timestamp AT TIME ZONE 'Asia/Jakarta'
       END AS closed_at
FROM recruitment.job_openings j
WHERE j.deleted_at IS NULL AND j.status <> 'draft';

-- +goose Down
DROP VIEW IF EXISTS recruitment.opening_windows;
DROP VIEW IF EXISTS recruitment.candidate_stage_spans;
DROP VIEW IF EXISTS recruitment.candidate_stage_events;
