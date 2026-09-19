/**
 * Data retention, run once per day by the cron dispatcher. Windows are long
 * enough that every report range in the UI (12 months plus its comparison
 * period) stays answerable. Logs and read notifications go sooner.
 */
import "server-only";
import { db } from "@/lib/db";
import { retentionCutoffs } from "./policy";

export type RetentionResult = { syncJobs: number; notifications: number; rankSnapshots: number; backlinkSnapshots: number; domainSnapshots: number };

export async function runRetention(now = new Date()): Promise<RetentionResult> {
  const c = retentionCutoffs(now);
  const [syncJobs, readNotifs, oldNotifs, rankSnapshots, backlinkSnapshots, domainSnapshots] = await Promise.all([
    db.syncJob.deleteMany({ where: { startedAt: { lt: c.syncJobs } } }),
    db.notification.deleteMany({ where: { readAt: { lt: c.readNotifications } } }),
    db.notification.deleteMany({ where: { createdAt: { lt: c.anyNotifications } } }),
    db.rankSnapshot.deleteMany({ where: { date: { lt: c.seoSnapshots } } }),
    db.backlinkSnapshot.deleteMany({ where: { date: { lt: c.seoSnapshots } } }),
    db.domainSnapshot.deleteMany({ where: { date: { lt: c.seoSnapshots } } }),
  ]);
  return {
    syncJobs: syncJobs.count,
    notifications: readNotifs.count + oldNotifs.count,
    rankSnapshots: rankSnapshots.count,
    backlinkSnapshots: backlinkSnapshots.count,
    domainSnapshots: domainSnapshots.count,
  };
}
