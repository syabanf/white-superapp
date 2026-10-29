import { describe, it, expect } from "vitest";
import {
  composeBody,
  contrastLevel,
  contrastRatio,
  formatHashtags,
  isHexColor,
  joinSampleCaptions,
  normalizeHex,
  parseHashtagList,
  parseKeywords,
  parseLines,
  parseSampleCaptions,
  readableTextOn,
  splitBody,
} from "@/features/brand/lib";
import { aspectRatio, designFilename, designerBrief, isSvgDataUrl, parseLayout } from "@/features/design-studio/lib";
import { mockAiStudio } from "@/lib/providers/ai-studio/mock";
import type { BrandContext } from "@/lib/providers/ai-studio/types";

describe("hex colors", () => {
  it("normalizes 3/6 digit hex with or without #", () => {
    expect(normalizeHex("#ABC")).toBe("#aabbcc");
    expect(normalizeHex("2B2265")).toBe("#2b2265");
    expect(normalizeHex(" #2b2265 ")).toBe("#2b2265");
  });
  it("rejects anything else", () => {
    expect(normalizeHex("#12345")).toBeNull();
    expect(normalizeHex("red")).toBeNull();
    expect(isHexColor("#ggg000")).toBe(false);
  });
});

describe("list parsing", () => {
  it("parses lines, drops bullets, blanks and duplicates", () => {
    expect(parseLines("- Pakai kamu\n\n• Pakai kamu\n* Singkat")).toEqual(["Pakai kamu", "Singkat"]);
  });
  it("caps the number of entries", () => {
    expect(parseLines("a\nb\nc", 2)).toEqual(["a", "b"]);
  });
  it("parses keywords by comma or newline", () => {
    expect(parseKeywords("kopi susu, kopi gula aren\nkopi susu")).toEqual(["kopi susu", "kopi gula aren"]);
  });
  it("parses hashtags without # and round-trips", () => {
    const tags = parseHashtagList("#kopi #KOPI, ngopi ##pagi");
    expect(tags).toEqual(["kopi", "ngopi", "pagi"]);
    expect(formatHashtags(tags)).toBe("#kopi #ngopi #pagi");
  });
  it("splits sample captions on --- lines", () => {
    const list = ["Halo\nbaris dua", "Caption kedua"];
    expect(parseSampleCaptions(joinSampleCaptions(list))).toEqual(list);
    expect(parseSampleCaptions("satu\n-----\n\n---\ndua")).toEqual(["satu", "dua"]);
  });
});

describe("contrast", () => {
  it("matches WCAG extremes", () => {
    expect(contrastRatio("#000000", "#ffffff")).toBe(21);
    expect(contrastRatio("#777777", "#777777")).toBe(1);
  });
  it("picks a readable text color", () => {
    expect(readableTextOn("#2b2265")).toBe("#ffffff");
    expect(readableTextOn("#fbeaf2")).toBe("#111111");
  });
  it("grades AA thresholds", () => {
    expect(contrastLevel(4.5)).toBe("good");
    expect(contrastLevel(3.2)).toBe("large-only");
    expect(contrastLevel(2)).toBe("poor");
  });
});

describe("caption body", () => {
  it("splits trailing hashtag lines off the body", () => {
    expect(splitBody("Kopi pagi #enak hari ini\n\n#kopi #ngopi\n#pagi\n")).toEqual({
      caption: "Kopi pagi #enak hari ini",
      hashtags: ["kopi", "ngopi", "pagi"],
    });
  });
  it("composes hook, caption and tags", () => {
    expect(composeBody({ hook: "Halo!", caption: "Isi caption", hashtags: ["kopi"] })).toBe("Halo!\n\nIsi caption\n\n#kopi");
    expect(composeBody({ hook: "Isi", caption: "Isi caption", hashtags: [] })).toBe("Isi caption");
  });
});

const brand: BrandContext = {
  clientName: "Kopi Nusantara",
  industry: "F&B",
  primaryColor: "#3b2416",
  secondaryColor: "#f4e7d7",
  accentColors: ["#d98c3f"],
  headingFont: "Poppins",
  bodyFont: "Inter",
  toneOfVoice: "hangat",
  audience: "pekerja muda",
  doList: [],
  dontList: ["janji berlebihan"],
  keywords: ["kopi susu"],
  hashtags: ["kopinusantara"],
  sampleCaptions: [],
  visualStyle: "foto terang",
};

describe("design studio", () => {
  it("reads layout JSON tolerantly", () => {
    const l = parseLayout({ headline: "Hai", composition: "nope", palette: ["#fff", "red", 3], elements: ["a", 1] });
    expect(l).toMatchObject({ headline: "Hai", subline: "", composition: "center-stack", palette: ["#fff"], elements: ["a"] });
    expect(parseLayout(null).headline).toBe("");
  });
  it("builds file names, aspect ratios and the designer brief", () => {
    expect(designFilename("Promo Kopi Susu!", "FEED_SQUARE", "png")).toBe("promo-kopi-susu-feed-square.png");
    expect(designFilename("", "STORY", "svg")).toBe("desain-story.svg");
    expect(aspectRatio("FEED_PORTRAIT")).toBe("1080 / 1350");
    const text = designerBrief({ title: "Promo", brief: "Brief", format: "STORY", layout: parseLayout({ headline: "Judul", palette: ["#000000"] }) });
    expect(text).toContain("Judul: Judul");
    expect(text).toContain("1080×1920");
  });
  it("mock layout + image are deterministic and on-brand", async () => {
    const input = { brand, title: "Promo 2 gelas", brief: "Beli dua lebih hemat.", format: "FEED_SQUARE" as const };
    const a = await mockAiStudio.designLayout(input);
    const b2 = await mockAiStudio.designLayout(input);
    expect(a).toEqual(b2);
    expect(a.palette[0]).toBe(brand.primaryColor);
    const img = await mockAiStudio.generateImage({ brand, layout: a, format: "FEED_SQUARE" });
    expect(isSvgDataUrl(img.dataUrl)).toBe(true);
    expect((await mockAiStudio.generateImage({ brand, layout: a, format: "FEED_SQUARE" })).dataUrl).toBe(img.dataUrl);
  });
  it("mock caption rewrite returns 3 variants", async () => {
    const v = await mockAiStudio.rewriteCaption({ brand, hook: "", caption: "Kopi susu baru.", hashtags: [], instruction: "", platform: "INSTAGRAM" });
    expect(v).toHaveLength(3);
    expect(v.every((x) => x.caption.length > 0)).toBe(true);
  });
});
