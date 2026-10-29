import { ProviderError } from "@/lib/action-result";
import { openRouterModel } from "@/lib/providers/openrouter/real";
import { mockLayout } from "./mock";
import {
  COMPOSITIONS,
  FORMAT_SIZE,
  type AiStudioProvider,
  type BrandContext,
  type CampaignSuggestion,
  type CaptionVariant,
  type DesignLayout,
} from "./types";

/**
 * OpenRouter adapter. Text tasks ask for a JSON object and validate the shape;
 * a malformed answer falls back to the deterministic layout so the user still
 * gets something usable. Images use an image-output model (modalities
 * ["image","text"]) and come back as a base64 data URL.
 */

const API_URL = "https://openrouter.ai/api/v1/chat/completions";
const TEXT_TIMEOUT_MS = 60_000;
const IMAGE_TIMEOUT_MS = 120_000;
export const DEFAULT_IMAGE_MODEL = "google/gemini-3.1-flash-image";

export function imageModel(): string {
  return process.env.OPENROUTER_IMAGE_MODEL || DEFAULT_IMAGE_MODEL;
}

function brandBrief(b: BrandContext): string {
  const lines = [
    `Brand: ${b.clientName}${b.industry ? ` (${b.industry})` : ""}`,
    b.toneOfVoice && `Tone of voice: ${b.toneOfVoice}`,
    b.audience && `Audiens: ${b.audience}`,
    `Warna: primer ${b.primaryColor}, sekunder ${b.secondaryColor}${b.accentColors.length ? `, aksen ${b.accentColors.join(", ")}` : ""}`,
    (b.headingFont || b.bodyFont) && `Font: judul ${b.headingFont || "-"}, teks ${b.bodyFont || "-"}`,
    b.visualStyle && `Gaya visual: ${b.visualStyle}`,
    b.keywords.length > 0 && `Kata kunci: ${b.keywords.join(", ")}`,
    b.hashtags.length > 0 && `Hashtag brand: ${b.hashtags.join(" ")}`,
    b.doList.length > 0 && `Lakukan: ${b.doList.join("; ")}`,
    b.dontList.length > 0 && `Jangan: ${b.dontList.join("; ")}`,
    b.sampleCaptions.length > 0 && `Contoh caption yang disukai:\n- ${b.sampleCaptions.slice(0, 3).join("\n- ")}`,
  ];
  return lines.filter(Boolean).join("\n");
}

async function post(body: unknown, timeoutMs: number): Promise<{ content?: string; images?: string[] }> {
  const apiKey = process.env.OPENROUTER_API_KEY;
  if (!apiKey) throw new ProviderError("openrouter", "PERMISSION", "OPENROUTER_API_KEY belum dikonfigurasi.");
  let res: Response;
  try {
    res = await fetch(API_URL, {
      method: "POST",
      signal: AbortSignal.timeout(timeoutMs),
      headers: {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
        "HTTP-Referer": process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3000",
        "X-Title": "Mixist",
      },
      body: JSON.stringify(body),
    });
  } catch (cause) {
    throw new ProviderError("openrouter", "NETWORK", "Tidak dapat menghubungi OpenRouter (timeout/jaringan).", { cause });
  }
  if (!res.ok) {
    const detail = (await res.text().catch(() => "")).slice(0, 300);
    const code = res.status === 401 ? "TOKEN_EXPIRED" : res.status === 403 ? "PERMISSION" : res.status === 429 ? "RATE_LIMIT" : res.status === 404 ? "NOT_FOUND" : "UNKNOWN";
    throw new ProviderError("openrouter", code, `OpenRouter mengembalikan status ${res.status}. ${detail}`);
  }
  type Msg = { content?: string; images?: { image_url?: { url?: string } }[] };
  const json = (await res.json().catch(() => null)) as { choices?: { message?: Msg }[] } | null;
  const msg = json?.choices?.[0]?.message;
  return {
    content: msg?.content?.trim(),
    images: (msg?.images ?? []).map((i) => i.image_url?.url ?? "").filter((u) => u.startsWith("data:image/")),
  };
}

async function chatJson<T>(system: string, user: string, maxTokens = 1500): Promise<T | null> {
  const { content } = await post(
    {
      model: openRouterModel(),
      max_tokens: maxTokens,
      temperature: 0.7,
      response_format: { type: "json_object" },
      messages: [
        { role: "system", content: `${system}\nJawab HANYA dengan satu objek JSON valid, tanpa markdown.` },
        { role: "user", content: user },
      ],
    },
    TEXT_TIMEOUT_MS,
  );
  if (!content) return null;
  try {
    return JSON.parse(content.replace(/^```(?:json)?\s*|\s*```$/g, "")) as T;
  } catch {
    return null;
  }
}

const str = (v: unknown, fallback = "") => (typeof v === "string" && v.trim() ? v.trim() : fallback);
const strArr = (v: unknown) => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string" && x.trim() !== "") : []);

