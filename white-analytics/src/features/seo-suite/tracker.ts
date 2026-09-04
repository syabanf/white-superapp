/**
 * SEO suite write paths shared by the server actions ("Perbarui") and the daily cron job.
 * Auth-free: callers are responsible for access checks. Provider errors propagate.
 */
import "server-only";
import { db } from "@/lib/db";
import { addDays } from "@/lib/dates";
import { isRankDropAlert, siteHost } from "@/lib/metrics/seo-suite";
import { seoData } from "@/lib/providers/dataforseo";
import type { Prisma } from "@/generated/prisma/client";

export type TrackerProperty = { id: string; siteUrl: string; locationCode: number; languageCode: string };

export type RankDrop = { keyword: string; prev: number | null; cur: number | null };

const SERP_BATCH = 20;

/** Fetch today's SERP positions for tracked keywords (optionally a subset) and upsert `RankSnapshot`. */
export async function trackRanks(property: TrackerProperty, opts: { date: Date; keywordIds?: string[] }): Promise<{ updated: number; drops: RankDrop[] }> {
  const keywords = await db.trackedKeyword.findMany({
    where: { propertyId: property.id, ...(opts.keywordIds ? { id: { in: opts.keywordIds } } : {}) },
    select: { id: true, keyword: true, device: true },
  });
  if (keywords.length === 0) return { updated: 0, drops: [] };
  const targetDomain = siteHost(property.siteUrl);
  const yesterday = addDays(opts.date, -1);
  const prevRows = await db.rankSnapshot.findMany({ where: { keywordId: { in: keywords.map((k) => k.id) }, date: yesterday }, select: { keywordId: true, position: true } });
  const prevBy = new Map(prevRows.map((r) => [r.keywordId, r.position]));

  let updated = 0;
  const drops: RankDrop[] = [];
  for (let i = 0; i < keywords.length; i += SERP_BATCH) {
    const batch = keywords.slice(i, i + SERP_BATCH);
    const results = await seoData.serpRanks(
      batch.map((k) => ({ keyword: k.keyword, device: k.device })),
      { locationCode: property.locationCode, languageCode: property.languageCode, targetDomain },
    );
    await Promise.all(
      batch.map((k, j) => {
        const r = results[j];
        if (!r) return Promise.resolve();
        const data = { position: r.position, url: r.url, serpFeatures: r.serpFeatures, topResults: r.top10 as unknown as Prisma.InputJsonValue };
        updated++;
        if (prevBy.has(k.id) && isRankDropAlert(prevBy.get(k.id) ?? null, r.position)) drops.push({ keyword: k.keyword, prev: prevBy.get(k.id) ?? null, cur: r.position });
        return db.rankSnapshot.upsert({
          where: { keywordId_date: { keywordId: k.id, date: opts.date } },
          create: { keywordId: k.id, date: opts.date, ...data },
          update: data,
        });
      }),
    );
  }
  return { updated, drops };
}

/** Provider summary → today's `BacklinkSnapshot`; provider link list → `Backlink` upserts (lost flags included). */
export async function refreshBacklinks(property: TrackerProperty, date: Date): Promise<{ lost: number; added: number }> {
  const domain = siteHost(property.siteUrl);
  const [summary, list, existing] = await Promise.all([
    seoData.backlinkSummary(domain),
    seoData.backlinks(domain, { limit: 300, mode: "all" }),
    db.backlink.findMany({ where: { propertyId: property.id }, select: { sourceUrl: true, targetUrl: true, isLost: true } }),
  ]);
  const existingKey = new Map(existing.map((e) => [`${e.sourceUrl}|${e.targetUrl}`, e.isLost]));
  let added = 0;
  let lost = 0;
  for (let i = 0; i < list.length; i += 25) {
    await Promise.all(
      list.slice(i, i + 25).map((b) => {
        const key = `${b.sourceUrl}|${b.targetUrl}`;
        const was = existingKey.get(key);
        if (was === undefined) added++;
        else if (!was && b.isLost) lost++;
        const firstSeen = new Date(b.firstSeen || date.toISOString());
        const lastSeen = new Date(b.lastSeen || date.toISOString());
        return db.backlink.upsert({
          where: { propertyId_sourceUrl_targetUrl: { propertyId: property.id, sourceUrl: b.sourceUrl, targetUrl: b.targetUrl } },
          create: { propertyId: property.id, sourceUrl: b.sourceUrl, sourceDomain: b.sourceDomain, targetUrl: b.targetUrl, anchor: b.anchor, dofollow: b.dofollow, domainRank: b.domainRank, spamScore: b.spamScore, firstSeen, lastSeen, isLost: b.isLost },
          update: { anchor: b.anchor, dofollow: b.dofollow, domainRank: b.domainRank, spamScore: b.spamScore, lastSeen, isLost: b.isLost },
        });
      }),
    );
  }
  // first run has no baseline: spread the provider's 30-day figures over a day
  const firstRun = existing.length === 0;
  const newBacklinks = firstRun ? Math.round(summary.newLast30 / 30) : added;
  const lostBacklinks = firstRun ? Math.round(summary.lostLast30 / 30) : lost;
  const data = {
    backlinks: summary.backlinks,
    referringDomains: summary.referringDomains,
    dofollow: summary.dofollow,
    nofollow: summary.nofollow,
    newBacklinks,
    lostBacklinks,
    domainRank: summary.domainRank,
    toxicShare: summary.toxicShare,
  };
  await db.backlinkSnapshot.upsert({ where: { propertyId_date: { propertyId: property.id, date } }, create: { propertyId: property.id, date, ...data }, update: data });
  return { lost: lostBacklinks, added: newBacklinks };
}

/** `DomainSnapshot` for the own domain + every competitor (or an explicit list). */
export async function refreshDomains(property: TrackerProperty, date: Date, only?: string[]): Promise<number> {
  const competitors = await db.competitorDomain.findMany({ where: { propertyId: property.id }, select: { domain: true } });
  const domains = only ?? [siteHost(property.siteUrl), ...competitors.map((c) => c.domain)];
  let n = 0;
  for (const domain of domains) {
    const o = await seoData.domainOverview(domain, property.locationCode, property.languageCode);
    const data = {
      organicKeywords: o.organicKeywords,
      organicTraffic: o.organicTraffic,
      organicCost: o.organicCost,
      top3: o.top3,
      top10: o.top10,
      top100: o.top100,
      backlinks: o.backlinks,
      referringDomains: o.referringDomains,
      domainRank: o.domainRank,
    };
    await db.domainSnapshot.upsert({ where: { propertyId_domain_date: { propertyId: property.id, domain, date } }, create: { propertyId: property.id, domain, date, ...data }, update: data });
    n++;
  }
  return n;
}
