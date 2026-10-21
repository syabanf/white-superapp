-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Ids are kept. client_id is denormalised from public."SeoProperty" so the
-- rows stay purgeable without the analytics service. Prisma stored naive UTC
-- timestamps, hence AT TIME ZONE 'UTC'.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."TrackedKeyword"') IS NULL OR to_regclass('public."SeoProperty"') IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO seosuite.tracked_keywords (id, client_id, property_id, keyword, location_code, language_code, device, tags,
                                           target_url, volume, difficulty, cpc, intent, created_at, updated_at)
    SELECT k.id, p."clientId", k."propertyId", k.keyword, k."locationCode", k."languageCode",
           k.device::text::seosuite.device_type, COALESCE(k.tags, '{}'::text[]),
           k."targetUrl", k.volume, k.difficulty, k.cpc, k.intent::text::seosuite.keyword_intent,
           k."createdAt" AT TIME ZONE 'UTC', k."updatedAt" AT TIME ZONE 'UTC'
    FROM public."TrackedKeyword" k
    JOIN public."SeoProperty" p ON p.id = k."propertyId"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.rank_snapshots (id, client_id, keyword_id, date, position, url, serp_features, top_results)
    SELECT r.id, p."clientId", r."keywordId", r.date, r.position, r.url, COALESCE(r."serpFeatures", '{}'::text[]), r."topResults"
    FROM public."RankSnapshot" r
    JOIN public."TrackedKeyword" k ON k.id = r."keywordId"
    JOIN public."SeoProperty" p ON p.id = k."propertyId"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.keyword_research (id, client_id, seed, location_code, language_code, provider, results, fetched_at)
    SELECT id, "clientId", seed, "locationCode", "languageCode", provider, results, "fetchedAt" AT TIME ZONE 'UTC'
    FROM public."KeywordResearch"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.backlinks (id, client_id, property_id, source_url, source_domain, target_url, anchor, dofollow,
                                    domain_rank, spam_score, first_seen, last_seen, is_lost)
    SELECT b.id, p."clientId", b."propertyId", b."sourceUrl", b."sourceDomain", b."targetUrl", b.anchor, b.dofollow,
           b."domainRank", b."spamScore", b."firstSeen", b."lastSeen", b."isLost"
    FROM public."Backlink" b
    JOIN public."SeoProperty" p ON p.id = b."propertyId"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.backlink_snapshots (id, client_id, property_id, date, backlinks, referring_domains, dofollow, nofollow,
                                             new_backlinks, lost_backlinks, domain_rank, toxic_share)
    SELECT s.id, p."clientId", s."propertyId", s.date, s.backlinks, s."referringDomains", s.dofollow, s.nofollow,
           s."newBacklinks", s."lostBacklinks", s."domainRank", s."toxicShare"
    FROM public."BacklinkSnapshot" s
    JOIN public."SeoProperty" p ON p.id = s."propertyId"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.competitor_domains (id, client_id, property_id, domain, created_at)
    SELECT c.id, p."clientId", c."propertyId", c.domain, c."createdAt" AT TIME ZONE 'UTC'
    FROM public."CompetitorDomain" c
    JOIN public."SeoProperty" p ON p.id = c."propertyId"
    ON CONFLICT (id) DO NOTHING;

    INSERT INTO seosuite.domain_snapshots (id, client_id, property_id, domain, date, organic_keywords, organic_traffic, organic_cost,
                                           top3, top10, top100, backlinks, referring_domains, domain_rank)
    SELECT d.id, p."clientId", d."propertyId", d.domain, d.date, d."organicKeywords", d."organicTraffic", d."organicCost",
           d.top3, d.top10, d.top100, d.backlinks, d."referringDomains", d."domainRank"
    FROM public."DomainSnapshot" d
    JOIN public."SeoProperty" p ON p.id = d."propertyId"
    ON CONFLICT (id) DO NOTHING;

    IF to_regclass('public."SyncJob"') IS NOT NULL THEN
        INSERT INTO seosuite.sync_jobs (id, client_id, kind, status, started_at, finished_at, message, error)
        SELECT id, "clientId", kind, status::text::seosuite.job_status,
               "startedAt" AT TIME ZONE 'UTC', "finishedAt" AT TIME ZONE 'UTC', message, error
        FROM public."SyncJob"
        WHERE kind LIKE 'SEO_SUITE_%' AND "clientId" IS NOT NULL
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
