import { describe, expect, it } from "vitest";
import { sumHours, timeWib, toQuery, weekdayDate } from "./format";
import { monthParam, todayWib } from "./perms";

describe("hr format helpers", () => {
  it("shows times in WIB", () => {
    expect(timeWib("2026-10-23T02:02:00Z")).toBe("09.02");
    expect(timeWib(null)).toBe("–");
  });
  it("labels a day with its weekday", () => {
    expect(weekdayDate("2026-10-26")).toMatch(/^Sen/);
  });
  it("builds query strings from non-empty values", () => {
    expect(toQuery({ a: "1", b: "", c: undefined, d: 2 })).toBe("?a=1&d=2");
    expect(toQuery({})).toBe("");
  });
  it("sums hours without float noise", () => {
    expect(sumHours([0.1, 0.2, 1.33])).toBe(1.63);
  });
});

describe("hr params", () => {
  it("parses or defaults the month", () => {
    expect(monthParam("2026-09")).toEqual({ ym: "2026-09", month: 9, year: 2026 });
    expect(monthParam("bad", new Date(2026, 0, 15))).toEqual({ ym: "2026-01", month: 1, year: 2026 });
  });
  it("uses the Jakarta date", () => {
    expect(todayWib(new Date("2026-10-25T18:30:00Z"))).toBe("2026-10-26");
  });
});
