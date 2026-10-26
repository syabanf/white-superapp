import { describe, expect, it } from "vitest";
import { boardFilterQuery, describeBoardFilters, matchItem, parseBoardFilters, wibMoment } from "@/features/content/board-filters";

const item = {
  status: "PUBLISHED" as const,
  plannedDate: "2026-03-10",
  publishedAt: "2026-03-11T02:30:00Z", // 09:30 WIB, Wednesday
  platform: "INSTAGRAM" as const,
  format: "REEL" as const,
  pillar: "Edukasi",
  revisionRound: 1,
};

describe("content board filters", () => {
  it("parses and round-trips the URL", () => {
    const f = parseBoardFilters({ status: "PUBLISHED,BOGUS", dateFrom: "2026-03-01", dateTo: "2026-03-31", dateBy: "published", weekday: "2", hour: "9" });
    expect(f.status).toEqual(["PUBLISHED"]);
    expect(boardFilterQuery(f).toString()).toBe("status=PUBLISHED&dateFrom=2026-03-01&dateTo=2026-03-31&dateBy=published&weekday=2&hour=9");
    expect(parseBoardFilters({ hour: "25", stage: "X" })).toMatchObject({ hour: null, stage: "" });
  });
  it("reads WIB time of a publish instant", () => {
    expect(wibMoment(item.publishedAt)).toEqual({ day: "2026-03-11", weekday: 2, hour: 9 });
  });
  it("matches by date field, stage, funnel reach, rounds and heat cell", () => {
    const today = "2026-03-20";
    const m = (sp: Record<string, string>) => matchItem(item, parseBoardFilters(sp), today);
    expect(m({ dateFrom: "2026-03-11", dateTo: "2026-03-11", dateBy: "published" })).toBe(true);
    expect(m({ dateFrom: "2026-03-11", dateTo: "2026-03-11" })).toBe(false);
    expect(m({ reached: "REVIEW" })).toBe(true);
    expect(m({ stage: "DESIGN" })).toBe(false);
    expect(m({ rounds: "any" })).toBe(true);
    expect(m({ rounds: "0" })).toBe(false);
    expect(m({ weekday: "2", hour: "9" })).toBe(true);
    expect(m({ weekday: "3" })).toBe(false);
    expect(matchItem({ ...item, status: "DESIGN", designDue: "2026-03-15" }, parseBoardFilters({ overdue: "1", stage: "DESIGN" }), today)).toBe(true);
  });
  it("describes active filters as chips", () => {
    const chips = describeBoardFilters(parseBoardFilters({ status: "PUBLISHED", platform: "INSTAGRAM", overdue: "1" }));
    expect(chips.map((c) => c.key)).toEqual(["status", "overdue", "platform"]);
  });
});
