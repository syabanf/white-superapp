import { describe, it, expect } from "vitest";
import {
  avgPosition,
  backlinkMatchesView,
  classifyRankChange,
  countTop,
  difficultyBand,
  dofollowShare,
  isCacheFresh,
  isRankDropAlert,
  isToxic,
  keywordGap,
  normalizeDomain,
  positionToScore,
  rankBucket,
  rankDistribution,
  shareOfVoice,
  siteHost,
  visibility,
} from "@/lib/metrics/seo-suite";
import { expectedCtr } from "@/lib/metrics";
import { filterIdeas, parseKeywordLines, parseTags, pickSeriesDomains, summarizeIdeas, topCounts } from "@/features/seo-suite/lib";
import { mockSeoData } from "@/lib/providers/dataforseo/mock";

describe("visibility", () => {
  it("is the mean expected CTR, unranked = 0", () => {
    expect(visibility([1, 1])).toBeCloseTo(expectedCtr(1));
    expect(visibility([1, null])).toBeCloseTo(expectedCtr(1) / 2);
    expect(visibility([])).toBe(0);
    expect(visibility([150])).toBe(0);
  });
  it("average position ignores unranked keywords", () => {
    expect(avgPosition([2, 4, null])).toBe(3);
    expect(avgPosition([null])).toBeNull();
    expect(countTop([1, 3, 4, 11, null], 3)).toBe(2);
    expect(countTop([1, 3, 4, 11, null], 10)).toBe(3);
  });
});

describe("rank buckets", () => {
  it("assigns the six buckets", () => {
    expect(rankBucket(1)).toBe("1-3");
    expect(rankBucket(3)).toBe("1-3");
    expect(rankBucket(4)).toBe("4-10");
    expect(rankBucket(10)).toBe("4-10");
    expect(rankBucket(11)).toBe("11-20");
    expect(rankBucket(21)).toBe("21-50");
    expect(rankBucket(51)).toBe("51-100");
    expect(rankBucket(100)).toBe("51-100");
    expect(rankBucket(101)).toBe("unranked");
    expect(rankBucket(null)).toBe("unranked");
  });
  it("counts a distribution in fixed order", () => {
    const d = rankDistribution([1, 2, 5, 15, 30, null]);
    expect(d.map((x) => x.bucket)).toEqual(["1-3", "4-10", "11-20", "21-50", "51-100", "unranked"]);
    expect(d.map((x) => x.count)).toEqual([2, 1, 1, 1, 0, 1]);
  });
  it("sparkline score inverts position", () => {
    expect(positionToScore(1)).toBe(100);
    expect(positionToScore(100)).toBe(1);
    expect(positionToScore(null)).toBe(0);
  });
});

describe("rank change classification", () => {
  it("classifies up/down/new/lost with prev − cur delta", () => {
    expect(classifyRankChange(8, 3)).toEqual({ kind: "up", delta: 5 });
    expect(classifyRankChange(3, 8)).toEqual({ kind: "down", delta: -5 });
    expect(classifyRankChange(5, 5)).toEqual({ kind: "flat", delta: 0 });
    expect(classifyRankChange(null, 5).kind).toBe("new");
    expect(classifyRankChange(5, null).kind).toBe("lost");
    expect(classifyRankChange(null, null).kind).toBe("none");
  });
  it("alerts on > 5 drop, leaving top 10, or leaving top 100", () => {
    expect(isRankDropAlert(3, 9)).toBe(true); // 6 positions
    expect(isRankDropAlert(3, 8)).toBe(false); // 5 positions, still top 10
    expect(isRankDropAlert(9, 11)).toBe(true); // left top 10
    expect(isRankDropAlert(12, 15)).toBe(false);
    expect(isRankDropAlert(40, null)).toBe(true);
    expect(isRankDropAlert(null, 40)).toBe(false);
    expect(isRankDropAlert(8, 2)).toBe(false);
  });
});

describe("share of voice", () => {
  it("weights by expected CTR and folds the tail into Lainnya", () => {
    const lists = [
      [
        { domain: "a.id", position: 1 },
        { domain: "b.id", position: 2 },
      ],
      [
        { domain: "a.id", position: 1 },
        { domain: "c.id", position: 10 },
      ],
    ];
    const sov = shareOfVoice(lists, 1);
    expect(sov[0]!.domain).toBe("a.id");
    expect(sov[0]!.weight).toBeCloseTo(2 * expectedCtr(1));
    expect(sov[1]!.domain).toBe("Lainnya");
    expect(sov.reduce((a, r) => a + r.share, 0)).toBeCloseTo(100);
  });
});

