import { describe, it, expect } from "vitest";
import {
  allowedTransitions,
  buildUtmUrl,
  canDelete,
  canEdit,
  canReschedule,
  canTransition,
  captionFor,
  countHashtags,
  defaultUtm,
  parseUtm,
  PLATFORM_LIMITS,
  resolvePostOutcome,
  statusAfterEdit,
  validatePost,
  V,
} from "@/features/publishing/lib";
import {
  clock,
  fromZoned,
  monthGrid,
  moveToDay,
  nextOccurrences,
  parseDateKey,
  rangeForDays,
  shiftMonths,
  startOfWeek,
  toZoned,
  tzOffsetMinutes,
  weekDays,
} from "@/features/publishing/time";
import { bestPostingTimes, DEFAULT_BEST_TIMES, failureRate, postStatusCounts } from "@/lib/metrics/publishing";
import { mockPublish, MOCK_FAIL_MARKER } from "@/lib/providers/social-publisher/mock";
import { isProviderError } from "@/lib/action-result";

const img = { kind: "IMAGE" as const };
const vid = { kind: "VIDEO" as const };
const NOW = new Date("2026-09-02T03:00:00Z");

describe("caption helpers", () => {
  it("counts hashtags, ignoring mid-word #", () => {
    expect(countHashtags("#promo #kopi_lokal test#no\n#baris")).toBe(3);
    expect(countHashtags("tanpa tagar")).toBe(0);
  });
  it("uses the override only when non-empty", () => {
    expect(captionFor("master", null)).toBe("master");
    expect(captionFor("master", "   ")).toBe("master");
    expect(captionFor("master", "khusus")).toBe("khusus");
  });
});

describe("validatePost", () => {
  it("Instagram requires media; carousel is 2–10 images only", () => {
    const ig = (media: { kind: "IMAGE" | "VIDEO" }[]) =>
      validatePost({ body: "hi", targets: [{ platform: "INSTAGRAM" }], media }, { level: "full", now: NOW });
    expect(ig([]).issues.map((i) => i.message)).toContain(V.igNeedsMedia);
    expect(ig([img]).ok).toBe(true);
    expect(ig([vid]).ok).toBe(true);
    expect(ig([img, img, img]).ok).toBe(true);
    expect(ig([img, vid]).issues.map((i) => i.message)).toContain(V.igCarouselVideo);
    expect(ig(Array(11).fill(img)).ok).toBe(false);
  });
  it("TikTok requires exactly one video; Facebook may be text-only", () => {
    const tt = (media: { kind: "IMAGE" | "VIDEO" }[]) => validatePost({ body: "x", targets: [{ platform: "TIKTOK" }], media }, { now: NOW });
    expect(tt([vid]).ok).toBe(true);
    expect(tt([img]).issues[0]!.message).toBe(V.tiktokOneVideo);
    expect(tt([vid, vid]).ok).toBe(false);
    expect(validatePost({ body: "teks saja", targets: [{ platform: "FACEBOOK" }], media: [] }, { now: NOW }).ok).toBe(true);
    expect(validatePost({ body: "", targets: [{ platform: "FACEBOOK" }], media: [] }, { now: NOW }).issues[0]!.message).toBe(V.emptyPost);
  });
  it("enforces caption and hashtag limits per platform, honouring overrides", () => {
    const long = "a".repeat(PLATFORM_LIMITS.INSTAGRAM.caption + 1);
    const r = validatePost({ body: long, targets: [{ platform: "INSTAGRAM" }, { platform: "FACEBOOK" }], media: [img] }, { now: NOW });
    expect(r.issues.filter((i) => i.scope === "INSTAGRAM")).toHaveLength(1);
    expect(r.issues.filter((i) => i.scope === "FACEBOOK")).toHaveLength(0);
    const tags = Array.from({ length: 31 }, (_, i) => `#t${i}`).join(" ");
    expect(validatePost({ body: tags, targets: [{ platform: "INSTAGRAM" }], media: [img] }, { now: NOW }).issues[0]!.message).toBe(V.tooManyHashtags(30));
    const fixed = validatePost({ body: long, targets: [{ platform: "INSTAGRAM", bodyOverride: "pendek" }], media: [img] }, { now: NOW });
    expect(fixed.ok).toBe(true);
  });
  it("draft level skips media/target requirements but keeps hard limits", () => {
    expect(validatePost({ body: "", targets: [], media: [] }, { level: "draft" }).ok).toBe(true);
    expect(validatePost({ body: "", targets: [], media: [], linkUrl: "bukan url" }, { level: "draft" }).issues[0]!.message).toBe(V.invalidLink);
  });
  it("schedule must be at least 5 minutes ahead", () => {
    const base = { body: "x", targets: [{ platform: "FACEBOOK" as const }], media: [] };
    expect(validatePost({ ...base, scheduledAt: null }, { requireSchedule: true, now: NOW }).issues[0]!.message).toBe(V.scheduleRequired);
    expect(validatePost({ ...base, scheduledAt: new Date(NOW.getTime() + 4 * 60_000) }, { requireSchedule: true, now: NOW }).issues[0]!.message).toBe(V.scheduleTooSoon);
    expect(validatePost({ ...base, scheduledAt: new Date(NOW.getTime() + 5 * 60_000) }, { requireSchedule: true, now: NOW }).ok).toBe(true);
  });
});

