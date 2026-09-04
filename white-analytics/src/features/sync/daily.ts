/**
 * Daily sync orchestrator (called by GET /api/cron/daily). Per client and per
 * source it picks the most real path available and records one SyncJob:
 *  - Social: Meta Insights (own accounts with a connection) → Apify public data → mock log
 *  - SEO:    GSC/GA4 via Google connection → mock log
 *  - Ads:    Marketing API via Meta connection → mock log
 * Failures are captured per client so one broken token never stops the run.
 */
import "server-only";
import { db } from "@/lib/db";
import { isApifyConfigured } from "@/lib/providers/apify";
import { PLATFORMS } from "@/features/social/lib";
import { syncSocialMeta, syncSocialPublic } from "@/features/social/sync";
import { syncSeoForClient } from "@/features/seo/sync";
import { syncAdsForClient } from "@/features/ads/sync";

export type DailySyncSummary = { client: string; kind: string; status: "SUCCESS" | "FAILED"; mode: "real" | "demo"; message: string }[];

async function mockJob(clientId: string, kind: string) {
  const now = new Date();
  await db.syncJob.create({ data: { clientId, kind, status: "SUCCESS", startedAt: now, finishedAt: now, message: "mock" } });
}

export async function runDailySync(): Promise<{ clients: number; summary: DailySyncSummary }> {
  const clients = await db.client.findMany({ select: { id: true, slug: true } });
  const summary: DailySyncSummary = [];
  const apify = isApifyConfigured();

  for (const client of clients) {
    // ── Social ──
    try {
      let updated = 0;
      let posts = 0;
      const failed: string[] = [];
      let real = false;
      for (const platform of PLATFORMS) {
        if (platform !== "TIKTOK") {
          const m = await syncSocialMeta({ clientId: client.id, platform });
          updated += m.updated;
          posts += m.posts;
          failed.push(...m.failed.map((f) => `${platform.toLowerCase()} @${f.username}: ${f.error}`));
          if (m.updated > 0) real = true;
          // Apify still refreshes competitors and any account without a Meta connection.
        }
        if (apify) {
          const r = await syncSocialPublic({ clientId: client.id, platform, includeCompetitors: true });
          updated += r.updated;
          posts += r.posts;
          failed.push(...r.failed.map((f) => `${platform.toLowerCase()} @${f.username}: ${f.error}`));
          if (r.updated > 0) real = true;
        }
      }
      if (real || failed.length) {
        const status = updated === 0 && failed.length ? "FAILED" : "SUCCESS";
        const message = `${updated} akun, ${posts} post`;
        await db.syncJob.create({ data: { clientId: client.id, kind: "SOCIAL_SNAPSHOT", status, finishedAt: new Date(), message, error: failed.length ? failed.join("; ").slice(0, 1000) : null } });
        summary.push({ client: client.slug, kind: "SOCIAL_SNAPSHOT", status, mode: "real", message });
      } else {
        await mockJob(client.id, "SOCIAL_SNAPSHOT");
        summary.push({ client: client.slug, kind: "SOCIAL_SNAPSHOT", status: "SUCCESS", mode: "demo", message: "mock" });
      }
    } catch (e) {
      summary.push({ client: client.slug, kind: "SOCIAL_SNAPSHOT", status: "FAILED", mode: "real", message: e instanceof Error ? e.message : String(e) });
    }

    // ── SEO (GSC + GA4) ──
    try {
      const r = await syncSeoForClient(client.id, { days: 30 });
      const failed = r.results.find((x) => x.error);
      summary.push({ client: client.slug, kind: "SEO_GSC", status: failed ? "FAILED" : "SUCCESS", mode: r.real > 0 ? "real" : "demo", message: failed?.error ?? (r.real > 0 ? `${r.real}/${r.properties} properti` : "mock") });
    } catch (e) {
      summary.push({ client: client.slug, kind: "SEO_GSC", status: "FAILED", mode: "real", message: e instanceof Error ? e.message : String(e) });
    }

    // ── Ads ──
    try {
      const r = await syncAdsForClient(client.id, { days: 30 });
      const failed = r.results.find((x) => x.error);
      summary.push({ client: client.slug, kind: "ADS_INSIGHTS", status: failed ? "FAILED" : "SUCCESS", mode: r.real > 0 ? "real" : "demo", message: failed?.error ?? (r.real > 0 ? `${r.real}/${r.accounts} akun` : "mock") });
    } catch (e) {
      summary.push({ client: client.slug, kind: "ADS_INSIGHTS", status: "FAILED", mode: "real", message: e instanceof Error ? e.message : String(e) });
    }
  }
  return { clients: clients.length, summary };
}
