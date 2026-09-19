/** Retention windows in days. Pure, tested in tests/cron.test.ts. */
export const RETENTION_DAYS = { syncJobs: 90, readNotifications: 30, anyNotifications: 90, seoSnapshots: 760 } as const;

export type RetentionCutoffs = Record<keyof typeof RETENTION_DAYS, Date>;

export function retentionCutoffs(now: Date): RetentionCutoffs {
  const at = (days: number) => new Date(now.getTime() - days * 86_400_000);
  return {
    syncJobs: at(RETENTION_DAYS.syncJobs),
    readNotifications: at(RETENTION_DAYS.readNotifications),
    anyNotifications: at(RETENTION_DAYS.anyNotifications),
    seoSnapshots: at(RETENTION_DAYS.seoSnapshots),
  };
}

/**
 * Fan-out plan for the daily cron: clients that waited longest go first, and
 * only as many as fit the time budget are started. The rest are deferred to the
 * next run, where their older `lastRunAt` puts them at the front.
 */
export function planDailyRun<T extends { id: string; lastRunAt: Date | null }>(
  clients: T[],
  opts: { concurrency: number; budgetMs: number; perClientMs: number },
): { run: T[]; deferred: T[] } {
  const sorted = [...clients].sort((a, b) => (a.lastRunAt?.getTime() ?? 0) - (b.lastRunAt?.getTime() ?? 0));
  const waves = Math.max(1, Math.floor(opts.budgetMs / opts.perClientMs));
  const capacity = waves * Math.max(1, opts.concurrency);
  return { run: sorted.slice(0, capacity), deferred: sorted.slice(capacity) };
}
