/**
 * SEO suite daily job — wired into GET /api/cron/daily by the integrator.
 * Per SeoProperty: rank tracked keywords, backlink snapshot, domain snapshots (own + competitors),
 * scheduled audits by `auditCadence`. One SyncJob (kind SEO_SUITE_DAILY) per property;
 * RANK_DROP / BACKLINK_LOST notifications to the client's audience.
 */
import "server-only";
import { db } from "@/lib/db";
import { addDays, todayUtc } from "@/lib/dates";
import { clientAudience, notify } from "@/lib/notify";
import { runAuditCore } from "@/features/seo/audit-core";
import { s } from "./strings";
import { refreshBacklinks, refreshDomains, trackRanks } from "./tracker";

export type SeoSuiteDailyResult = { properties: number; ranks: number; backlinks: number; domains: number; audits: number };

const BACKLINK_LOST_THRESHOLD = 5;

function shouldAudit(cadence: "NONE" | "WEEKLY" | "DAILY", lastRunAt: Date | null, now: Date): boolean {
  if (cadence === "NONE") return false;
  if (cadence === "DAILY") return !lastRunAt || now.getTime() - lastRunAt.getTime() > 20 * 60 * 60 * 1000;
  return !lastRunAt || now.getTime() - lastRunAt.getTime() > 6 * 86_400_000;
}

export async function runSeoSuiteDaily(opts: { now?: Date } = {}): Promise<SeoSuiteDailyResult> {
  const now = opts.now ?? new Date();
  const date = todayUtc(now);
  const out: SeoSuiteDailyResult = { properties: 0, ranks: 0, backlinks: 0, domains: 0, audits: 0 };

  const properties = await db.seoProperty.findMany({
    select: { id: true, siteUrl: true, locationCode: true, languageCode: true, auditCadence: true, clientId: true, client: { select: { slug: true } } },
  });

  for (const p of properties) {
    out.properties++;
    const job = await db.syncJob.create({ data: { clientId: p.clientId, kind: "SEO_SUITE_DAILY", status: "RUNNING", message: p.siteUrl } });
    const notes: string[] = [];
    const errors: string[] = [];
    const href = (page: string) => `/clients/${p.client.slug}/seo/${page}`;

    // 1) ranks
    try {
      const r = await trackRanks(p, { date });
      out.ranks += r.updated;
      notes.push(`${r.updated} peringkat`);
      if (r.drops.length > 0) {
        const sample = r.drops
          .slice(0, 3)
          .map((d) => `${d.keyword} (${d.prev ?? "–"}→${d.cur ?? "–"})`)
          .join(", ");
        await notify(await clientAudience(p.clientId), {
          type: "RANK_DROP",
          title: s.notifRankDropTitle,
          body: s.notifRankDropBody(r.drops.length, sample),
          href: href("rank"),
          clientId: p.clientId,
        });
      }
    } catch (e) {
      errors.push(`ranks: ${e instanceof Error ? e.message : String(e)}`);
    }

    // 2) backlinks
    try {
      const b = await refreshBacklinks(p, date);
      out.backlinks++;
      notes.push(`backlink +${b.added}/−${b.lost}`);
      if (b.lost > BACKLINK_LOST_THRESHOLD) {
        await notify(await clientAudience(p.clientId), {
          type: "BACKLINK_LOST",
          title: s.notifBacklinkLostTitle,
          body: s.notifBacklinkLostBody(b.lost),
          href: href("backlinks"),
          clientId: p.clientId,
        });
      }
    } catch (e) {
      errors.push(`backlinks: ${e instanceof Error ? e.message : String(e)}`);
    }

    // 3) domains
    try {
      const n = await refreshDomains(p, date);
      out.domains += n;
      notes.push(`${n} domain`);
    } catch (e) {
      errors.push(`domains: ${e instanceof Error ? e.message : String(e)}`);
    }

    // 4) scheduled audit
    try {
      const last = await db.seoAudit.findFirst({ where: { propertyId: p.id }, orderBy: { runAt: "desc" }, select: { runAt: true } });
      if (shouldAudit(p.auditCadence, last?.runAt ?? null, now)) {
        const res = await runAuditCore(p.id);
        if (!res.allFailed) out.audits++;
        notes.push(`audit ${res.allFailed ? "gagal" : "ok"}`);
        if (res.errorParts.length) errors.push(`audit: ${res.errorParts[0]}`);
      }
    } catch (e) {
      errors.push(`audit: ${e instanceof Error ? e.message : String(e)}`);
    }

    await db.syncJob.update({
      where: { id: job.id },
      data: {
        status: errors.length && notes.length === 0 ? "FAILED" : "SUCCESS",
        finishedAt: new Date(),
        message: notes.join(" · ") || null,
        error: errors.length ? errors.join(" | ").slice(0, 900) : null,
      },
    });
  }

  // housekeeping: research cache older than 30 days is never reused
  await db.keywordResearch.deleteMany({ where: { fetchedAt: { lt: addDays(date, -30) } } });
  return out;
}
