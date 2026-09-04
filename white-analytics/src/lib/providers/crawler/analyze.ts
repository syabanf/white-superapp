/**
 * Pure HTML analysis for one crawled page: extracts SeoCrawlPage fields,
 * emits SeoIssue rows, and returns discovered links. No network, unit-testable.
 */
import * as cheerio from "cheerio";
import { ISSUE_DEFS, type CrawlIssue, type CrawlPageData, type IssueCode } from "./types";

export type AnalyzeInput = {
  url: string;
  html: string;
  statusCode: number;
  /** total request duration incl. redirects, ms */
  loadMs?: number | null;
  /** X-Robots-Tag response header (may contain "noindex") */
  xRobotsTag?: string | null;
  /** number of redirect hops followed to reach the final response */
  redirectHops?: number;
};

export type AnalyzeOutput = {
  page: CrawlPageData;
  issues: CrawlIssue[];
  internalUrls: string[];
  externalUrls: string[];
};

const TITLE_MAX = 60;
const TITLE_MIN = 30;
const META_DESC_MAX = 160;
const THIN_WORDS = 300;
const SLOW_MS = 1500;

function makeIssue(code: IssueCode, url: string, message?: string): CrawlIssue {
  const def = ISSUE_DEFS[code];
  return { code, severity: def.severity, message: message ?? def.message, url };
}

/** Resolve + normalize a link target; returns null for non-http(s)/invalid hrefs. */
function resolveLink(href: string, baseUrl: string): URL | null {
  const trimmed = href.trim();
  if (!trimmed || trimmed.startsWith("#")) return null;
  if (/^(mailto:|tel:|javascript:|data:|ftp:)/i.test(trimmed)) return null;
  try {
    const u = new URL(trimmed, baseUrl);
    if (u.protocol !== "http:" && u.protocol !== "https:") return null;
    u.hash = "";
    return u;
  } catch {
    return null;
  }
}

export function analyzeHtml(input: AnalyzeInput): AnalyzeOutput {
  const { url, html, statusCode } = input;
  const loadMs = input.loadMs ?? null;
  const $ = cheerio.load(html);
  const issues: CrawlIssue[] = [];
  const pageUrl = new URL(url);

  // ── metadata ────────────────────────────────────────────────
  const titleRaw = $("head title").first().text() || $("title").first().text();
  const title = titleRaw ? titleRaw.trim() : "";

  let metaDescription: string | null = null;
  let robotsMeta = "";
  $("meta").each((_, el) => {
    const name = ($(el).attr("name") ?? "").trim().toLowerCase();
    if (name === "description" && metaDescription == null) metaDescription = ($(el).attr("content") ?? "").trim();
    if (name === "robots") robotsMeta += ` ${($(el).attr("content") ?? "").toLowerCase()}`;
  });
  if (metaDescription === "") metaDescription = null;

  const canonicalHref = $('link[rel="canonical"]').first().attr("href")?.trim() || null;
  let canonical: string | null = null;
  if (canonicalHref) {
    try {
      canonical = new URL(canonicalHref, url).toString();
    } catch {
      canonical = canonicalHref;
    }
  }

  const noindex = robotsMeta.includes("noindex") || (input.xRobotsTag ?? "").toLowerCase().includes("noindex");
  const indexable = !noindex && statusCode < 400;

  const h1Count = $("h1").length;

  // ── visible words ──────────────────────────────────────────
  const body = $("body").clone();
  body.find("script,style,noscript,template,svg").remove();
  const text = body.text().replace(/\s+/g, " ").trim();
  const wordCount = text ? text.split(" ").filter(Boolean).length : 0;

  // ── images ─────────────────────────────────────────────────
  let imagesTotal = 0;
  let imagesMissingAlt = 0;
  $("img").each((_, el) => {
    imagesTotal += 1;
    if ($(el).attr("alt") == null) imagesMissingAlt += 1;
  });

  // ── links ──────────────────────────────────────────────────
  const internal = new Set<string>();
  const external = new Set<string>();
  $("a[href]").each((_, el) => {
    const u = resolveLink($(el).attr("href") ?? "", url);
    if (!u) return;
    if (u.origin === pageUrl.origin) {
      if (u.toString() !== pageUrl.toString()) internal.add(u.toString());
    } else {
      external.add(u.toString());
    }
  });

  // ── structured data / hreflang ─────────────────────────────
  const hasJsonLd = $('script[type="application/ld+json"]').length > 0;
  const hasHreflang = $("link[rel='alternate'][hreflang]").length > 0;

  // ── mixed content (https page loading http assets) ─────────
  let mixed = false;
  if (pageUrl.protocol === "https:") {
    $("img[src], script[src], iframe[src], video[src], audio[src], source[src], link[rel='stylesheet'][href]").each((_, el) => {
      const v = ($(el).attr("src") ?? $(el).attr("href") ?? "").trim();
      if (/^http:\/\//i.test(v)) mixed = true;
    });
  }

  // ── issues (content checks only for OK responses) ──────────
  if (statusCode < 400) {
    if (!title) issues.push(makeIssue("MISSING_TITLE", url));
    else if (title.length > TITLE_MAX) issues.push(makeIssue("TITLE_TOO_LONG", url));
    else if (title.length < TITLE_MIN) issues.push(makeIssue("TITLE_TOO_SHORT", url));

    if (!metaDescription) issues.push(makeIssue("MISSING_META_DESCRIPTION", url));
    else if ((metaDescription as string).length > META_DESC_MAX) issues.push(makeIssue("META_DESCRIPTION_TOO_LONG", url));

    if (h1Count === 0) issues.push(makeIssue("MISSING_H1", url));
    else if (h1Count > 1) issues.push(makeIssue("MULTIPLE_H1", url));

    if (imagesMissingAlt > 0) issues.push(makeIssue("IMG_MISSING_ALT", url, `${imagesMissingAlt} gambar tanpa atribut alt.`));
    if (!canonical) issues.push(makeIssue("MISSING_CANONICAL", url));
    if (noindex) issues.push(makeIssue("NOINDEX", url));
    if (wordCount < THIN_WORDS) issues.push(makeIssue("THIN_CONTENT", url));
    if (!hasJsonLd) issues.push(makeIssue("NO_STRUCTURED_DATA", url));
    if (mixed) issues.push(makeIssue("MIXED_CONTENT", url));
  }
  if (loadMs != null && loadMs > SLOW_MS) issues.push(makeIssue("SLOW_RESPONSE", url));
  if ((input.redirectHops ?? 0) > 1) issues.push(makeIssue("REDIRECT_CHAIN", url));

  const page: CrawlPageData = {
    url,
    statusCode,
    title: title || null,
    metaDescription,
    h1Count,
    wordCount,
    canonical,
    indexable,
    imagesTotal,
    imagesMissingAlt,
    internalLinks: internal.size,
    externalLinks: external.size,
    loadMs: loadMs == null ? null : Math.round(loadMs),
    hasJsonLd,
    hasHreflang,
  };

  return { page, issues, internalUrls: [...internal], externalUrls: [...external] };
}