describe("state machine", () => {
  const admin = { role: "ADMIN" as const, isAuthor: true };
  const managerAuthor = { role: "MANAGER" as const, isAuthor: true };
  const manager = { role: "MANAGER" as const, isAuthor: false };
  const viewer = { role: "VIEWER" as const, isAuthor: false };

  it("follows draft → review → approve → schedule → publish", () => {
    expect(canTransition("DRAFT", "IN_REVIEW", manager)).toBe(true);
    expect(canTransition("IN_REVIEW", "APPROVED", manager)).toBe(true);
    expect(canTransition("APPROVED", "SCHEDULED", manager)).toBe(true);
    expect(canTransition("SCHEDULED", "PUBLISHING", manager)).toBe(true);
    expect(canTransition("DRAFT", "SCHEDULED", admin)).toBe(false);
    expect(canTransition("PUBLISHED", "DRAFT", admin)).toBe(false);
  });
  it("viewer is read-only; manager cannot approve or reject own post; admin can", () => {
    expect(allowedTransitions("IN_REVIEW", viewer)).toEqual([]);
    expect(canTransition("IN_REVIEW", "APPROVED", managerAuthor)).toBe(false);
    expect(canTransition("IN_REVIEW", "REJECTED", managerAuthor)).toBe(false);
    expect(canTransition("IN_REVIEW", "DRAFT", managerAuthor)).toBe(true);
    expect(canTransition("IN_REVIEW", "APPROVED", admin)).toBe(true);
  });
  it("system-only PUBLISHING transitions are closed to humans", () => {
    expect(allowedTransitions("PUBLISHING", admin)).toEqual([]);
    expect(canTransition("FAILED", "PUBLISHING", manager)).toBe(true);
  });
  it("edit / reschedule / delete guards", () => {
    expect(canEdit("APPROVED", "MANAGER")).toBe(true);
    expect(canEdit("SCHEDULED", "MANAGER")).toBe(false);
    expect(canEdit("DRAFT", "VIEWER")).toBe(false);
    expect(statusAfterEdit("APPROVED")).toBe("DRAFT");
    expect(statusAfterEdit("DRAFT")).toBe("DRAFT");
    expect(canReschedule("SCHEDULED", "ADMIN")).toBe(true);
    expect(canReschedule("PUBLISHED", "ADMIN")).toBe(false);
    expect(canDelete("PUBLISHING", "ADMIN")).toBe(false);
    expect(canDelete("PUBLISHED", "MANAGER")).toBe(true);
  });
});

describe("resolvePostOutcome (scheduler)", () => {
  it("publishes when all targets published", () => {
    expect(resolvePostOutcome([{ status: "PUBLISHED", attempts: 1 }, { status: "PUBLISHED", attempts: 2 }])).toBe("PUBLISHED");
  });
  it("retries until attempts are exhausted, then fails", () => {
    expect(resolvePostOutcome([{ status: "PUBLISHED", attempts: 1 }, { status: "FAILED", attempts: 1 }])).toBe("SCHEDULED");
    expect(resolvePostOutcome([{ status: "FAILED", attempts: 3 }])).toBe("FAILED");
    expect(resolvePostOutcome([{ status: "FAILED", attempts: 1 }], { manual: true })).toBe("FAILED");
    expect(resolvePostOutcome([])).toBe("FAILED");
  });
});

describe("UTM", () => {
  it("appends utm params, preserving existing query and overwriting utm_*", () => {
    expect(buildUtmUrl("https://kopi.id/promo?ref=x&utm_source=old", { source: "instagram", medium: "social", campaign: "sep" })).toBe(
      "https://kopi.id/promo?ref=x&utm_source=instagram&utm_medium=social&utm_campaign=sep",
    );
  });
  it("leaves links alone without utm or when invalid", () => {
    expect(buildUtmUrl("https://kopi.id/", {})).toBe("https://kopi.id/");
    expect(buildUtmUrl("bukan url", { source: "x" })).toBe("bukan url");
  });
  it("parses stored JSON defensively", () => {
    expect(parseUtm({ source: " ig ", medium: "", junk: 1 })).toEqual({ source: "ig" });
    expect(parseUtm(null)).toEqual({});
    expect(defaultUtm("FACEBOOK")).toEqual({ source: "facebook", medium: "social" });
  });
});

