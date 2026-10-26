import { describe, expect, it } from "vitest";
import { dashboardQuery, engagementRate, funnelSteps, kpiDelta, parseDashFilters, topWithOthers, waterfallBars } from "@/features/content/dashboard";

describe("content dashboard helpers", () => {
  it("parses URL filters and drops unknown values", () => {
    const f = parseDashFilters({ tab: "performa", cmp: "year", platform: "TIKTOK", format: "GIF", pillar: "Promo" });
    expect(f).toMatchObject({ tab: "performa", compareMode: "year", platform: "TIKTOK", format: "", pillar: "Promo" });
    expect(parseDashFilters({ tab: "x" }).tab).toBe("ringkasan");
  });

  it("builds the gateway query with camelCase filter names", () => {
    const q = new URLSearchParams(dashboardQuery("2026-01-01", "2026-01-31", { ...parseDashFilters({}), assignee: "u1", plan: "p1" }));
    expect(Object.fromEntries(q)).toEqual({ from: "2026-01-01", to: "2026-01-31", compare: "previous", assigneeId: "u1", planId: "p1" });
  });

  it("computes funnel conversion from the first and previous step", () => {
    const s = funnelSteps([
      { key: "IDEA", count: 10 },
      { key: "REVIEW", count: 8 },
      { key: "PUBLISHED", count: 6 },
    ]);
    expect(s[2]).toMatchObject({ ofFirst: 60, ofPrev: 75 });
  });

  it("folds the tail into others", () => {
    const r = topWithOthers([1, 5, 3, 2], 2, (x) => x);
    expect(r).toEqual({ top: [5, 3], others: 3, othersCount: 2 });
  });

  it("hangs waterfall losses from the running total", () => {
    const bars = waterfallBars([
      { label: "Rencana", value: 10, kind: "total" },
      { label: "Proses", value: 2, kind: "minus" },
      { label: "Gagal", value: 1, kind: "minus" },
      { label: "Terbit", value: 7, kind: "total" },
    ]);
    expect(bars.map((b) => b.base)).toEqual([0, 8, 7, 0]);
  });

  it("handles int64 strings and zero reach", () => {
    expect(kpiDelta("150", "100").pct).toBe(50);
    expect(engagementRate("50", "0")).toBe(0);
    expect(engagementRate("50", "1000")).toBe(5);
  });
});
