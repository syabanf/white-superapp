import { describe, expect, it } from "vitest";
import {
  DEFAULT_AUDIENCE,
  OBJECTIVE_DEFAULTS,
  canEditDraft,
  canSubmitDraft,
  draftIssues,
  emptyCreative,
  estimateDailyReach,
  filledCreatives,
  minDailyBudget,
  mergeSuggestion,
  parseAudience,
  parseCreatives,
  parsePlacements,
  scheduleDays,
  statusAfterSave,
  todayKey,
  validateBudget,
  validateSchedule,
  type DraftInput,
} from "@/features/campaigns/lib";
import { mockAdsWriter, mockCreateCampaign, MOCK_WRITER_FAIL_MARKER } from "@/lib/providers/meta-ads-writer/mock";
import {
  buildAdSetPayload,
  buildCampaignPayload,
  buildCreativePayload,
  buildTargeting,
  normalizeAdAccountId,
  splitLocations,
  toMetaBudget,
  writerGate,
} from "@/lib/providers/meta-ads-writer/payload";
import type { WriterDraft } from "@/lib/providers/meta-ads-writer/types";
import { isProviderError } from "@/lib/action-result";

const TODAY = "2026-10-29";
const complete = (over: Partial<DraftInput> = {}): DraftInput => ({
  name: "Promo Oktober",
  objective: "OUTCOME_TRAFFIC",
  dailyBudget: 50_000,
  currency: "IDR",
  startDate: "2026-11-01",
  endDate: "2026-11-10",
  adAccountId: "acc1",
  audience: { locations: ["Jakarta"], ageMin: 21, ageMax: 40, genders: "ALL", interests: ["Kopi"] },
  placements: ["instagram_feed"],
  creatives: [{ headline: "Kopi susu", primaryText: "Diskon 20%", cta: "SHOP_NOW", mediaUrl: null }, emptyCreative(), emptyCreative()],
  ...over,
});
const opts = { today: TODAY, requireAdAccount: true };

describe("campaigns lib — budget & schedule", () => {
  it("enforces Rp10.000/day minimum for IDR", () => {
    expect(minDailyBudget("IDR")).toBe(10_000);
    expect(validateBudget(9_999, "IDR")).toMatch(/minimal/);
    expect(validateBudget(10_000, "IDR")).toBeNull();
    expect(validateBudget(0, "IDR")).not.toBeNull();
    expect(validateBudget(Number.NaN, "IDR")).not.toBeNull();
    expect(validateBudget(1, "USD")).toBeNull();
  });

  it("validates schedule order and past start", () => {
    expect(validateSchedule("2026-10-28", null, TODAY)).not.toBeNull();
    expect(validateSchedule(TODAY, null, TODAY)).toBeNull();
    expect(validateSchedule("2026-11-01", "2026-11-01", TODAY)).not.toBeNull();
    expect(validateSchedule("2026-11-01", "2026-11-02", TODAY)).toBeNull();
    expect(validateSchedule("nope", null, TODAY)).not.toBeNull();
    expect(scheduleDays("2026-11-01", "2026-11-10")).toBe(10);
    expect(scheduleDays("2026-11-01", null)).toBeNull();
  });

  it("computes today's key in the client time zone", () => {
    // 20:00 UTC on the 28th is already the 29th in Jakarta (UTC+7)
    expect(todayKey(new Date("2026-10-28T20:00:00Z"), "Asia/Jakarta")).toBe("2026-10-29");
    expect(todayKey(new Date("2026-10-28T20:00:00Z"), "UTC")).toBe("2026-10-28");
  });
});

describe("campaigns lib — completeness & status", () => {
  it("a complete draft has no issues", () => {
    expect(draftIssues(complete(), opts)).toEqual([]);
  });

  it("flags missing pieces with the right field", () => {
    const fields = (d: DraftInput, o = opts) => draftIssues(d, o).map((i) => i.field);
    expect(fields(complete({ name: " " }))).toContain("name");
    expect(fields(complete({ adAccountId: null }))).toContain("adAccount");
    expect(fields(complete({ adAccountId: null }), { ...opts, requireAdAccount: false })).not.toContain("adAccount");
    expect(fields(complete({ audience: { ...DEFAULT_AUDIENCE } }))).toContain("audience");
    expect(fields(complete({ creatives: [emptyCreative(), emptyCreative(), emptyCreative()] }))).toContain("creatives");
    const partial = complete({ creatives: [complete().creatives[0]!, { headline: "Hanya judul", primaryText: "", cta: "", mediaUrl: null }, emptyCreative()] });
    expect(draftIssues(partial, opts).filter((i) => i.field === "creative-1")).toHaveLength(2);
    expect(fields(complete({ placements: ["nope"] }))).toContain("placements");
    // empty placements = Advantage+ → allowed
    expect(fields(complete({ placements: [] }))).toEqual([]);
  });

  it("save never promotes, but demotes an incomplete READY draft", () => {
    expect(statusAfterSave(null, true)).toBe("DRAFT");
    expect(statusAfterSave("DRAFT", true)).toBe("DRAFT");
    expect(statusAfterSave("READY", true)).toBe("READY");
    expect(statusAfterSave("READY", false)).toBe("DRAFT");
    expect(canEditDraft("SUBMITTED")).toBe(false);
    expect(canSubmitDraft("READY")).toBe(true);
    expect(canSubmitDraft("DRAFT")).toBe(false);
  });
});

