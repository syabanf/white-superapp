import { describe, it, expect } from "vitest";
import {
  detectColumns,
  normalizeGender,
  normalizeResultType,
  parseAdsCsv,
  parseAdsDate,
  parseAdsNumber,
} from "@/features/ads/csv-import";

describe("ads csv — column detection", () => {
  it("detects English Meta Ads Manager headers", () => {
    const d = detectColumns([
      "Reporting starts",
      "Reporting ends",
      "Campaign name",
      "Ad set name",
      "Ad name",
      "Day",
      "Amount spent (IDR)",
      "Impressions",
      "Reach",
      "Link clicks",
      "Clicks (all)",
      "Results",
      "Result indicator",
      "Purchases conversion value",
    ]);
    expect(d.campaignName).toBe("Campaign name");
    expect(d.adSetName).toBe("Ad set name");
    expect(d.adName).toBe("Ad name");
    expect(d.day).toBe("Day"); // "Day" menang atas "Reporting starts"
    expect(d.reportingEnds).toBe("Reporting ends");
    expect(d.spend).toBe("Amount spent (IDR)");
    expect(d.impressions).toBe("Impressions");
    expect(d.reach).toBe("Reach");
    expect(d.linkClicks).toBe("Link clicks");
    expect(d.clicks).toBe("Clicks (all)");
    expect(d.results).toBe("Results");
    expect(d.resultType).toBe("Result indicator");
    expect(d.purchaseValue).toBe("Purchases conversion value");
    expect(d.age).toBeNull();
    expect(d.gender).toBeNull();
  });

  it("detects Indonesian Meta Ads Manager headers", () => {
    const d = detectColumns([
      "Nama kampanye",
      "Nama set iklan",
      "Nama iklan",
      "Hari",
      "Jumlah yang dibelanjakan (IDR)",
      "Tayangan",
      "Jangkauan",
      "Klik tautan",
      "Klik (semua)",
      "Hasil",
      "Indikator hasil",
      "Usia",
      "Gender",
      "Nilai konversi pembelian",
    ]);
    expect(d.campaignName).toBe("Nama kampanye");
    expect(d.adSetName).toBe("Nama set iklan");
    expect(d.adName).toBe("Nama iklan");
    expect(d.day).toBe("Hari");
    expect(d.spend).toBe("Jumlah yang dibelanjakan (IDR)");
    expect(d.impressions).toBe("Tayangan");
    expect(d.reach).toBe("Jangkauan");
    expect(d.linkClicks).toBe("Klik tautan");
    expect(d.clicks).toBe("Klik (semua)");
    expect(d.results).toBe("Hasil");
    expect(d.resultType).toBe("Indikator hasil");
    expect(d.age).toBe("Usia");
    expect(d.gender).toBe("Gender");
    expect(d.purchaseValue).toBe("Nilai konversi pembelian");
  });

  it("uses Reporting starts as the date when Day is absent, and detects IDs", () => {
    const d = detectColumns(["Reporting starts", "Campaign name", "Campaign ID", "Ad set ID", "Ad ID", "Amount spent (USD)"]);
    expect(d.day).toBe("Reporting starts");
    expect(d.campaignId).toBe("Campaign ID");
    expect(d.adSetId).toBe("Ad set ID");
    expect(d.adId).toBe("Ad ID");
    expect(d.spend).toBe("Amount spent (USD)"); // awalan cocok untuk mata uang apa pun
  });
});

describe("ads csv — number parsing", () => {
  it("parses plain and English-formatted numbers", () => {
    expect(parseAdsNumber("1234")).toBe(1234);
    expect(parseAdsNumber("1234.56")).toBe(1234.56);
    expect(parseAdsNumber("1,234,567.89")).toBe(1234567.89);
    expect(parseAdsNumber("1,234")).toBe(1234);
  });
  it("parses Indonesian-formatted numbers (thousands dots, decimal comma)", () => {
    expect(parseAdsNumber("1.234.567,89")).toBe(1234567.89);
    expect(parseAdsNumber("1.234")).toBe(1234);
    expect(parseAdsNumber("12,5")).toBe(12.5);
    expect(parseAdsNumber("Rp 1.500.000")).toBe(1500000);
  });
  it("handles empty, dashes, and garbage", () => {
    expect(parseAdsNumber("")).toBeNull();
    expect(parseAdsNumber("--")).toBeNull();
    expect(parseAdsNumber(null)).toBeNull();
    expect(parseAdsNumber(undefined)).toBeNull();
    expect(parseAdsNumber("abc")).toBeNull();
    expect(parseAdsNumber("-250")).toBe(-250);
  });
});

describe("ads csv — date parsing", () => {
  it("prefers YYYY-MM-DD", () => {
    expect(parseAdsDate("2026-08-01")).toBe("2026-08-01");
    expect(parseAdsDate("2026-8-1")).toBe("2026-08-01");
    expect(parseAdsDate("2026-08-01T00:00:00")).toBe("2026-08-01");
  });
  it("assumes DD/MM/YYYY for slashes", () => {
    expect(parseAdsDate("05/08/2026")).toBe("2026-08-05");
    expect(parseAdsDate("31/01/2026")).toBe("2026-01-31");
  });
  it("falls back to MM/DD/YYYY when DD/MM is impossible", () => {
    expect(parseAdsDate("08/15/2026")).toBe("2026-08-15");
  });
  it("rejects invalid dates", () => {
    expect(parseAdsDate("31/02/2026")).toBeNull();
    expect(parseAdsDate("2026-13-01")).toBeNull();
    expect(parseAdsDate("not a date")).toBeNull();
    expect(parseAdsDate("")).toBeNull();
  });
});

