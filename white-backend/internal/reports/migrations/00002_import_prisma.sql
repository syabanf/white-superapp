-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Ids are kept. Prisma stored naive UTC timestamps, hence AT TIME ZONE 'UTC'.
-- Report.pdfPath was never written and is not carried over.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."Report"') IS NOT NULL THEN
        INSERT INTO reports.reports (id, client_id, title, date_from, date_to, compare, modules, language,
                                     ai_summary, created_by_id, created_at)
        SELECT id, "clientId", title, "dateFrom", "dateTo", compare, COALESCE(modules::text[], '{}'::text[]),
               language, "aiSummary", "createdById", "createdAt" AT TIME ZONE 'UTC'
        FROM public."Report"
        ON CONFLICT (id) DO NOTHING;
    END IF;

    IF to_regclass('public."AiInsight"') IS NOT NULL THEN
        INSERT INTO reports.ai_insights (id, client_id, module, date_from, date_to, language, model, content, created_at)
        SELECT id, "clientId", module, "dateFrom", "dateTo", language, model, content, "createdAt" AT TIME ZONE 'UTC'
        FROM public."AiInsight"
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