describe("campaigns lib — JSON parsing", () => {
  it("parses audience defensively", () => {
    expect(parseAudience(null)).toEqual(DEFAULT_AUDIENCE);
    expect(parseAudience({ locations: ["Bali", "Bali", 3], ageMin: 70, ageMax: 10, genders: "X" })).toEqual({
      locations: ["Bali"],
      ageMin: 18,
      ageMax: 65,
      genders: "ALL",
      interests: [],
    });
  });

  it("always yields three creative slots", () => {
    const list = parseCreatives([{ headline: "A", primaryText: "B", cta: "SHOP_NOW" }]);
    expect(list).toHaveLength(3);
    expect(list[0]!.mediaUrl).toBeNull();
    expect(filledCreatives(list)).toHaveLength(1);
    expect(parsePlacements(["instagram_feed", "bogus", "instagram_feed"])).toEqual(["instagram_feed"]);
  });
});

describe("campaigns lib — AI merge", () => {
  const suggestion = {
    audience: { locations: ["Indonesia"], ageMin: 25, ageMax: 45, genders: "FEMALE" as const, interests: ["Kuliner", "Kopi"] },
    placements: ["instagram_reels", "unknown"],
    creatives: [
      { headline: "AI 1", primaryText: "AI teks 1", cta: "LEARN_MORE" },
      { headline: "AI 2", primaryText: "AI teks 2", cta: "NOT_A_CTA" },
      { headline: "AI 3", primaryText: "AI teks 3", cta: "SIGN_UP" },
    ],
    notes: " Uji 3 variasi. ",
  };

  it("fills only what the user left empty", () => {
    const current = {
      audience: { locations: ["Bandung"], ageMin: 18, ageMax: 65, genders: "ALL" as const, interests: ["Kopi"] },
      placements: [],
      creatives: [{ headline: "Punya saya", primaryText: "", cta: "", mediaUrl: "/api/media/x.jpg" }, emptyCreative(), emptyCreative()],
    };
    const m = mergeSuggestion(current, suggestion, "OUTCOME_SALES");
    expect(m.audience.locations).toEqual(["Bandung"]);
    expect(m.audience.ageMin).toBe(25);
    expect(m.audience.genders).toBe("FEMALE");
    expect(m.audience.interests).toEqual(["Kopi", "Kuliner"]);
    expect(m.placements).toEqual(["instagram_reels"]);
    expect(m.creatives[0]).toEqual({ headline: "Punya saya", primaryText: "AI teks 1", cta: "LEARN_MORE", mediaUrl: "/api/media/x.jpg" });
    expect(m.creatives[1]!.cta).toBe(OBJECTIVE_DEFAULTS.OUTCOME_SALES.cta);
    expect(m.aiNotes).toBe("Uji 3 variasi.");
  });

  it("keeps the user's age, gender and placements when set", () => {
    const current = { audience: { ...DEFAULT_AUDIENCE, ageMin: 30, genders: "MALE" as const }, placements: ["facebook_feed"], creatives: [] };
    const m = mergeSuggestion(current, { ...suggestion, placements: [] }, "OUTCOME_TRAFFIC");
    expect([m.audience.ageMin, m.audience.ageMax, m.audience.genders]).toEqual([30, 65, "MALE"]);
    expect(m.placements).toEqual(["facebook_feed"]);
    expect(m.creatives).toHaveLength(3);
  });
});

describe("campaigns lib — perkiraan jangkauan", () => {
  it("returns a low–high band for IDR and null when unknown", () => {
    const r = estimateDailyReach(50_000, "IDR")!;
    expect(r.low).toBeGreaterThan(0);
    expect(r.high).toBeGreaterThan(r.low);
    expect(estimateDailyReach(50_000, "EUR")).toBeNull();
    expect(estimateDailyReach(0, "IDR")).toBeNull();
  });
});

const writerDraft: WriterDraft = {
  id: "draft_1",
  name: "Promo",
  objective: "OUTCOME_SALES",
  dailyBudget: 50_000,
  currency: "IDR",
  startDate: new Date("2026-11-01T00:00:00Z"),
  endDate: new Date("2026-11-10T00:00:00Z"),
  audience: { locations: ["Indonesia", "Jakarta"], ageMin: 21, ageMax: 40, genders: "FEMALE", interests: ["Kopi"] },
  placements: ["instagram_feed", "instagram_stories", "facebook_feed"],
  creatives: [{ headline: "H", primaryText: "T", cta: "SHOP_NOW", mediaUrl: "https://x.test/a.jpg" }],
};

