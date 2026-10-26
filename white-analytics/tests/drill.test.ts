import { describe, expect, it } from "vitest";
import { bucketForRange, bucketLabel, bucketRange, timeDrill } from "@/features/erp/shared/drill/time";
import { formatDrill, parseDrill, withParams } from "@/features/erp/shared/drill/href";
import { changeTone, computeDelta, seriesDelta, trendColor } from "@/lib/metrics";

describe("drill time helpers", () => {
  it("maps buckets to inclusive ranges", () => {
    expect(bucketRange("2026-03-01", "month")).toEqual({ from: "2026-03-01", to: "2026-03-31" });
    expect(bucketRange("2026-02", "month")).toEqual({ from: "2026-02-01", to: "2026-02-28" });
    expect(bucketRange("2026-05-10", "quarter")).toEqual({ from: "2026-04-01", to: "2026-06-30" });
    expect(bucketRange("2026-12-28", "week")).toEqual({ from: "2026-12-28", to: "2027-01-03" });
    expect(bucketRange("2026-07-04", "year")).toEqual({ from: "2026-01-01", to: "2026-12-31" });
  });
  it("drills one level finer and stops at days", () => {
    expect(timeDrill("2026-03-01", "year")).toEqual({
      from: "2026-01-01",
      to: "2026-12-31",
      bucket: "month",
    });
    expect(timeDrill("2026-03-01", "month")?.bucket).toBe("day");
    expect(timeDrill("2026-03-01", "day")).toBeNull();
  });
  it("labels buckets in Indonesian", () => {
    expect(bucketLabel("2026-03-01", "month")).toBe("Maret 2026");
    expect(bucketLabel("2026-08-01", "quarter")).toBe("K3 2026");
    expect(bucketLabel("2026-03-02", "day")).toBe("2 Mar 2026");
  });
  it("picks a bucket for a range", () => {
    expect(bucketForRange("2026-03-01", "2026-03-31")).toBe("day");
    expect(bucketForRange("2025-11-01", "2026-10-31")).toBe("month");
    expect(bucketForRange("2022-01-01", "2026-10-31")).toBe("year");
  });
});

describe("drill URL helpers", () => {
  it("round-trips the drill param, keys may hold colons", () => {
    expect(parseDrill(formatDrill("cell", "1:3"))).toEqual({ kind: "cell", key: "1:3" });
    expect(parseDrill("broken")).toBeNull();
    expect(parseDrill(null)).toBeNull();
  });
  it("patches params, drops page and removes empty values", () => {
    expect(withParams("/erp/hr", "preset=ytd&page=3&dept=a", { dept: "b", drill: null })).toBe(
      "/erp/hr?preset=ytd&dept=b",
    );
    expect(withParams("/x", { a: ["1", "2"], b: undefined }, { c: 0 }, [])).toBe("/x?a=1&a=2&c=0");
    expect(withParams("/x", null, { a: "" })).toBe("/x");
  });
});

describe("growth coloring", () => {
  it("respects lower-is-better and stays neutral when flat", () => {
    expect(changeTone(computeDelta(120, 100))).toBe("positive");
    expect(changeTone(computeDelta(120, 100), true)).toBe("negative");
    expect(changeTone(computeDelta(5, 5))).toBe("neutral");
    expect(changeTone(-3)).toBe("negative");
    expect(changeTone(-3, true)).toBe("positive");
    expect(changeTone(null)).toBe("neutral");
    expect(trendColor(4)).toBe("var(--positive)");
  });
  it("measures a series from first to last finite value", () => {
    expect(seriesDelta([null, 10, 12, undefined, 8])?.direction).toBe("down");
    expect(seriesDelta([3])).toBeNull();
  });
});
