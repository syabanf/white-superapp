import { runDailySync } from "@/features/sync/daily";
import { runSeoSuiteDaily } from "@/features/seo-suite/daily";

export const dynamic = "force-dynamic";

/**
 * Daily cron (02:00 UTC via vercel.json). Guarded by `Authorization: Bearer ${CRON_SECRET}`.
 * 1. Source sync per client: Social (Meta Insights → Apify → mock), SEO (GSC/GA4), Ads (Marketing API).
 *    Each picks the most real path its connections allow — see features/sync/daily.ts.
 * 2. SEO suite daily job: rank snapshots, backlinks, domain snapshots, scheduled audits, alerts.
 * Due posts are published by the separate 5-minute cron `/api/cron/publish`.
 */
export async function GET(req: Request) {
  const secret = process.env.CRON_SECRET;
  const auth = req.headers.get("authorization");
  if (!secret || auth !== `Bearer ${secret}`) {
    return Response.json({ error: "UNAUTHORIZED" }, { status: 401 });
  }
  const startedAt = new Date();

  let sync: Awaited<ReturnType<typeof runDailySync>> | { error: string };
  try {
    sync = await runDailySync();
  } catch (e) {
    sync = { error: e instanceof Error ? e.message : String(e) };
  }

  let seoSuite: Awaited<ReturnType<typeof runSeoSuiteDaily>> | { error: string };
  try {
    seoSuite = await runSeoSuiteDaily({ now: startedAt });
  } catch (e) {
    seoSuite = { error: e instanceof Error ? e.message : String(e) };
  }

  return Response.json({ ok: true, ranAt: startedAt.toISOString(), sync, seoSuite });
}
