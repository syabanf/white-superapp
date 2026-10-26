import { describe, expect, it } from "vitest";
import { pickQuery, withParam, withQuery } from "./drill-query";

describe("drill-query", () => {
  it("keeps only the requested keys", () => {
    expect(pickQuery({ from: "2026-01-01", to: "2026-01-31", q: ["a"], x: "1" })).toBe("from=2026-01-01&to=2026-01-31");
  });

  it("adds, replaces and drops params and appends a hash", () => {
    expect(withQuery("/a", "from=1&q=old", { q: "new", bucket: null }, "list")).toBe("/a?from=1&q=new#list");
    expect(withQuery("/a", "", {})).toBe("/a");
  });

  it("sets a param on an href with query and hash", () => {
    expect(withParam("/r?from=1#keywords", "bucket", "1-3")).toBe("/r?from=1&bucket=1-3#keywords");
    expect(withParam("/r", "q", "kopi susu")).toBe("/r?q=kopi+susu");
  });
});
