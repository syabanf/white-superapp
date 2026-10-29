import { describe, expect, it } from "vitest";
import {
  accountSubject,
  detectAnomalies,
  detectSubjectAnomalies,
  isNotifiable,
  relChange,
  splitWindow,
  sumDays,
  tier,
  type AdDay,
  type AnomalySubject,
} from "@/lib/metrics/ads-anomaly";

const ANCHOR = "2026-10-28";

function iso(offset: number): string {
  const d = new Date(`${ANCHOR}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + offset);
  return d.toISOString().slice(0, 10);
}

/** A steady baseline day: Rp100k spend, 10k impressions, 5k reach, 100 clicks (1% CTR), 10 results (CPR 10k). */
function day(offset: number, over: Partial<AdDay> = {}): AdDay {
  return { date: iso(offset), spend: 100_000, impressions: 10_000, reach: 5_000, linkClicks: 100, results: 10, purchaseValue: null, ...over };
}

/** 7 baseline days (−7..−1) + the anchor day. */
function series(anchor: Partial<AdDay>, base: Partial<AdDay> = {}): AdDay[] {
  return [...Array.from({ length: 7 }, (_, i) => day(i - 7, base)), day(0, anchor)];
}

function subject(days: AdDay[], over: Partial<AnomalySubject> = {}): AnomalySubject {
  return { campaignId: "c1", name: "Promo", active: true, isConversion: true, days, ...over };
}

const rules = (s: AnomalySubject) => detectSubjectAnomalies(s, ANCHOR).map((a) => `${a.rule}:${a.severity}`);

describe("helpers", () => {
  it("relChange and tier", () => {
    expect(relChange(150, 100)).toBeCloseTo(0.5);
    expect(relChange(1, 0)).toBeNull();
    expect(tier(0.5, { info: 0.25, warning: 0.4, critical: 0.8 })).toBe("WARNING");
    expect(tier(0.9, { info: 0.25, warning: 0.4, critical: 0.8 })).toBe("CRITICAL");
    expect(tier(0.1, { info: 0.25, warning: 0.4, critical: 0.8 })).toBeNull();
    expect(tier(-0.35, { info: -0.2, warning: -0.3, critical: -0.5 })).toBe("WARNING");
    expect(tier(-0.1, { info: -0.2, warning: -0.3, critical: -0.5 })).toBeNull();
  });

  it("splitWindow fills missing days with zeros and sums duplicate dates", () => {
    const { current, baseline } = splitWindow([day(-1), day(-1), day(0)], ANCHOR);
    expect(baseline).toHaveLength(7);
    expect(baseline[6]!.spend).toBe(200_000);
    expect(baseline[0]!.spend).toBe(0);
    expect(current.spend).toBe(100_000);
  });

  it("sumDays keeps purchaseValue null when never reported", () => {
    expect(sumDays([day(0), day(1)]).purchaseValue).toBeNull();
    expect(sumDays([day(0, { purchaseValue: 5 }), day(1)]).purchaseValue).toBe(5);
  });

  it("only WARNING/CRITICAL notify", () => {
    expect(isNotifiable("INFO")).toBe(false);
    expect(isNotifiable("WARNING")).toBe(true);
    expect(isNotifiable("CRITICAL")).toBe(true);
  });
});

describe("steady campaign", () => {
  it("raises nothing", () => {
    expect(rules(subject(series({})))).toEqual([]);
  });
});

describe("CPR_SPIKE", () => {
  it("WARNING at +50%, CRITICAL at +100%, INFO at +30%", () => {
    expect(rules(subject(series({ results: 10, spend: 150_000 })))).toContain("CPR_SPIKE:WARNING");
    expect(rules(subject(series({ results: 5 })))).toContain("CPR_SPIKE:CRITICAL");
    expect(rules(subject(series({ results: 10, spend: 130_000 })))).toContain("CPR_SPIKE:INFO");
  });

  it("guards: too few results today, too few in baseline, non-conversion objective", () => {
    expect(rules(subject(series({ results: 2 })))).not.toContain("CPR_SPIKE:CRITICAL");
    expect(rules(subject(series({ results: 2 }))).some((r) => r.startsWith("CPR_SPIKE"))).toBe(false);
    expect(rules(subject(series({ results: 3, spend: 100_000 }, { results: 1 }))).some((r) => r.startsWith("CPR_SPIKE"))).toBe(false);
    expect(rules(subject(series({ results: 5 }), { isConversion: false })).some((r) => r.startsWith("CPR_SPIKE"))).toBe(false);
  });

  it("spend without any result is CRITICAL once it costs 2× the usual CPR", () => {
    const a = detectSubjectAnomalies(subject(series({ results: 0, spend: 25_000 })), ANCHOR).find((x) => x.rule === "CPR_SPIKE");
    expect(a?.severity).toBe("CRITICAL");
    expect(a?.message).toContain("tanpa hasil");
    expect(rules(subject(series({ results: 0, spend: 15_000 }))).some((r) => r.startsWith("CPR_SPIKE"))).toBe(false);
  });

  it("needs ≥ 4 delivering baseline days", () => {
    const days = [day(-3), day(-2), day(-1), day(0, { results: 5 })];
    expect(rules(subject(days))).toEqual([]);
  });

  it("writes an Indonesian message with both values", () => {
    const a = detectSubjectAnomalies(subject(series({ results: 5 })), ANCHOR)[0]!;
    expect(a.metric).toBe("cpr");
    expect(a.current).toBe(20_000);
    expect(a.baseline).toBe(10_000);
    expect(a.message).toMatch(/Biaya per hasil kampanye "Promo" naik 100%/);
    expect(a.message).not.toContain("—");
  });
});

describe("CTR_DROP", () => {
  it("WARNING at −40%, CRITICAL at −60%", () => {
    expect(rules(subject(series({ linkClicks: 60 })))).toContain("CTR_DROP:WARNING");
    expect(rules(subject(series({ linkClicks: 40 })))).toContain("CTR_DROP:CRITICAL");
  });
  it("skips low-impression days", () => {
    expect(rules(subject(series({ impressions: 900, linkClicks: 1, reach: 500 }))).some((r) => r.startsWith("CTR_DROP"))).toBe(false);
  });
});

describe("ROAS_DROP", () => {
  it("fires only when purchase value is reported", () => {
    const withPv = series({ purchaseValue: 200_000 }, { purchaseValue: 400_000 });
    expect(rules(subject(withPv))).toContain("ROAS_DROP:CRITICAL");
    expect(rules(subject(series({ purchaseValue: 300_000 }, { purchaseValue: 400_000 })))).toContain("ROAS_DROP:INFO");
    expect(rules(subject(series({}))).some((r) => r.startsWith("ROAS_DROP"))).toBe(false);
  });
  it("is not evaluated at account level (mixed objectives)", () => {
    const withPv = series({ purchaseValue: 200_000 }, { purchaseValue: 400_000 });
    expect(rules(subject(withPv, { campaignId: "" })).some((r) => r.startsWith("ROAS_DROP"))).toBe(false);
  });
});

describe("SPEND_SPIKE", () => {
  it("WARNING at +70%, CRITICAL at +120%", () => {
    // keep CPR flat by scaling results with spend
    expect(rules(subject(series({ spend: 170_000, results: 17, impressions: 17_000, reach: 8_500, linkClicks: 170 })))).toEqual(["SPEND_SPIKE:WARNING"]);
    expect(rules(subject(series({ spend: 220_000, results: 22, impressions: 22_000, reach: 11_000, linkClicks: 220 })))).toEqual(["SPEND_SPIKE:CRITICAL"]);
  });
  it("uses mean over delivering days, so a fresh campaign is not a spike", () => {
    const days = [day(-5), day(-4), day(-3), day(-2), day(-1), day(0)];
    expect(rules(subject(days))).toEqual([]);
  });
});

describe("SPEND_STALL", () => {
  it("WARNING after one zero day, CRITICAL after two", () => {
    const one = [...Array.from({ length: 7 }, (_, i) => day(i - 7))];
    expect(rules(subject(one))).toEqual(["SPEND_STALL:WARNING"]);
    const two = [...Array.from({ length: 6 }, (_, i) => day(i - 7))];
    const a = detectSubjectAnomalies(subject(two), ANCHOR);
    expect(a.map((x) => `${x.rule}:${x.severity}`)).toEqual(["SPEND_STALL:CRITICAL"]);
    expect(a[0]!.message).toContain("2 hari");
  });
  it("ignores paused campaigns and campaigns that barely spent", () => {
    const one = [...Array.from({ length: 7 }, (_, i) => day(i - 7))];
    expect(rules(subject(one, { active: false }))).toEqual([]);
    expect(rules(subject([day(-2), day(-1)]))).toEqual([]);
  });
});

describe("FATIGUE", () => {
  it("frequency over 3 is WARNING, over 5 CRITICAL", () => {
    expect(rules(subject(series({ reach: 2_500 }, { reach: 2_500 })))).toEqual(["FATIGUE:WARNING"]);
    expect(rules(subject(series({ reach: 1_500 }, { reach: 1_500 })))).toEqual(["FATIGUE:CRITICAL"]);
  });
  it("needs enough impressions and an active campaign", () => {
    expect(rules(subject(series({ reach: 2_500 }, { reach: 2_500 }), { active: false }))).toEqual([]);
    const tiny = series({ impressions: 400, reach: 100, linkClicks: 4, spend: 4_000, results: 0 }, { impressions: 400, reach: 100, linkClicks: 4, spend: 4_000, results: 0 });
    expect(rules(subject(tiny)).some((r) => r.startsWith("FATIGUE"))).toBe(false);
  });
});

describe("detectAnomalies + accountSubject", () => {
  it("sorts CRITICAL first, is deterministic, and tags the anchor date", () => {
    const s1 = subject(series({ results: 5 }), { campaignId: "a" }); // CPR critical
    const s2 = subject(series({ linkClicks: 60 }), { campaignId: "b" }); // CTR warning
    const out = detectAnomalies([s2, s1], ANCHOR);
    expect(out[0]!.severity).toBe("CRITICAL");
    expect(out.every((a) => a.date === ANCHOR)).toBe(true);
    expect(detectAnomalies([s2, s1], ANCHOR)).toEqual(out);
  });

  it("account level sums campaigns and only counts conversion results", () => {
    const conv = subject(series({}), { campaignId: "a" });
    const reach = subject(series({}, {}), { campaignId: "b", isConversion: false });
    const acc = accountSubject([conv, { ...reach, days: reach.days.map((d) => ({ ...d, results: 999 })) }]);
    expect(acc.campaignId).toBe("");
    const today = acc.days.find((d) => d.date === ANCHOR)!;
    expect(today.spend).toBe(200_000);
    expect(today.results).toBe(10);
    const stalled = accountSubject([subject(series({}).slice(0, 7))]);
    const a = detectSubjectAnomalies(stalled, ANCHOR)[0]!;
    expect(a.rule).toBe("SPEND_STALL");
    expect(a.message.startsWith("Akun iklan")).toBe(true);
  });
});