describe("meta-ads-writer — gate", () => {
  const ok = { metaConfigured: true, demoClient: false, realConnection: true, scopes: ["ads_read", "ads_management"] };
  it("is real only when every condition holds", () => {
    expect(writerGate(ok)).toEqual({ mode: "real", reasons: [] });
    expect(writerGate({ ...ok, scopes: ["ads_read"] })).toEqual({ mode: "simulation", reasons: ["MISSING_ADS_MANAGEMENT"] });
    expect(writerGate({ ...ok, demoClient: true }).mode).toBe("simulation");
    expect(writerGate({ metaConfigured: false, demoClient: true, realConnection: false, scopes: [] }).reasons).toEqual([
      "META_NOT_CONFIGURED",
      "DEMO_CLIENT",
      "NO_CONNECTION",
      "MISSING_ADS_MANAGEMENT",
    ]);
  });
});

describe("meta-ads-writer — mock (simulation)", () => {
  it("returns a deterministic sim_ id and never touches Meta", async () => {
    const a = await mockAdsWriter.createCampaign(writerDraft, { accessToken: "", adAccountExternalId: "act_1" });
    const b = mockCreateCampaign(writerDraft, { adAccountExternalId: "act_1" });
    expect(a.simulated).toBe(true);
    expect(a.externalId).toMatch(/^sim_[0-9a-z]+$/);
    expect(a).toEqual(b);
    expect(a.adIds).toHaveLength(1);
    expect(mockCreateCampaign({ ...writerDraft, id: "other" }, { adAccountExternalId: "act_1" }).externalId).not.toBe(a.externalId);
  });

  it("throws a ProviderError for the [fail] marker", () => {
    try {
      mockCreateCampaign({ ...writerDraft, name: `x ${MOCK_WRITER_FAIL_MARKER}` }, { adAccountExternalId: "act_1" });
      expect.unreachable();
    } catch (e) {
      expect(isProviderError(e)).toBe(true);
    }
  });
});

describe("meta-ads-writer — payloads", () => {
  it("uses Meta currency offsets for daily_budget", () => {
    expect(toMetaBudget(50_000, "IDR")).toBe(50_000);
    expect(toMetaBudget(12.5, "USD")).toBe(1250);
    expect(normalizeAdAccountId("123")).toBe("act_123");
    expect(normalizeAdAccountId("act_123")).toBe("act_123");
  });

  it("creates the campaign paused with no special ad categories", () => {
    const p = buildCampaignPayload(writerDraft);
    expect(p).toMatchObject({ status: "PAUSED", special_ad_categories: [], objective: "OUTCOME_SALES" });
  });

  it("builds targeting from audience and placements", () => {
    expect(splitLocations(["Indonesia", "Jakarta", "SG"])).toEqual({ countries: ["ID", "SG"], places: ["Jakarta"] });
    const t = buildTargeting(writerDraft.audience, writerDraft.placements, { cities: [{ key: "2490299" }], interests: [{ id: "6003", name: "Coffee" }] });
    expect(t).toMatchObject({
      geo_locations: { countries: ["ID"], cities: [{ key: "2490299" }] },
      age_min: 21,
      age_max: 40,
      genders: [2],
      publisher_platforms: ["instagram", "facebook"],
      instagram_positions: ["stream", "story"],
      facebook_positions: ["feed"],
      flexible_spec: [{ interests: [{ id: "6003", name: "Coffee" }] }],
    });
    // nothing resolvable → falls back to Indonesia; no placements → Advantage+ (no publisher_platforms)
    const auto = buildTargeting({ ...writerDraft.audience, locations: ["Kota Antah"] }, []);
    expect(auto.geo_locations).toEqual({ countries: ["ID"] });
    expect(auto.publisher_platforms).toBeUndefined();
  });

  it("builds ad set and creative payloads", () => {
    const as = buildAdSetPayload(writerDraft, "c1", { geo_locations: { countries: ["ID"] } });
    expect(as).toMatchObject({ campaign_id: "c1", daily_budget: 50_000, status: "PAUSED", optimization_goal: "LINK_CLICKS", end_time: "2026-11-10T00:00:00.000Z" });
    const cr = buildCreativePayload(writerDraft, 0, { pageId: "p1", linkUrl: "https://kopi.test" });
    expect(cr).toMatchObject({
      object_story_spec: { page_id: "p1", link_data: { link: "https://kopi.test", name: "H", message: "T", picture: "https://x.test/a.jpg", call_to_action: { type: "SHOP_NOW" } } },
    });
  });
});
