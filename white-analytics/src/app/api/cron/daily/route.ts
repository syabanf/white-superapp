import { db } from "@/lib/db";
import { runDailySync } from "@/features/sync/daily";
import { runSeoSuiteDaily } from "@/features/seo-suite/daily";
import { runRetention } from "@/features/sync/retention";
import { planDailyRun } from "@/features/sync/policy";

export const dynamic = "force-dynamic";
export const maxDuration = 300;

/**
 * Daily cron (02:00 UTC via vercel.json), guarded by `Authorization: Bearer ${CRON_SECRET}`.
 *
 * Without `?client=` this is the dispatcher: it runs retention, then calls itself
 * once per client so every client gets its own invocation and its own 300 s
 * budget. Clients are ordered by their last daily run, a fixed number run in
 * parallel, and whatever does not fit the dispatcher's budget is reported as
 * deferred and goes first tomorrow.
 *
 * With `?client=<id>` it does the work for one client: source sync (Social,
 * GSC/GA4, Ads) and the SEO suite job (ranks, backlinks, domains, scheduled audits).
 * Due posts are published by the separate 5-minute cron `/api/cron/publish`.
 */
const CONCURRENCY = 4;
const BUDGET_MS = 270_000;
const PER_CLIENT_MS = 90_000;

async function attempt<T>(fn: () => Promise<T>): Promise<T | { error: string }> {
  try {
    return await fn();
  } catch (e) {
    return { error: e instanceof Error ? e.message : String(e) };
  }
}

export async function GET(req: Request) {
  const secret = process.env.CRON_SECRET;
  if (!secret || req.headers.get("authorization") !== `Bearer ${secret}`) {
    return Response.json({ error: "UNAUTHORIZED" }, { status: 401 });
  }
  const url = new URL(req.url);
  const startedAt = new Date();

  const clientId = url.searchParams.get("client");
  if (clientId) {
    const sync = await attempt(() => runDailySync({ clientId }));
    const seoSuite = await attempt(() => runSeoSuiteDaily({ now: startedAt, clientId }));
    return Response.json({ ok: true, client: clientId, sync, seoSuite });
  }

  const retention = await attempt(() => runRetention(startedAt));
  const [clients, lastRuns] = await Promise.all([
    db.client.findMany({ select: { id: true, slug: true } }),
    db.syncJob.groupBy({ by: ["clientId"], where: { kind: "SEO_GSC" }, _max: { startedAt: true } }),
  ]);
  const lastBy = new Map(lastRuns.map((r) => [r.clientId, r._max.startedAt]));
  const plan = planDailyRun(
    clients.map((c) => ({ ...c, lastRunAt: lastBy.get(c.id) ?? null })),
    { concurrency: CONCURRENCY, budgetMs: BUDGET_MS, perClientMs: PER_CLIENT_MS },
  );

  const results: { client: string; ok: boolean; status: number; ms: number }[] = [];
  const queue = [...plan.run];
  const worker = async () => {
    for (let c = queue.shift(); c; c = queue.shift()) {
      const t0 = Date.now();
      const target = `${url.origin}/api/cron/daily?client=${encodeURIComponent(c.id)}`;
      const res = await attempt(() => fetch(target, { headers: { authorization: `Bearer ${secret}` }, cache: "no-store" }));
      const status = "error" in res ? 0 : res.status;
      results.push({ client: c.slug, ok: status === 200, status, ms: Date.now() - t0 });
    }
  };
  await Promise.all(Array.from({ length: CONCURRENCY }, worker));

  return Response.json({
    ok: results.every((r) => r.ok),
    ranAt: startedAt.toISOString(),
    retention,
    clients: clients.length,
    results,
    deferred: plan.deferred.map((c) => c.slug),
  });
}
