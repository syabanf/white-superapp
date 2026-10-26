import { describe, expect, it } from "vitest";
import { filterPosts, postsListHref } from "./post-filters";

const post = (type: string, day: number, hour: number, publishedAt: string) => ({ type, day, hour, publishedAt });
const rows = [post("Reels", 0, 19, "2026-10-19T12:00:00.000Z"), post("Foto", 2, 9, "2026-10-21T02:00:00.000Z")];

describe("social post filters", () => {
  it("filters by type, weekday-hour slot and Jakarta date", () => {
    expect(filterPosts(rows, { type: "Reels" })).toHaveLength(1);
    expect(filterPosts(rows, { day: 2, hour: 9 })[0]?.type).toBe("Foto");
    // 19 Oct 17:00 UTC+7 is still 19 Oct in Jakarta
    expect(filterPosts(rows, { date: "2026-10-19" })).toHaveLength(1);
    expect(filterPosts(rows, {})).toHaveLength(2);
  });

  it("builds the posts list href with filters and the anchor", () => {
    expect(postsListHref("/clients/x/social", { type: "Reels", day: 0 }, "from=2026-10-01")).toBe(
      "/clients/x/social?from=2026-10-01&type=Reels&day=0#posts",
    );
  });
});
