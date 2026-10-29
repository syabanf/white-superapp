import { hashSeed, mulberry32 } from "@/lib/providers/meta-graph/mock";
import { renderLayoutSvg, svgDataUrl } from "./svg";
import {
  COMPOSITIONS,
  type AiStudioProvider,
  type BrandContext,
  type CampaignObjective,
  type CaptionVariant,
  type DesignLayout,
} from "./types";

/**
 * Deterministic offline AI: same input → same output, so demos and tests are
 * stable. Copy is template-based but uses the brand memory (tone, keywords,
 * hashtags, palette) so the brand kit visibly matters even without a key.
 */

const pick = <T,>(rand: () => number, list: readonly T[]): T => list[Math.floor(rand() * list.length)]!;

const CTA_BY_TONE = ["Cek sekarang", "Pesan hari ini", "Lihat detailnya", "Ambil promonya", "Kunjungi kami"];

function firstSentence(text: string, max = 60): string {
  const s = text.trim().split(/[.!?\n]/)[0] ?? "";
  return s.length > max ? `${s.slice(0, max).trimEnd()}…` : s;
}

/** Whole words up to Meta's 40-character headline limit, never cut mid-word. */
function shortHeadline(text: string, max = 40): string {
  let out = "";
  for (const w of firstSentence(text, 200).split(/\s+/)) {
    const next = out ? `${out} ${w}` : w;
    if (next.length > max) break;
    out = next;
  }
  return out || text.slice(0, max);
}

function palette(brand: BrandContext): string[] {
  return [brand.primaryColor, brand.secondaryColor, "#ffffff", brand.accentColors[0] ?? "#f59e0b"];
}

export function mockLayout(brand: BrandContext, title: string, brief: string, seed: string): DesignLayout {
  const rand = mulberry32(hashSeed(`layout:${seed}`));
  const keyword = brand.keywords[0] ?? brand.clientName;
  const headline = title.trim() || firstSentence(brief) || keyword;
  return {
    headline,
    subline: firstSentence(brief, 90) || `${brand.clientName} untuk ${brand.audience || "kamu"}`,
    cta: pick(rand, CTA_BY_TONE),
    composition: pick(rand, COMPOSITIONS),
    palette: palette(brand),
    elements: [
      `Latar ${brand.primaryColor} dengan gradasi ke warna aksen`,
      `Judul memakai ${brand.headingFont || "font judul brand"}, maksimal 6 kata`,
      `Subjudul memakai ${brand.bodyFont || "font teks brand"}`,
      "Logo di pojok bawah, ukuran 8–10% lebar",
      brand.visualStyle ? `Gaya visual: ${brand.visualStyle}` : "Foto produk sebagai fokus utama",
      ...brand.dontList.slice(0, 2).map((d) => `Hindari: ${d}`),
    ],
    imagePrompt: `Professional social media visual for ${brand.clientName} (${brand.industry ?? "brand"}). ${brief}. Color palette ${palette(brand).join(", ")}. ${brand.visualStyle}. No text, no logos.`,
  };
}

function variant(label: string, hook: string, caption: string, hashtags: string[]): CaptionVariant {
  return { label, hook, caption, hashtags };
}

export const mockAiStudio: AiStudioProvider = {
  id: "mock",
  model: "mock",

  async designLayout({ brand, title, brief, format }) {
    return mockLayout(brand, title, brief, `${brand.clientName}|${title}|${brief}|${format}`);
  },

  async generateImage({ brand, layout, format }) {
    const svg = renderLayoutSvg(layout, format, { heading: brand.headingFont, body: brand.bodyFont });
    return { dataUrl: svgDataUrl(svg), mimeType: "image/svg+xml", model: "mock" };
  },

  async rewriteCaption({ brand, hook, caption, hashtags, instruction }) {
    // Inline #tags move to the hashtag list so they are not repeated in the body.
    const inline = (caption.match(/#[\p{L}\p{N}_]+/gu) ?? []).map((h) => h.slice(1));
    const base = caption.replace(/#[\p{L}\p{N}_]+/gu, "").replace(/\s+/g, " ").trim() || hook.trim() || brand.clientName;
    const tags = [...new Set([...hashtags, ...inline, ...brand.hashtags])].slice(0, 8);
    const kw = brand.keywords[0] ? ` ${brand.keywords[0]}` : "";
    const lead = firstSentence(base, 70);
    const note = instruction.trim() ? ` (${instruction.trim()})` : "";
    return [
      variant("Lebih singkat", hook || lead, `${lead}.${kw ? ` Rasakan${kw} versi terbaik.` : ""}`, tags.slice(0, 4)),
      variant(
        "Sesuai tone brand",
        hook || `Kenalan lagi dengan ${brand.clientName}`,
        `${base}\n\n${brand.toneOfVoice ? `Gaya: ${brand.toneOfVoice}.` : ""} Yuk, mampir dan buktikan sendiri!${note}`,
        tags,
      ),
      variant("Fokus ajakan", `Jangan sampai kelewatan!`, `${lead}. Klik link di bio sekarang sebelum kehabisan.`, tags.slice(0, 5)),
    ];
  },

  async suggestCampaign({ brand, objective, offer, dailyBudget }) {
    const rand = mulberry32(hashSeed(`campaign:${brand.clientName}|${objective}|${offer}`));
    const ctaFor: Record<CampaignObjective, string> = {
      OUTCOME_AWARENESS: "LEARN_MORE",
      OUTCOME_TRAFFIC: "LEARN_MORE",
      OUTCOME_ENGAGEMENT: "SEND_MESSAGE",
      OUTCOME_LEADS: "SIGN_UP",
      OUTCOME_SALES: "SHOP_NOW",
    };
    const interests = brand.keywords.length ? brand.keywords.slice(0, 4) : [brand.industry ?? "Lifestyle", "Kuliner", "Belanja online"];
    const ageMin = pick(rand, [18, 21, 25]);
    const headline = offer.trim() ? shortHeadline(offer) : `${brand.clientName} untuk kamu`;
    return {
      audience: { locations: ["Indonesia"], ageMin, ageMax: ageMin + 20, genders: "ALL", interests },
      placements: ["instagram_feed", "instagram_stories", "instagram_reels", "facebook_feed"],
      creatives: [
        { headline, primaryText: `${offer || brand.clientName}. ${brand.toneOfVoice ? "" : "Kualitas terbaik, harga bersahabat."}`.trim(), cta: ctaFor[objective] },
        { headline: `Hanya minggu ini: ${headline}`, primaryText: `Jangan lewatkan ${offer || "penawaran spesial"} dari ${brand.clientName}.`, cta: ctaFor[objective] },
        { headline: `Kenapa ${brand.clientName}?`, primaryText: `${brand.audience ? `Untuk ${brand.audience}: ` : ""}${offer || "solusi yang kamu cari"}.`, cta: ctaFor[objective] },
      ],
      notes: [
        `Mulai dengan anggaran harian ${Math.round(dailyBudget).toLocaleString("id-ID")} selama 3–4 hari fase belajar sebelum mengubah apa pun.`,
        "Uji 3 variasi materi, matikan yang CPR-nya 30% di atas rata-rata setelah 1.000 tayangan.",
        "Gunakan placement otomatis kecuali data menunjukkan satu placement jauh lebih mahal.",
      ].join("\n"),
    };
  },
};
