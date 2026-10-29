/**
 * Pure Design Studio helpers (safe in client components, server code and tests).
 */
import {
  COMPOSITIONS,
  DESIGN_FORMATS,
  FORMAT_SIZE,
  type Composition,
  type DesignFormat,
  type DesignLayout,
} from "@/lib/providers/ai-studio/types";

export { DESIGN_FORMATS, FORMAT_SIZE };
export type { DesignFormat, DesignLayout };

export const FORMAT_LABEL: Record<DesignFormat, string> = {
  FEED_SQUARE: "Feed persegi",
  FEED_PORTRAIT: "Feed potret",
  STORY: "Story",
  REEL_COVER: "Sampul Reels",
  CAROUSEL: "Carousel (slide 1)",
};

export const COMPOSITION_LABEL: Record<Composition, string> = {
  "center-stack": "Tumpuk di tengah",
  "split-left": "Terbagi, teks di kiri",
  "bottom-band": "Pita teks di bawah",
  framed: "Berbingkai",
  "big-type": "Tipografi besar",
};

export function isDesignFormat(v: unknown): v is DesignFormat {
  return typeof v === "string" && (DESIGN_FORMATS as readonly string[]).includes(v);
}

export function formatSizeLabel(f: DesignFormat): string {
  const { width, height } = FORMAT_SIZE[f];
  return `${width}×${height}`;
}

/** CSS `aspect-ratio` value for a format, e.g. `1080 / 1350`. */
export function aspectRatio(f: DesignFormat): string {
  const { width, height } = FORMAT_SIZE[f];
  return `${width} / ${height}`;
}

const str = (v: unknown, fallback = ""): string => (typeof v === "string" ? v : fallback);
const strArr = (v: unknown): string[] => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : []);

/** Tolerant read of `DesignDraft.layout` (Json column): missing fields become safe defaults. */
export function parseLayout(json: unknown): DesignLayout {
  const o = (json && typeof json === "object" ? json : {}) as Record<string, unknown>;
  const composition = (COMPOSITIONS as readonly string[]).includes(str(o.composition))
    ? (o.composition as Composition)
    : "center-stack";
  return {
    headline: str(o.headline),
    subline: str(o.subline),
    cta: str(o.cta),
    composition,
    palette: strArr(o.palette).filter((c) => /^#[0-9a-f]{3,8}$/i.test(c)),
    elements: strArr(o.elements),
    imagePrompt: str(o.imagePrompt),
  };
}

/** Serializable draft for the UI. */
export type DraftRow = {
  id: string;
  title: string;
  brief: string;
  format: DesignFormat;
  layout: DesignLayout;
  imagePrompt: string;
  imageUrl: string | null;
  mediaAssetId: string | null;
  model: string;
  createdAt: string;
};

/** The image is an inline SVG (mock output) that must be rasterized before it can enter the library. */
export function isSvgDataUrl(url: string | null | undefined): boolean {
  return typeof url === "string" && url.startsWith("data:image/svg+xml");
}

/** Plain-text brief for the designer (copy to clipboard). */
export function designerBrief(d: Pick<DraftRow, "title" | "brief" | "format" | "layout">): string {
  const l = d.layout;
  const lines = [
    `BRIEF DESAIN: ${d.title}`,
    `Format: ${FORMAT_LABEL[d.format]} (${formatSizeLabel(d.format)} px)`,
    "",
    `Judul: ${l.headline || "-"}`,
    `Subjudul: ${l.subline || "-"}`,
    `CTA: ${l.cta || "-"}`,
    `Komposisi: ${COMPOSITION_LABEL[l.composition]}`,
    `Palet: ${l.palette.join(", ") || "-"}`,
  ];
  if (l.elements.length) lines.push("", "Catatan elemen:", ...l.elements.map((e) => `- ${e}`));
  if (d.brief.trim()) lines.push("", "Brief awal:", d.brief.trim());
  if (l.imagePrompt) lines.push("", "Prompt gambar:", l.imagePrompt);
  return lines.join("\n");
}

/** File name for a saved design: `promo-ramadan-feed-square.png`. */
export function designFilename(title: string, format: DesignFormat, ext: string): string {
  const base = title
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60);
  return `${base || "desain"}-${format.toLowerCase().replace(/_/g, "-")}.${ext}`;
}