export const realAiStudio: AiStudioProvider = {
  id: "openrouter",
  get model() {
    return openRouterModel();
  },

  async designLayout({ brand, title, brief, format }) {
    const fallback = mockLayout(brand, title, brief, `${title}|${brief}|${format}`);
    const { width, height } = FORMAT_SIZE[format];
    const raw = await chatJson<Partial<DesignLayout>>(
      "Kamu art director agency sosial media di Indonesia. Susun konsep layout satu desain yang patuh brand guideline.",
      [
        brandBrief(brand),
        "",
        `Format: ${format} (${width}x${height}px). Judul konten: ${title}. Brief: ${brief}`,
        `Kembalikan JSON: {"headline": maks 6 kata, "subline": maks 14 kata, "cta": maks 3 kata, "composition": salah satu ${COMPOSITIONS.join("|")}, "palette": [4 hex: latar, permukaan, teks, aksen] dari warna brand, "elements": [4-6 catatan layout untuk desainer], "imagePrompt": prompt bahasa Inggris untuk model gambar, tanpa teks/logo di gambar}`,
      ].join("\n"),
    );
    if (!raw) return fallback;
    const composition = COMPOSITIONS.includes(raw.composition as never) ? (raw.composition as DesignLayout["composition"]) : fallback.composition;
    const palette = strArr(raw.palette).filter((c) => /^#[0-9a-f]{6}$/i.test(c));
    return {
      headline: str(raw.headline, fallback.headline),
      subline: str(raw.subline, fallback.subline),
      cta: str(raw.cta, fallback.cta),
      composition,
      palette: palette.length >= 4 ? palette.slice(0, 4) : fallback.palette,
      elements: strArr(raw.elements).length ? strArr(raw.elements).slice(0, 8) : fallback.elements,
      imagePrompt: str(raw.imagePrompt, fallback.imagePrompt),
    };
  },

  async generateImage({ brand, layout, format }) {
    const { width, height } = FORMAT_SIZE[format];
    const prompt = [
      layout.imagePrompt,
      `Brand palette: ${layout.palette.join(", ")}. Style: ${brand.visualStyle || "clean, modern, premium"}.`,
      `Aspect ratio ${width}:${height}. Leave clear space for a headline overlay. Do not render any text, letters or logos.`,
    ].join(" ");
    const { images } = await post(
      { model: imageModel(), modalities: ["image", "text"], messages: [{ role: "user", content: prompt }] },
      IMAGE_TIMEOUT_MS,
    );
    const url = images?.[0];
    if (!url) throw new ProviderError("openrouter", "UNKNOWN", "Model gambar tidak mengembalikan gambar.");
    const mimeType = url.slice(5, url.indexOf(";")) || "image/png";
    return { dataUrl: url, mimeType, model: imageModel() };
  },

  async rewriteCaption({ brand, hook, caption, hashtags, instruction, platform }) {
    const raw = await chatJson<{ variants?: Partial<CaptionVariant>[] }>(
      "Kamu copywriter sosial media senior di Indonesia. Tulis ulang caption agar sesuai tone brand, tetap faktual, tanpa janji berlebihan.",
      [
        brandBrief(brand),
        "",
        `Platform: ${platform}`,
        `Hook sekarang: ${hook || "-"}`,
        `Caption sekarang:\n${caption || "-"}`,
        `Hashtag sekarang: ${hashtags.join(" ") || "-"}`,
        `Instruksi tambahan: ${instruction || "-"}`,
        'Kembalikan JSON {"variants":[3 objek {"label": nama singkat gaya, "hook", "caption", "hashtags": [tanpa #]}]}',
      ].join("\n"),
      2000,
    );
    const variants = (raw?.variants ?? [])
      .map((v) => ({ label: str(v.label, "Variasi"), hook: str(v.hook), caption: str(v.caption), hashtags: strArr(v.hashtags).map((h) => h.replace(/^#/, "")) }))
      .filter((v) => v.caption);
    if (variants.length === 0) throw new ProviderError("openrouter", "UNKNOWN", "AI tidak mengembalikan variasi caption yang valid.");
    return variants.slice(0, 3);
  },

  async suggestCampaign({ brand, objective, offer, dailyBudget, currency }) {
    const raw = await chatJson<Partial<CampaignSuggestion>>(
      "Kamu media buyer Meta Ads senior untuk pasar Indonesia. Usulkan setup campaign yang realistis.",
      [
        brandBrief(brand),
        "",
        `Objektif: ${objective}. Penawaran: ${offer || "-"}. Anggaran harian: ${dailyBudget} ${currency}.`,
        'Kembalikan JSON {"audience":{"locations":[...],"ageMin":n,"ageMax":n,"genders":"ALL|MALE|FEMALE","interests":[...]},"placements":[...],"creatives":[3 objek {"headline": maks 40 karakter,"primaryText": maks 125 karakter,"cta": kode CTA Meta seperti SHOP_NOW/LEARN_MORE/SIGN_UP/SEND_MESSAGE}],"notes": saran optimasi singkat}',
      ].join("\n"),
      2000,
    );
    if (!raw?.audience || !Array.isArray(raw.creatives)) throw new ProviderError("openrouter", "UNKNOWN", "AI tidak mengembalikan saran campaign yang valid.");
    const a = raw.audience;
    return {
      audience: {
        locations: strArr(a.locations).length ? strArr(a.locations) : ["Indonesia"],
        ageMin: Number.isFinite(a.ageMin) ? Math.max(13, Number(a.ageMin)) : 18,
        ageMax: Number.isFinite(a.ageMax) ? Math.min(65, Number(a.ageMax)) : 45,
        genders: a.genders === "MALE" || a.genders === "FEMALE" ? a.genders : "ALL",
        interests: strArr(a.interests).slice(0, 8),
      },
      placements: strArr(raw.placements),
      creatives: raw.creatives.slice(0, 5).map((c) => ({ headline: str(c?.headline), primaryText: str(c?.primaryText), cta: str(c?.cta, "LEARN_MORE") })),
      notes: str(raw.notes),
    };
  },
};
