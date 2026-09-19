import { describe, expect, it } from "vitest";
import { RETENTION_DAYS, planDailyRun, retentionCutoffs } from "@/features/sync/policy";
import { sniffContentType } from "@/lib/storage";

describe("retention policy", () => {
  it("keeps SEO snapshots long enough for a 12-month range plus comparison", () => {
    expect(RETENTION_DAYS.seoSnapshots).toBeGreaterThanOrEqual(730);
    const c = retentionCutoffs(new Date("2026-09-19T00:00:00Z"));
    expect(c.syncJobs.toISOString()).toBe("2026-06-21T00:00:00.000Z");
    expect(c.readNotifications.getTime()).toBeGreaterThan(c.anyNotifications.getTime());
  });
});

describe("daily cron plan", () => {
  const client = (id: string, last: string | null) => ({ id, lastRunAt: last ? new Date(last) : null });
  it("runs never-synced and longest-waiting clients first, defers what does not fit", () => {
    const clients = [client("a", "2026-09-18"), client("b", null), client("c", "2026-09-10"), client("d", "2026-09-17"), client("e", "2026-09-16")];
    const plan = planDailyRun(clients, { concurrency: 2, budgetMs: 200, perClientMs: 100 });
    expect(plan.run.map((c) => c.id)).toEqual(["b", "c", "e", "d"]);
    expect(plan.deferred.map((c) => c.id)).toEqual(["a"]);
  });
  it("always runs at least one wave", () => {
    expect(planDailyRun([client("a", null)], { concurrency: 1, budgetMs: 10, perClientMs: 100 }).run).toHaveLength(1);
  });
});

describe("upload type sniffing", () => {
  const pad = new Array<number>(16).fill(0);
  const bytes = (...n: number[]) => Uint8Array.from([...n, ...pad]);
  const codes = (s: string) => [...s].map((ch) => ch.charCodeAt(0));
  it("recognises allowed formats by their leading bytes", () => {
    expect(sniffContentType(bytes(0xff, 0xd8, 0xff, 0xe0))).toBe("image/jpeg");
    expect(sniffContentType(bytes(0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a))).toBe("image/png");
    expect(sniffContentType(bytes(...codes("GIF89a")))).toBe("image/gif");
    expect(sniffContentType(bytes(...codes("RIFF"), 1, 2, 3, 4, ...codes("WEBP")))).toBe("image/webp");
    expect(sniffContentType(bytes(0, 0, 0, 24, ...codes("ftypisom")))).toBe("video/mp4");
    expect(sniffContentType(bytes(0, 0, 0, 20, ...codes("ftypqt  ")))).toBe("video/quicktime");
  });
  it("rejects scripts and HTML renamed to images", () => {
    expect(sniffContentType(bytes(...codes("<svg onload=alert(1)>")))).toBeNull();
    expect(sniffContentType(bytes(...codes("<!doctype html>")))).toBeNull();
    expect(sniffContentType(new Uint8Array())).toBeNull();
  });
});
