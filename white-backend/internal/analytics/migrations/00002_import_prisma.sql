-- +goose Up
-- One-time copy from the Prisma-era tables in `public`, when they exist.
-- Rows keep their ids so references from other services stay valid.
-- +goose StatementBegin
DO $$
BEGIN
    IF to_regclass('public."SocialAccount"') IS NULL THEN
        RETURN;
    END IF;

    INSERT INTO analytics.social_accounts (id, client_id, platform, external_id, username, display_name, avatar_url, biography,
                                           is_competitor, connection_id, created_at, updated_at)
    SELECT id, "clientId", platform::text::analytics.social_platform, "externalId", username, "displayName", "avatarUrl", biography,
           "isCompetitor", "connectionId", "createdAt", "updatedAt"
    FROM public."SocialAccount"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.social_snapshots (id, social_account_id, date, followers, following, media_count, reach, impressions, profile_views, website_clicks)
    SELECT s.id, s."socialAccountId", s.date, s.followers, s.following, s."mediaCount", s.reach, s.impressions, s."profileViews", s."websiteClicks"
    FROM public."SocialSnapshot" s
    JOIN analytics.social_accounts a ON a.id = s."socialAccountId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.social_posts (id, social_account_id, external_id, caption, media_type, product_type, permalink, media_url, thumbnail_url,
                                        published_at, likes, comments, shares, saves, views, reach, impressions, synced_at)
    SELECT p.id, p."socialAccountId", p."externalId", p.caption, p."mediaType", p."productType", p.permalink, p."mediaUrl", p."thumbnailUrl",
           p."publishedAt", p.likes, p.comments, p.shares, p.saves, p.views, p.reach, p.impressions, p."syncedAt"
    FROM public."SocialPost" p
    JOIN analytics.social_accounts a ON a.id = p."socialAccountId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_properties (id, client_id, site_url, ga4_property_id, connection_id, location_code, language_code, audit_cadence, created_at, updated_at)
    SELECT id, "clientId", "siteUrl", "ga4PropertyId", "connectionId", "locationCode", "languageCode", "auditCadence"::text::analytics.audit_cadence, "createdAt", "updatedAt"
    FROM public."SeoProperty"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_daily_metrics (id, property_id, date, clicks, impressions, ctr, position)
    SELECT m.id, m."propertyId", m.date, m.clicks, m.impressions, m.ctr, m.position
    FROM public."SeoDailyMetric" m
    JOIN analytics.seo_properties p ON p.id = m."propertyId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_dimension_metrics (id, property_id, date, dimension, key, clicks, impressions, ctr, position)
    SELECT m.id, m."propertyId", m.date, m.dimension::text::analytics.seo_dimension, m.key, m.clicks, m.impressions, m.ctr, m.position
    FROM public."SeoDimensionMetric" m
    JOIN analytics.seo_properties p ON p.id = m."propertyId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ga4_daily_metrics (id, property_id, date, sessions, organic_sessions, users, engaged_sessions, conversions, avg_engagement_sec)
    SELECT m.id, m."propertyId", m.date, m.sessions, m."organicSessions", m.users, m."engagedSessions", m.conversions, m."avgEngagementSec"
    FROM public."Ga4DailyMetric" m
    JOIN analytics.seo_properties p ON p.id = m."propertyId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_audits (id, property_id, url, strategy, run_at, performance, seo, accessibility, best_practices,
                                      lcp_ms, inp_ms, cls, fcp_ms, ttfb_ms, tbt_ms, speed_index_ms, raw)
    SELECT a.id, a."propertyId", a.url, a.strategy::text::analytics.audit_strategy, a."runAt", a.performance, a.seo, a.accessibility, a."bestPractices",
           a."lcpMs", a."inpMs", a.cls, a."fcpMs", a."ttfbMs", a."tbtMs", a."speedIndexMs", a.raw
    FROM public."SeoAudit" a
    JOIN analytics.seo_properties p ON p.id = a."propertyId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_crawls (id, property_id, started_at, finished_at, status, pages_crawled, max_pages, error)
    SELECT c.id, c."propertyId", c."startedAt", c."finishedAt", c.status::text::analytics.job_status, c."pagesCrawled", c."maxPages", c.error
    FROM public."SeoCrawl" c
    JOIN analytics.seo_properties p ON p.id = c."propertyId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_crawl_pages (id, crawl_id, url, status_code, title, meta_description, h1_count, word_count, canonical, indexable,
                                           images_total, images_missing_alt, internal_links, external_links, load_ms, has_json_ld, has_hreflang)
    SELECT g.id, g."crawlId", g.url, g."statusCode", g.title, g."metaDescription", g."h1Count", g."wordCount", g.canonical, g.indexable,
           g."imagesTotal", g."imagesMissingAlt", g."internalLinks", g."externalLinks", g."loadMs", g."hasJsonLd", g."hasHreflang"
    FROM public."SeoCrawlPage" g
    JOIN analytics.seo_crawls c ON c.id = g."crawlId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.seo_issues (id, crawl_id, severity, code, message, url)
    SELECT i.id, i."crawlId", i.severity::text::analytics.issue_severity, i.code, i.message, i.url
    FROM public."SeoIssue" i
    JOIN analytics.seo_crawls c ON c.id = i."crawlId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ad_accounts (id, client_id, external_id, name, currency, connection_id, last_synced_at, created_at, updated_at)
    SELECT id, "clientId", "externalId", name, currency, "connectionId", "lastSyncedAt", "createdAt", "updatedAt"
    FROM public."AdAccount"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ad_campaigns (id, ad_account_id, external_id, name, objective, status, daily_budget, start_time, created_at, updated_at)
    SELECT c.id, c."adAccountId", c."externalId", c.name, c.objective, c.status, c."dailyBudget", c."startTime", c."createdAt", c."updatedAt"
    FROM public."AdCampaign" c
    JOIN analytics.ad_accounts a ON a.id = c."adAccountId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ad_sets (id, campaign_id, external_id, name, status, daily_budget, created_at, updated_at)
    SELECT s.id, s."campaignId", s."externalId", s.name, s.status, s."dailyBudget", s."createdAt", s."updatedAt"
    FROM public."AdSet" s
    JOIN analytics.ad_campaigns c ON c.id = s."campaignId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ads (id, ad_set_id, external_id, name, status, thumbnail_url, created_at, updated_at)
    SELECT d.id, d."adSetId", d."externalId", d.name, d.status, d."thumbnailUrl", d."createdAt", d."updatedAt"
    FROM public."Ad" d
    JOIN analytics.ad_sets s ON s.id = d."adSetId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ad_daily_insights (id, ad_account_id, date, campaign_id, ad_set_id, ad_id, spend, impressions, reach, clicks, link_clicks,
                                             results, result_type, purchase_value)
    SELECT i.id, i."adAccountId", i.date, i."campaignId", i."adSetId", i."adId", i.spend, i.impressions, i.reach, i.clicks, i."linkClicks",
           i.results, i."resultType", i."purchaseValue"
    FROM public."AdDailyInsight" i
    JOIN analytics.ad_accounts a ON a.id = i."adAccountId"
    WHERE (i."campaignId" IS NULL OR EXISTS (SELECT 1 FROM analytics.ad_campaigns c WHERE c.id = i."campaignId"))
      AND (i."adSetId" IS NULL OR EXISTS (SELECT 1 FROM analytics.ad_sets s WHERE s.id = i."adSetId"))
      AND (i."adId" IS NULL OR EXISTS (SELECT 1 FROM analytics.ads d WHERE d.id = i."adId"))
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.ad_demographics (id, ad_account_id, date_from, date_to, campaign_id, age, gender, spend, impressions, reach, results)
    SELECT d.id, d."adAccountId", d."dateFrom", d."dateTo", d."campaignId", d.age, d.gender, d.spend, d.impressions, d.reach, d.results
    FROM public."AdDemographic" d
    JOIN analytics.ad_accounts a ON a.id = d."adAccountId"
    ON CONFLICT DO NOTHING;

    INSERT INTO analytics.sync_jobs (id, client_id, kind, status, started_at, finished_at, message, error)
    SELECT id, "clientId", kind, status::text::analytics.job_status, "startedAt", "finishedAt", message, error
    FROM public."SyncJob"
    WHERE "clientId" IS NOT NULL
      AND kind IN ('SOCIAL_SNAPSHOT', 'SEO_GSC', 'SEO_GA4', 'SEO_AUDIT', 'ADS_INSIGHTS', 'ADS_CSV_IMPORT')
    ON CONFLICT DO NOTHING;
END $$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
