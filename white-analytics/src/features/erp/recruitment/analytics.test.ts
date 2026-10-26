import { describe, expect, it } from "vitest";
import { presetRange, resolvePeriod } from "@/features/erp/shared/dashboard-period";
import { candidatesHref, overlay, rate, responseBuckets, topN } from "./analytics";
import type { DashboardOffer, RecruitmentTrendPoint } from "./types";

const week = (start: string, applications: number): RecruitmentTrendPoint => ({
  start,
  applications,
  hires: 0,
  offersSent: 0,
  offersAccepted: 0,
  offersDeclined: 0,
  offersExpired: 0,
  offersNegotiating: 0,
  inProcess: 0,
  openPositions: 0,
  openHeadcount: 0,
});

describe("dashboard period", () => {
  it("resolves presets in WIB days", () => {
    expect(presetRange("thisMonth", "2026-10-26")).toEqual({ from: "2026-10-01", to: "2026-10-26" });
    expect(presetRange("lastMonth", "2026-10-26")).toEqual({ from: "2026-09-01", to: "2026-09-30" });
    expect(presetRange("thisQuarter", "2026-10-26")).toEqual({ from: "2026-10-01", to: "2026-10-26" });
    expect(presetRange("ytd", "2026-10-26")).toEqual({ from: "2026-01-01", to: "2026-10-26" });
    expect(presetRange("12m", "2026-10-26")).toEqual({ from: "2025-10-27", to: "2026-10-26" });
  });
  it("prefers a valid custom range and falls back to the default preset", () => {
    expect(resolvePeriod({ from: "2026-08-01", to: "2026-08-31", compare: "year" })).toMatchObject({
      preset: "custom",
      compare: "year",
    });
    expect(resolvePeriod({ from: "2026-09-01", to: "2026-08-01" }).preset).toBe("12m");
  });
});

describe("recruitment analytics helpers", () => {
  it("overlays the comparison weeks by position", () => {
    expect(overlay([week("2026-10-19", 4), week("2026-10-26", 2)], [week("2025-10-20", 3)], "applications")).toEqual([
      { date: "2026-10-19", value: 4, previous: 3 },
      { date: "2026-10-26", value: 2, previous: null },
    ]);
  });
  it("folds the tail into one row", () => {
    const rows = topN([1, 5, 3, 2, 4], 2, (n) => n, (rest) => rest.reduce((a, b) => a + b, 0));
    expect(rows).toEqual([5, 4, 6]);
  });
  it("buckets decided offers by response days", () => {
    const offer = (status: string, responseDays: number | null) => ({ status, responseDays }) as DashboardOffer;
    expect(responseBuckets([offer("accepted", 0.5), offer("declined", 2), offer("accepted", 9), offer("sent", null)])).toEqual([1, 1, 0, 1]);
  });
  it("builds drill-down links with the period", () => {
    expect(candidatesHref({ from: "2026-01-01", to: "2026-01-31", job: "j1" }, { status: "hired" })).toBe(
      "/erp/hr/recruitment/candidates?status=hired&job=j1&from=2026-01-01&to=2026-01-31",
    );
    expect(rate(1, 0)).toBeNull();
  });
});
