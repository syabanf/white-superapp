/**
 * Parser CSV ekspor Meta Ads Manager — murni (tanpa DB/server), dipakai oleh
 * server action impor dan pratinjau di client. Diuji di tests/ads-csv.test.ts.
 *
 * Mendukung header Inggris & Indonesia, angka dengan pemisah ribuan / koma
 * desimal, dan tanggal YYYY-MM-DD, DD/MM/YYYY, MM/DD/YYYY.
 */
import Papa from "papaparse";

export type CsvField =
  | "campaignName"
  | "adSetName"
  | "adName"
  | "campaignId"
  | "adSetId"
  | "adId"
  | "day"
  | "reportingEnds"
  | "spend"
  | "impressions"
  | "reach"
  | "linkClicks"
  | "clicks"
  | "results"
  | "resultType"
  | "age"
  | "gender"
  | "purchaseValue";

export type ParsedAdsRow = {
  kind: "daily" | "demographic";
  campaignName: string;
  campaignExternalId: string | null;
  adSetName: string | null;
  adSetExternalId: string | null;
  adName: string | null;
  adExternalId: string | null;
  /** ISO YYYY-MM-DD (null hanya mungkin pada baris demografi) */
  date: string | null;
  age: string | null;
  gender: "male" | "female" | "unknown" | null;
  spend: number;
  impressions: number;
  reach: number;
  clicks: number;
  linkClicks: number;
  results: number;
  resultType: string | null;
  purchaseValue: number | null;
};

export type ParseAdsCsvResult = {
  rows: ParsedAdsRow[];
  /** field → nama header asli yang terdeteksi (null bila tidak ada) */
  detected: Record<CsvField, string | null>;
  warnings: string[];
};

/** Alias header (dinormalkan: lowercase, spasi tunggal). Urutan = prioritas. */
const ALIASES: Record<CsvField, string[]> = {
  campaignName: ["campaign name", "nama kampanye"],
  adSetName: ["ad set name", "nama set iklan"],
  adName: ["ad name", "nama iklan"],
  campaignId: ["campaign id", "id kampanye"],
  adSetId: ["ad set id", "id set iklan"],
  adId: ["ad id", "id iklan"],
  day: ["day", "hari", "reporting starts", "tanggal mulai pelaporan"],
  reportingEnds: ["reporting ends", "tanggal selesai pelaporan"],
  spend: ["amount spent", "jumlah yang dibelanjakan"],
  impressions: ["impressions", "tayangan"],
  reach: ["reach", "jangkauan"],
  linkClicks: ["link clicks", "klik tautan"],
  clicks: ["clicks (all)", "klik (semua)"],
  results: ["results", "hasil"],
  resultType: ["result indicator", "result type", "indikator hasil", "tipe hasil"],
  age: ["age", "usia"],
  gender: ["gender", "jenis kelamin"],
  purchaseValue: ["purchases conversion value", "nilai konversi pembelian", "purchase conversion value"],
};

/** Field yang boleh dicocokkan dengan awalan (suffix mata uang bervariasi). */
const PREFIX_FIELDS: ReadonlySet<CsvField> = new Set(["spend", "purchaseValue"]);

function normalizeHeader(h: string): string {
  return h.replace(/^﻿/, "").trim().toLowerCase().replace(/\s+/g, " ");
}

/** Deteksi kolom berdasarkan alias EN/ID. Mengembalikan nama header asli. */
export function detectColumns(headers: string[]): Record<CsvField, string | null> {
  const normalized = headers.map((h) => ({ original: h, norm: normalizeHeader(h) }));
  const used = new Set<string>();
  const out = {} as Record<CsvField, string | null>;
  for (const field of Object.keys(ALIASES) as CsvField[]) {
    out[field] = null;
    for (const alias of ALIASES[field]) {
      const hit = normalized.find(
        (h) => !used.has(h.original) && (h.norm === alias || (PREFIX_FIELDS.has(field) && h.norm.startsWith(alias))),
      );
      if (hit) {
        out[field] = hit.original;
        used.add(hit.original);
        break;
      }
    }
  }
  return out;
}

/**
 * Angka fleksibel: "1.234.567,89" → 1234567.89 · "1,234,567.89" → 1234567.89 ·
 * "1.234" → 1234 (ribuan id) · "12,5" → 12.5 (desimal koma) · "Rp 1.500" → 1500.
 * null untuk sel kosong / tak terbaca.
 */