describe("timezone helpers", () => {
  it("round-trips Jakarta wall-clock time", () => {
    const at = fromZoned("2026-09-02", "14:30", "Asia/Jakarta");
    expect(at.toISOString()).toBe("2026-09-02T07:30:00.000Z");
    const z = toZoned(at, "Asia/Jakarta");
    expect(z.date).toBe("2026-09-02");
    expect(z.time).toBe("14:30");
    expect(z.weekday).toBe(2); // Wednesday, Mon=0
    expect(tzOffsetMinutes(at, "Asia/Jakarta")).toBe(420);
    expect(clock(at, "Asia/Jakarta")).toBe("14.30");
  });
  it("handles a DST zone across the transition", () => {
    expect(fromZoned("2026-03-29", "03:00", "Europe/Berlin").toISOString()).toBe("2026-03-29T01:00:00.000Z");
    expect(fromZoned("2026-01-15", "12:00", "Europe/Berlin").toISOString()).toBe("2026-01-15T11:00:00.000Z");
  });
  it("moves an instant to another day keeping the wall-clock time", () => {
    const at = fromZoned("2026-09-02", "19:00", "Asia/Jakarta");
    expect(toZoned(moveToDay(at, "2026-09-10", "Asia/Jakarta"), "Asia/Jakarta")).toMatchObject({ date: "2026-09-10", time: "19:00" });
  });
  it("finds the next occurrence of a weekday/hour slot at least 5 minutes ahead", () => {
    // NOW is Wed 2026-09-02 10:00 WIB
    const [wed10, wed11, mon9] = nextOccurrences([{ day: 2, hour: 10 }, { day: 2, hour: 11 }, { day: 0, hour: 9 }], "Asia/Jakarta", NOW);
    expect(toZoned(wed10!, "Asia/Jakarta").date).toBe("2026-09-09"); // this week's slot already passed
    expect(toZoned(wed11!, "Asia/Jakarta")).toMatchObject({ date: "2026-09-02", time: "11:00" });
    expect(toZoned(mon9!, "Asia/Jakarta").date).toBe("2026-09-07");
  });
});

describe("calendar grid", () => {
  it("builds a Monday-first 6×7 grid around the month", () => {
    const grid = monthGrid("2026-09-15");
    expect(grid).toHaveLength(6);
    expect(grid[0]![0]).toBe("2026-08-31"); // Sep 1 2026 is a Tuesday
    expect(grid[0]![1]).toBe("2026-09-01");
    expect(grid[5]![6]).toBe("2026-10-11");
  });
  it("week and month arithmetic", () => {
    expect(startOfWeek("2026-09-02")).toBe("2026-08-31");
    expect(weekDays("2026-09-02")[6]).toBe("2026-09-06");
    expect(shiftMonths("2026-01-31", 1)).toBe("2026-02-28");
    expect(parseDateKey("2026-02-30", "x")).toBe("x");
    expect(parseDateKey(["2026-09-02"], "x")).toBe("2026-09-02");
  });
  it("converts client-tz days into UTC bounds", () => {
    const r = rangeForDays(["2026-09-01", "2026-09-02"], "Asia/Jakarta");
    expect(r.from.toISOString()).toBe("2026-08-31T17:00:00.000Z");
    expect(r.to.toISOString()).toBe("2026-09-02T17:00:00.000Z");
  });
});

describe("publishing metrics", () => {
  it("ranks slots by engagement × log(posts) on distinct days, topping up with defaults", () => {
    const cells = [
      { day: 1, hour: 9, posts: 5, avgEngagement: 100 },
      { day: 1, hour: 18, posts: 5, avgEngagement: 90 },
      { day: 3, hour: 12, posts: 3, avgEngagement: 80 },
      { day: 5, hour: 20, posts: 1, avgEngagement: 999 }, // below minPosts
    ];
    const best = bestPostingTimes(cells);
    expect(best.map((b) => [b.day, b.hour])).toEqual([
      [1, 9],
      [3, 12],
      [1, 18],
    ]);
    expect(bestPostingTimes([])).toEqual(DEFAULT_BEST_TIMES);
  });
  it("counts statuses and failure rate", () => {
    const counts = postStatusCounts([{ status: "DRAFT" }, { status: "DRAFT" }, { status: "PUBLISHED" }, { status: "BOGUS" }]);
    expect(counts.DRAFT).toBe(2);
    expect(counts.FAILED).toBe(0);
    expect(failureRate([{ status: "PUBLISHED" }, { status: "FAILED" }, { status: "SCHEDULED" }])).toBe(50);
    expect(failureRate([{ status: "DRAFT" }])).toBeNull();
  });
});

describe("mock publisher", () => {
  it("is deterministic and produces a platform permalink", () => {
    const input = { platform: "INSTAGRAM" as const, accessToken: "", accountExternalId: "178", text: "halo", media: [{ url: "u", kind: "IMAGE" as const }] };
    const a = mockPublish(input);
    const b = mockPublish(input);
    expect(a).toEqual(b);
    expect(a.permalink).toMatch(/^https:\/\/www\.instagram\.com\/p\//);
    expect(mockPublish({ ...input, platform: "FACEBOOK" }).permalink).toContain("facebook.com/178/posts/");
  });
  it("fails with a ProviderError when the caption contains [fail]", () => {
    try {
      mockPublish({ platform: "FACEBOOK", accessToken: "", accountExternalId: "1", text: `x ${MOCK_FAIL_MARKER}`, media: [] });
      expect.unreachable();
    } catch (e) {
      expect(isProviderError(e)).toBe(true);
    }
  });
});