describe("difficulty band", () => {
  it("maps KD to four bands", () => {
    expect(difficultyBand(0)).toBe("easy");
    expect(difficultyBand(29)).toBe("easy");
    expect(difficultyBand(30)).toBe("medium");
    expect(difficultyBand(59)).toBe("medium");
    expect(difficultyBand(60)).toBe("hard");
    expect(difficultyBand(80)).toBe("very_hard");
    expect(difficultyBand(null)).toBeNull();
  });
});

describe("research cache freshness", () => {
  const now = new Date("2026-09-02T10:00:00Z");
  it("is fresh under 7 days and stale after", () => {
    expect(isCacheFresh(new Date("2026-08-30T00:00:00Z"), now)).toBe(true);
    expect(isCacheFresh(new Date("2026-08-26T09:00:00Z"), now)).toBe(false);
    expect(isCacheFresh("2026-09-01T00:00:00Z", now)).toBe(true);
    expect(isCacheFresh("2026-09-03T00:00:00Z", now)).toBe(false); // from the future → refetch
    expect(isCacheFresh("garbage", now)).toBe(false);
  });
});

describe("keyword gap", () => {
  it("splits into missing / weak / strong / shared, sorted by volume", () => {
    const own = [
      { keyword: "kopi susu", position: 3, volume: 1000 },
      { keyword: "kopi tubruk", position: 12, volume: 400 },
      { keyword: "cold brew", position: 5, volume: 800 },
    ];
    const comp = [
      { keyword: "Kopi Susu", position: 8, volume: 1000 },
      { keyword: "kopi tubruk", position: 2, volume: 400 },
      { keyword: "kopi arabika", position: 4, volume: 2000 },
      { keyword: "kopi robusta", position: 9, volume: 300 },
    ];
    const gap = keywordGap(own, comp);
    expect(gap.missing.map((r) => r.keyword)).toEqual(["kopi arabika", "kopi robusta"]);
    expect(gap.weak.map((r) => r.keyword)).toEqual(["kopi tubruk"]);
    expect(gap.strong.map((r) => r.keyword)).toEqual(["Kopi Susu"]);
    expect(gap.shared).toHaveLength(2);
    expect(gap.missing[0]!.ownPosition).toBeNull();
  });
});

describe("backlinks", () => {
  const now = new Date("2026-09-02T00:00:00Z");
  it("flags toxic at spam score ≥ 60", () => {
    expect(isToxic(60)).toBe(true);
    expect(isToxic(59)).toBe(false);
    expect(isToxic(null)).toBe(false);
  });
  it("matches tab views", () => {
    const fresh = { firstSeen: "2026-08-20", isLost: false, spamScore: 10 };
    const old = { firstSeen: "2026-01-20", isLost: false, spamScore: 75 };
    const lost = { firstSeen: "2026-08-25", isLost: true, spamScore: 80 };
    expect(backlinkMatchesView(fresh, "new", now)).toBe(true);
    expect(backlinkMatchesView(old, "new", now)).toBe(false);
    expect(backlinkMatchesView(old, "toxic", now)).toBe(true);
    expect(backlinkMatchesView(lost, "toxic", now)).toBe(false);
    expect(backlinkMatchesView(lost, "lost", now)).toBe(true);
    expect(backlinkMatchesView(lost, "all", now)).toBe(true);
  });
  it("computes dofollow share", () => {
    expect(dofollowShare(75, 25)).toBe(75);
    expect(dofollowShare(0, 0)).toBe(0);
  });
});

describe("domain normalisation", () => {
  it("strips scheme, www, path, port", () => {
    expect(normalizeDomain("https://www.KopiKenangan.com/menu?x=1")).toBe("kopikenangan.com");
    expect(normalizeDomain("  janjijiwa.com  ")).toBe("janjijiwa.com");
    expect(normalizeDomain("http://user@shop.tokopedia.com:8080/")).toBe("shop.tokopedia.com");
    expect(normalizeDomain("sc-domain:adiharjo.co.id")).toBe("adiharjo.co.id");
  });
  it("rejects junk", () => {
    expect(normalizeDomain("")).toBeNull();
    expect(normalizeDomain("kopi")).toBeNull();
    expect(normalizeDomain("not a domain")).toBeNull();
    expect(normalizeDomain("-bad-.com")).toBeNull();
  });
  it("derives the own host from a GSC property", () => {
    expect(siteHost("https://kopinusantara.id/")).toBe("kopinusantara.id");
    expect(siteHost("sc-domain:adiharjo.co.id")).toBe("adiharjo.co.id");
  });
});