export function parseAdsNumber(raw: string | null | undefined): number | null {
  if (raw == null) return null;
  let s = String(raw).trim();
  if (!s || s === "--" || s === "-" || s === "–") return null;
  s = s.replace(/[^\d.,-]/g, "");
  const neg = s.startsWith("-");
  if (neg) s = s.slice(1);
  if (!s) return null;
  const lastDot = s.lastIndexOf(".");
  const lastComma = s.lastIndexOf(",");
  if (lastDot !== -1 && lastComma !== -1) {
    // Pemisah yang muncul terakhir adalah desimal; sisanya ribuan.
    if (lastDot > lastComma) s = s.replace(/,/g, "");
    else s = s.replace(/\./g, "").replace(",", ".");
  } else if (lastComma !== -1) {
    // Hanya koma: pola kelompok-3 → ribuan (EN); selain itu → desimal koma (ID).
    if (/^\d{1,3}(,\d{3})+$/.test(s)) s = s.replace(/,/g, "");
    else if ((s.match(/,/g) ?? []).length === 1) s = s.replace(",", ".");
    else return null;
  } else if (lastDot !== -1) {
    // Hanya titik: pola kelompok-3 → ribuan (ID); selain itu → desimal titik (EN).
    if (/^\d{1,3}(\.\d{3})+$/.test(s)) s = s.replace(/\./g, "");
    else if ((s.match(/\./g) ?? []).length > 1) return null;
  }
  const n = Number(s);
  return Number.isFinite(n) ? (neg ? -n : n) : null;
}

