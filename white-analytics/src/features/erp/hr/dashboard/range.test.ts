import { describe, expect, it } from "vitest";
import { bucketEndOf, compareRange, monthEndOf, presetRange, resolveHrRange } from "./range";

describe("hr dashboard range", () => {
  const today = "2026-10-26";
  it("resolves presets in WIB days", () => {
    expect(presetRange("thisMonth", today)).toEqual({ from: "2026-10-01", to: "2026-10-26" });
    expect(presetRange("lastMonth", today)).toEqual({ from: "2026-09-01", to: "2026-09-30" });
    expect(presetRange("quarter", today)).toEqual({ from: "2026-10-01", to: today });
    expect(presetRange("quarter", "2026-08-15")).toEqual({ from: "2026-07-01", to: "2026-08-15" });
    expect(presetRange("ytd", today)).toEqual({ from: "2026-01-01", to: today });
    expect(presetRange("12m", today)).toEqual({ from: "2025-11-01", to: today });
  });
  it("prefers a valid custom range and falls back to this month", () => {
    expect(resolveHrRange({ from: "2026-03-01", to: "2026-03-31", compare: "year" }, today)).toEqual({
      preset: null,
      from: "2026-03-01",
      to: "2026-03-31",
      compare: "year",
    });
    expect(resolveHrRange({ from: "2026-03-31", to: "2026-03-01" }, today).preset).toBe("thisMonth");
    expect(resolveHrRange({ preset: "nope" }, today).compare).toBe("previous");
  });
  it("finds bucket ends", () => {
    expect(monthEndOf("2026-02-01")).toBe("2026-02-28");
    expect(monthEndOf("2026-10-01", today)).toBe(today);
    expect(bucketEndOf("2026-10-19", "week", today)).toBe("2026-10-25");
    expect(bucketEndOf("2026-10-26", "week", today)).toBe(today);
    expect(bucketEndOf("2026-10-05", "day", today)).toBe("2026-10-05");
  });
  it("matches the API comparison periods", () => {
    expect(compareRange({ preset: null, from: "2026-09-01", to: "2026-09-30", compare: "previous" })).toEqual({ from: "2026-08-02", to: "2026-08-31" });
    expect(compareRange({ preset: null, from: "2026-10-01", to: "2026-10-26", compare: "year" })).toEqual({ from: "2025-10-01", to: "2025-10-26" });
  });
});
