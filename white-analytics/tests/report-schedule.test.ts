import { describe, expect, it } from "vitest";
import {
  computeNextRun,
  isDue,
  isInsightStale,
  periodLabel,
  reportPeriod,
  scheduledReportTitle,
} from "@/features/ads-automation/lib";
import { toISODate } from "@/lib/dates";

const at = (s: string) => new Date(s);
const d = (x: Date) => toISODate(x);

describe("computeNextRun", () => {
  it("weekly → next Monday, strictly after the day", () => {
    expect(d(computeNextRun("WEEKLY", at("2026-10-29T10:00:00Z")))).toBe("2026-11-02"); // Thu
    expect(d(computeNextRun("WEEKLY", at("2026-11-01T23:59:00Z")))).toBe("2026-11-02"); // Sun
    expect(d(computeNextRun("WEEKLY", at("2026-11-02T00:00:00Z")))).toBe("2026-11-09"); // Mon
    expect(d(computeNextRun("WEEKLY", at("2026-11-02T02:00:00Z")))).toBe("2026-11-09");
  });
  it("monthly → the 1st of next month, across year end", () => {
    expect(d(computeNextRun("MONTHLY", at("2026-10-29T10:00:00Z")))).toBe("2026-11-01");
    expect(d(computeNextRun("MONTHLY", at("2026-11-01T02:00:00Z")))).toBe("2026-12-01");
    expect(d(computeNextRun("MONTHLY", at("2026-12-31T02:00:00Z")))).toBe("2027-01-01");
  });
  it("always lands on UTC midnight", () => {
    expect(computeNextRun("WEEKLY", at("2026-10-29T17:45:00Z")).getUTCHours()).toBe(0);
  });
});

describe("reportPeriod", () => {
  it("weekly covers last full Monday–Sunday week", () => {
    const p = reportPeriod("WEEKLY", at("2026-11-02T02:00:00Z")); // Monday run
    expect([d(p.from), d(p.to)]).toEqual(["2026-10-26", "2026-11-01"]);
    const q = reportPeriod("WEEKLY", at("2026-10-29T10:00:00Z")); // Thursday ("Jalankan sekarang")
    expect([d(q.from), d(q.to)]).toEqual(["2026-10-19", "2026-10-25"]);
    const sun = reportPeriod("WEEKLY", at("2026-11-01T10:00:00Z"));
    expect([d(sun.from), d(sun.to)]).toEqual(["2026-10-19", "2026-10-25"]);
  });
  it("monthly covers the previous calendar month (incl. January → December)", () => {
    const p = reportPeriod("MONTHLY", at("2026-11-01T02:00:00Z"));
    expect([d(p.from), d(p.to)]).toEqual(["2026-10-01", "2026-10-31"]);
    const j = reportPeriod("MONTHLY", at("2027-01-01T02:00:00Z"));
    expect([d(j.from), d(j.to)]).toEqual(["2026-12-01", "2026-12-31"]);
    const f = reportPeriod("MONTHLY", at("2026-03-15T02:00:00Z"));
    expect([d(f.from), d(f.to)]).toEqual(["2026-02-01", "2026-02-28"]);
  });
  it("the run scheduled by computeNextRun reports the period that just ended", () => {
    const next = computeNextRun("WEEKLY", at("2026-10-29T10:00:00Z"));
    const p = reportPeriod("WEEKLY", next);
    expect(d(p.to)).toBe(d(new Date(next.getTime() - 86_400_000)));
  });
});

describe("isDue / labels / insight cadence", () => {
  it("isDue respects enabled and nextRunAt", () => {
    const now = at("2026-11-02T02:00:00Z");
    expect(isDue({ enabled: true, nextRunAt: at("2026-11-02T00:00:00Z") }, now)).toBe(true);
    expect(isDue({ enabled: false, nextRunAt: at("2026-11-02T00:00:00Z") }, now)).toBe(false);
    expect(isDue({ enabled: true, nextRunAt: at("2026-11-09T00:00:00Z") }, now)).toBe(false);
  });
  it("titles carry the period", () => {
    const w = reportPeriod("WEEKLY", at("2026-11-02T02:00:00Z"));
    expect(periodLabel("WEEKLY", w)).toBe("26 Okt – 1 Nov 2026");
    const m = reportPeriod("MONTHLY", at("2026-11-01T02:00:00Z"));
    expect(scheduledReportTitle(" Laporan bulanan ", "MONTHLY", m)).toBe("Laporan bulanan · Oktober 2026");
  });
  it("auto insight regenerates after 7 days (1h slack for cron drift)", () => {
    const now = at("2026-11-02T02:00:00Z");
    expect(isInsightStale(null, now)).toBe(true);
    expect(isInsightStale(at("2026-10-26T02:30:00Z"), now)).toBe(true);
    expect(isInsightStale(at("2026-10-28T02:00:00Z"), now)).toBe(false);
  });
});