function isoDate(y: number, m: number, d: number): string | null {
  if (m < 1 || m > 12 || d < 1 || d > 31) return null;
  const dt = new Date(Date.UTC(y, m - 1, d));
  if (dt.getUTCFullYear() !== y || dt.getUTCMonth() !== m - 1 || dt.getUTCDate() !== d) return null;
  return `${String(y).padStart(4, "0")}-${String(m).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
}

/**
 * Tanggal → ISO YYYY-MM-DD. Prioritas YYYY-MM-DD; untuk garis miring
 * diasumsikan DD/MM/YYYY kecuali interpretasi itu mustahil (bagian kedua > 12
 * sedangkan bagian pertama ≤ 12 → MM/DD/YYYY).
 */
export function parseAdsDate(raw: string | null | undefined): string | null {
  if (raw == null) return null;
  const s = String(raw).trim();
  if (!s) return null;
  let m = /^(\d{4})-(\d{1,2})-(\d{1,2})/.exec(s);
  if (m) return isoDate(Number(m[1]), Number(m[2]), Number(m[3]));
  m = /^(\d{1,2})[/.](\d{1,2})[/.](\d{4})$/.exec(s);
  if (m) {
    const a = Number(m[1]);
    const b = Number(m[2]);
    const y = Number(m[3]);
    // Asumsi DD/MM; bila mustahil sebagai DD/MM tetapi valid sebagai MM/DD → MM/DD.
    if (b > 12 && a <= 12) return isoDate(y, a, b);
    return isoDate(y, b, a);
  }
  return null;
}

/** Normalisasi gender Meta (EN/ID) → male | female | unknown | null (tanpa breakdown). */
export function normalizeGender(raw: string | null | undefined): "male" | "female" | "unknown" | null {
  if (raw == null) return null;
  const g = raw.trim().toLowerCase();
  if (!g || g === "all" || g === "semua") return null;
  if (["male", "pria", "laki-laki", "m"].includes(g)) return "male";
  if (["female", "wanita", "perempuan", "f"].includes(g)) return "female";
  return "unknown";
}

function normalizeAge(raw: string | null | undefined): string | null {
  if (raw == null) return null;
  const a = raw.trim();
  if (!a) return null;
  const low = a.toLowerCase();
  if (low === "all" || low === "semua") return null;
  if (low === "unknown" || low === "uncategorized" || low === "tidak diketahui") return "unknown";
  return a;
}

/**
 * Normalisasi "Result indicator" Meta → tipe hasil kanonis
 * (purchase, lead, messaging_conversation_started, link_click, reach, …).
 */
export function normalizeResultType(raw: string | null | undefined): string | null {
  if (raw == null) return null;
  const v = raw.trim().toLowerCase();
  if (!v) return null;
  if (v.includes("purchase") || v.includes("pembelian")) return "purchase";
  if (v.includes("messaging_conversation") || v.includes("messaging conversation") || v.includes("percakapan")) return "messaging_conversation_started";
  if (v.includes("lead") || v.includes("prospek")) return "lead";
  if (v.includes("link_click") || v.includes("link click") || v.includes("klik tautan")) return "link_click";
  if (v.includes("thruplay")) return "thruplay";
  if (v.includes("video_view") || v.includes("video view")) return "video_view";
  if (v.includes("reach") || v.includes("jangkauan")) return "reach";
  if (v.includes("impression") || v.includes("impresi") || v.includes("tayangan")) return "impressions";
  const tail = v.split(":").pop()!.split(".").pop()!.trim().replace(/\s+/g, "_");
  return tail || null;
}

const MAX_WARNINGS = 8;

/** Parse isi file CSV ekspor Meta Ads Manager. */
export function parseAdsCsv(content: string): ParseAdsCsvResult {
  const warnings: string[] = [];
  const parsed = Papa.parse<Record<string, string>>(content, {
    header: true,
    skipEmptyLines: "greedy",
    transformHeader: (h) => h.replace(/^﻿/, "").trim(),
  });
  for (const err of parsed.errors.slice(0, 3)) {
    warnings.push(`Baris ${err.row != null ? err.row + 2 : "?"}: ${err.message}`);
  }
  const headers = parsed.meta.fields ?? [];
  const detected = detectColumns(headers);

  const missing: string[] = [];
  if (!detected.campaignName) missing.push("Campaign name / Nama kampanye");
  if (!detected.spend) missing.push("Amount spent / Jumlah yang dibelanjakan");
  if (missing.length > 0) {
    warnings.push(`Kolom wajib tidak ditemukan: ${missing.join(", ")}.`);
    return { rows: [], detected, warnings };
  }
  if (!detected.day) {
    warnings.push("Kolom tanggal (Day / Hari / Reporting starts) tidak ditemukan — baris harian tanpa tanggal dilewati.");
  }

  const get = (row: Record<string, string>, field: CsvField): string | null => {
    const header = detected[field];
    if (!header) return null;
    const v = row[header];
    return v == null || String(v).trim() === "" ? null : String(v);
  };
  const num = (row: Record<string, string>, field: CsvField): number => parseAdsNumber(get(row, field)) ?? 0;

  const rows: ParsedAdsRow[] = [];
  let skippedNoName = 0;
  let skippedNoDate = 0;
  let badDates = 0;

  for (const raw of parsed.data) {
    const campaignName = get(raw, "campaignName")?.trim() ?? null;
    if (!campaignName) {
      skippedNoName++;
      continue; // baris subtotal/footer ekspor Meta
    }
    const age = normalizeAge(get(raw, "age"));
    const gender = normalizeGender(get(raw, "gender"));
    const kind: ParsedAdsRow["kind"] = age != null || gender != null ? "demographic" : "daily";

    const dayRaw = get(raw, "day");
    const date = parseAdsDate(dayRaw);
    if (dayRaw != null && date == null) badDates++;
    if (kind === "daily" && date == null) {
      skippedNoDate++;
      continue;
    }

    rows.push({
      kind,
      campaignName,
      campaignExternalId: get(raw, "campaignId")?.trim() ?? null,
      adSetName: get(raw, "adSetName")?.trim() ?? null,
      adSetExternalId: get(raw, "adSetId")?.trim() ?? null,
      adName: get(raw, "adName")?.trim() ?? null,
      adExternalId: get(raw, "adId")?.trim() ?? null,
      date,
      age: age ?? (gender != null ? "unknown" : null),
      gender: gender ?? (age != null ? "unknown" : null),
      spend: num(raw, "spend"),
      impressions: Math.round(num(raw, "impressions")),
      reach: Math.round(num(raw, "reach")),
      clicks: Math.round(num(raw, "clicks")),
      linkClicks: Math.round(num(raw, "linkClicks")),
      results: Math.round(num(raw, "results")),
      resultType: normalizeResultType(get(raw, "resultType")),
      purchaseValue: parseAdsNumber(get(raw, "purchaseValue")),
    });
  }

  if (skippedNoDate > 0) warnings.push(`${skippedNoDate} baris harian dilewati karena tanggal kosong/tak terbaca.`);
  if (badDates > 0 && badDates !== skippedNoDate) warnings.push(`${badDates} nilai tanggal tidak terbaca.`);
  if (skippedNoName > 0 && rows.length === 0) warnings.push(`${skippedNoName} baris tanpa nama kampanye dilewati.`);
  if (rows.length === 0 && parsed.data.length > 0 && warnings.length === 0) {
    warnings.push("Tidak ada baris data yang bisa dibaca dari file.");
  }

  return { rows, detected, warnings: warnings.slice(0, MAX_WARNINGS) };
}