describe("ads csv — normalizers", () => {
  it("normalizes gender EN/ID", () => {
    expect(normalizeGender("male")).toBe("male");
    expect(normalizeGender("Pria")).toBe("male");
    expect(normalizeGender("female")).toBe("female");
    expect(normalizeGender("Wanita")).toBe("female");
    expect(normalizeGender("uncategorized")).toBe("unknown");
    expect(normalizeGender("")).toBeNull();
    expect(normalizeGender("All")).toBeNull();
  });
  it("normalizes result indicators to canonical types", () => {
    expect(normalizeResultType("actions:link_click")).toBe("link_click");
    expect(normalizeResultType("actions:offsite_conversion.fb_pixel_purchase")).toBe("purchase");
    expect(normalizeResultType("actions:onsite_conversion.messaging_conversation_started_7d")).toBe("messaging_conversation_started");
    expect(normalizeResultType("actions:lead")).toBe("lead");
    expect(normalizeResultType("reach")).toBe("reach");
    expect(normalizeResultType("Klik tautan")).toBe("link_click");
    expect(normalizeResultType("")).toBeNull();
  });
});

describe("ads csv — full parse", () => {
  const EN_CSV = [
    "Campaign name,Ad set name,Ad name,Day,Amount spent (IDR),Impressions,Reach,Link clicks,Clicks (all),Results,Result indicator",
    '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-01,"350,000",50000,30000,950,1200,950,actions:link_click',
    '"[Prospecting] Kopi – Traffic","Jaksel 25-34","Video 15s",2026-08-02,"275,500.50",41000,26000,801,1002,801,actions:link_click',
    '"[Retargeting] Payday","Visitors 30D","Promo 25%",2026-08-01,"150,000",20000,15000,410,520,12,actions:offsite_conversion.fb_pixel_purchase',
  ].join("\n");

  it("parses English export into daily rows", () => {
    const { rows, detected, warnings } = parseAdsCsv(EN_CSV);
    expect(warnings).toEqual([]);
    expect(detected.campaignName).toBe("Campaign name");
    expect(rows).toHaveLength(3);
    expect(rows.every((r) => r.kind === "daily")).toBe(true);
    expect(rows[0]).toMatchObject({
      campaignName: "[Prospecting] Kopi – Traffic",
      adSetName: "Jaksel 25-34",
      adName: "Video 15s",
      date: "2026-08-01",
      spend: 350000,
      impressions: 50000,
      reach: 30000,
      linkClicks: 950,
      clicks: 1200,
      results: 950,
      resultType: "link_click",
    });
    expect(rows[1]!.spend).toBeCloseTo(275500.5);
    expect(rows[2]!.resultType).toBe("purchase");
  });

  const ID_DEMO_CSV = [
    "Nama kampanye,Hari,Usia,Gender,Jumlah yang dibelanjakan (IDR),Tayangan,Jangkauan,Hasil,Indikator hasil",
    '"Kampanye A",01/08/2026,25-34,Pria,"1.250.000,50",90000,61000,300,actions:link_click',
    '"Kampanye A",01/08/2026,25-34,Wanita,"1.500.000",100000,70000,410,actions:link_click',
    '"Kampanye A",02/08/2026,,,"500.000",40000,29000,120,actions:link_click',
  ].join("\n");

  it("splits demographic rows (age/gender) from daily rows and parses ID numbers/dates", () => {
    const { rows } = parseAdsCsv(ID_DEMO_CSV);
    expect(rows).toHaveLength(3);
    const demo = rows.filter((r) => r.kind === "demographic");
    const daily = rows.filter((r) => r.kind === "daily");
    expect(demo).toHaveLength(2);
    expect(daily).toHaveLength(1);
    expect(demo[0]).toMatchObject({ age: "25-34", gender: "male", spend: 1250000.5, date: "2026-08-01" });
    expect(demo[1]).toMatchObject({ gender: "female", spend: 1500000 });
    expect(daily[0]).toMatchObject({ date: "2026-08-02", spend: 500000, age: null, gender: null });
  });

  it("fills the missing half of an age/gender breakdown with unknown", () => {
    const csv = ["Campaign name,Day,Age,Amount spent (IDR),Results", "Camp,2026-08-01,18-24,1000,5"].join("\n");
    const { rows } = parseAdsCsv(csv);
    expect(rows[0]).toMatchObject({ kind: "demographic", age: "18-24", gender: "unknown" });
  });

  it("warns and returns no rows when required columns are missing", () => {
    const { rows, warnings } = parseAdsCsv("Foo,Bar\n1,2");
    expect(rows).toEqual([]);
    expect(warnings.some((w) => w.includes("Kolom wajib"))).toBe(true);
  });

  it("skips subtotal rows without campaign name and daily rows without a date", () => {
    const csv = [
      "Campaign name,Day,Amount spent (IDR),Results",
      "Camp A,2026-08-01,1000,5",
      ",2026-08-01,9999,99", // subtotal
      "Camp A,,500,2", // tanpa tanggal
    ].join("\n");
    const { rows, warnings } = parseAdsCsv(csv);
    expect(rows).toHaveLength(1);
    expect(warnings.some((w) => w.includes("dilewati"))).toBe(true);
  });
});