describe("suite helpers", () => {
  it("parses keyword lines and tags", () => {
    expect(parseKeywordLines(" Kopi Susu \nkopi susu\n\nkedai kopi, cold  brew")).toEqual(["kopi susu", "kedai kopi", "cold brew"]);
    expect(parseKeywordLines("a\nb\nc", 2)).toEqual(["a", "b"]);
    expect(parseTags("Brand, produk ,, brand")).toEqual(["brand", "produk"]);
  });
  it("filters ideas", () => {
    const ideas = [
      { keyword: "kopi susu harga", volume: 1000, difficulty: 20, intent: "TRANSACTIONAL" as const },
      { keyword: "cara membuat kopi", volume: 50, difficulty: 70, intent: "INFORMATIONAL" as const },
    ];
    expect(filterIdeas(ideas, { minVolume: 100, maxKd: 100, intent: "all", include: "", exclude: "" })).toHaveLength(1);
    expect(filterIdeas(ideas, { minVolume: 0, maxKd: 50, intent: "all", include: "", exclude: "" })).toHaveLength(1);
    expect(filterIdeas(ideas, { minVolume: 0, maxKd: 100, intent: "INFORMATIONAL", include: "", exclude: "" })[0]!.keyword).toBe("cara membuat kopi");
    expect(filterIdeas(ideas, { minVolume: 0, maxKd: 100, intent: "all", include: "kopi harga", exclude: "" })).toHaveLength(1);
    expect(filterIdeas(ideas, { minVolume: 0, maxKd: 100, intent: "all", include: "", exclude: "cara" })).toHaveLength(1);
    expect(summarizeIdeas(ideas)).toEqual({ count: 2, totalVolume: 1050, avgKd: 45 });
  });
  it("counts tops and picks series domains", () => {
    expect(topCounts(["a", "b", "a", ""], 2)).toEqual([
      { label: "a", count: 2 },
      { label: "b", count: 1 },
    ]);
    expect(topCounts([""], 1)).toEqual([{ label: "(kosong)", count: 1 }]);
    expect(pickSeriesDomains("me.id", { "me.id": 10, "x.id": 50, "y.id": 70, "z.id": 60, "w.id": 5 })).toEqual(["me.id", "y.id", "z.id", "x.id"]);
  });
});

describe("mock DataForSEO", () => {
  it("is deterministic and Indonesian-flavoured", async () => {
    const a = await mockSeoData.keywordIdeas("kopi susu", { locationCode: 2360, languageCode: "id", mode: "ideas", limit: 30 });
    const b = await mockSeoData.keywordIdeas("kopi susu", { locationCode: 2360, languageCode: "id", mode: "ideas", limit: 30 });
    expect(a).toEqual(b);
    expect(a.length).toBeGreaterThan(10);
    expect(a.every((i) => i.trend.length === 12 && i.difficulty >= 0 && i.difficulty <= 100 && i.cpc >= 500)).toBe(true);
    const q = await mockSeoData.keywordIdeas("kopi susu", { locationCode: 2360, languageCode: "id", mode: "questions", limit: 30 });
    expect(q.some((i) => /^(cara|apa itu|bagaimana)/.test(i.keyword))).toBe(true);
  });
  it("ranks the target consistently within a day and lists it in the top 10 when ranked", async () => {
    const [r1] = await mockSeoData.serpRanks([{ keyword: "kopi susu gula aren", device: "MOBILE" }], { locationCode: 2360, languageCode: "id", targetDomain: "kopinusantara.id" });
    const [r2] = await mockSeoData.serpRanks([{ keyword: "kopi susu gula aren", device: "MOBILE" }], { locationCode: 2360, languageCode: "id", targetDomain: "kopinusantara.id" });
    expect(r1).toEqual(r2);
    expect(r1!.top10).toHaveLength(10);
    if (r1!.position != null && r1!.position <= 10) expect(r1!.top10.find((t) => t.domain === "kopinusantara.id")?.position).toBe(r1!.position);
  });
  it("produces overlapping keyword universes for same-industry domains", async () => {
    const own = await mockSeoData.domainKeywords("kopinusantara.id", 2360, "id", 100);
    const comp = await mockSeoData.domainKeywords("kopikenangan.com", 2360, "id", 100);
    const gap = keywordGap(own, comp);
    expect(gap.shared.length).toBeGreaterThan(5);
    expect(gap.missing.length).toBeGreaterThan(0);
  });
  it("has a few toxic backlinks and some lost ones", async () => {
    const list = await mockSeoData.backlinks("kopinusantara.id", { mode: "all" });
    expect(list.some((b) => b.spamScore >= 60)).toBe(true);
    expect(list.some((b) => b.isLost)).toBe(true);
    const lost = await mockSeoData.backlinks("kopinusantara.id", { mode: "lost" });
    expect(lost.every((b) => b.isLost)).toBe(true);
  });
});
