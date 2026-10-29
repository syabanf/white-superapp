/**
 * Pure helpers for the Brand Kit and the caption AI (no server or DOM imports,
 * safe in client components and tests).
 */

export const BRAND_DEFAULTS = { primaryColor: "#2b2265", secondaryColor: "#fbeaf2" } as const;

export const BRAND_LIMITS = {
  accentColors: 4,
  font: 80,
  text: 1000,
  notes: 4000,
  listItems: 20,
  listItemChars: 200,
  tags: 30,
  tagChars: 60,
  sampleCaptions: 10,
  sampleCaptionChars: 2200,
} as const;

const HEX6 = /^#[0-9a-f]{6}$/i;
const HEX3 = /^#[0-9a-f]{3}$/i;

/** `#abc`, `abc`, `#AABBCC` → `#aabbcc`; anything else → null. */
export function normalizeHex(raw: string): string | null {
  const v = raw.trim();
  const withHash = v.startsWith("#") ? v : `#${v}`;
  if (HEX6.test(withHash)) return withHash.toLowerCase();
  if (HEX3.test(withHash)) {
    const [, r, g, b] = withHash.toLowerCase();
    return `#${r}${r}${g}${g}${b}${b}`;
  }
  return null;
}

export function isHexColor(raw: string): boolean {
  return normalizeHex(raw) !== null;
}

/** One entry per line (bullets and blank lines dropped), trimmed, de-duplicated case-insensitively. */
export function parseLines(raw: string, max: number = BRAND_LIMITS.listItems): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const line of raw.split(/\r?\n/)) {
    const v = line.replace(/^\s*[-*•]\s*/, "").trim();
    if (!v || seen.has(v.toLowerCase())) continue;
    seen.add(v.toLowerCase());
    out.push(v);
    if (out.length >= max) break;
  }
  return out;
}

/** Comma- or newline-separated keywords. */
export function parseKeywords(raw: string, max: number = BRAND_LIMITS.tags): string[] {
  return parseLines(raw.replace(/,/g, "\n"), max);
}

/** Hashtags from free text (`#a #b, c`) → `["a","b","c"]` without the leading `#`. */
export function parseHashtagList(raw: string, max: number = BRAND_LIMITS.tags): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const part of raw.split(/[\s,]+/)) {
    const tag = part.replace(/^#+/, "").trim();
    if (!tag || seen.has(tag.toLowerCase())) continue;
    seen.add(tag.toLowerCase());
    out.push(tag);
    if (out.length >= max) break;
  }
  return out;
}

/** `["a","#b"]` → `"#a #b"` */
export function formatHashtags(tags: string[]): string {
  return tags
    .map((t) => t.replace(/^#+/, "").trim())
    .filter(Boolean)
    .map((t) => `#${t}`)
    .join(" ");
}

/** Sample captions are separated by a line holding only `---` (captions themselves span lines). */
export const CAPTION_SEPARATOR = "---";

export function parseSampleCaptions(raw: string): string[] {
  return raw
    .split(/^\s*-{3,}\s*$/m)
    .map((s) => s.trim())
    .filter(Boolean)
    .slice(0, BRAND_LIMITS.sampleCaptions)
    .map((s) => s.slice(0, BRAND_LIMITS.sampleCaptionChars));
}

export function joinSampleCaptions(list: string[]): string {
  return list.join(`\n${CAPTION_SEPARATOR}\n`);
}

// ── Contrast (WCAG 2.x) ──────────────────────────────────────

function channel(c: number): number {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
}

export function relativeLuminance(hex: string): number {
  const h = normalizeHex(hex);
  if (!h) return 0;
  const r = parseInt(h.slice(1, 3), 16);
  const g = parseInt(h.slice(3, 5), 16);
  const b = parseInt(h.slice(5, 7), 16);
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

/** WCAG contrast ratio between two colors, 1–21 (rounded to 2 decimals). */
export function contrastRatio(a: string, b: string): number {
  const la = relativeLuminance(a);
  const lb = relativeLuminance(b);
  const [hi, lo] = la >= lb ? [la, lb] : [lb, la];
  return Math.round(((hi + 0.05) / (lo + 0.05)) * 100) / 100;
}

/** Ink or white, whichever reads better on `bg`. */
export function readableTextOn(bg: string): "#ffffff" | "#111111" {
  return contrastRatio(bg, "#ffffff") >= contrastRatio(bg, "#111111") ? "#ffffff" : "#111111";
}

export type ContrastLevel = "good" | "large-only" | "poor";

/** AA thresholds: 4.5 for body text, 3 for large headlines. */
export function contrastLevel(ratio: number): ContrastLevel {
  if (ratio >= 4.5) return "good";
  if (ratio >= 3) return "large-only";
  return "poor";
}

// ── Caption helpers (publishing composer keeps hook + caption + tags in one body) ──

/**
 * Split a single caption body into text and its hashtags. Trailing lines that
 * hold only hashtags become the tag list; inline hashtags stay in the text.
 */
export function splitBody(body: string): { caption: string; hashtags: string[] } {
  const lines = body.replace(/\s+$/, "").split(/\r?\n/);
  const tagLine = /^(\s*#[^\s#]+\s*)+$/;
  const tags: string[] = [];
  while (lines.length && (tagLine.test(lines[lines.length - 1]!) || !lines[lines.length - 1]!.trim())) {
    const line = lines.pop()!;
    if (line.trim()) tags.unshift(...parseHashtagList(line, 100));
  }
  return { caption: lines.join("\n").trim(), hashtags: parseHashtagList(tags.join(" ")) };
}

/** Hook, caption and hashtags back into one body (blank line between blocks). */
export function composeBody(v: { hook: string; caption: string; hashtags: string[] }): string {
  const hook = v.hook.trim();
  const caption = v.caption.trim();
  const blocks = [hook && !caption.startsWith(hook) ? hook : "", caption, formatHashtags(v.hashtags)];
  return blocks.filter(Boolean).join("\n\n");
}
