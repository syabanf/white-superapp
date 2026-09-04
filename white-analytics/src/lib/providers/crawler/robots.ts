/** robots.txt + sitemap.xml parsing — pure functions (unit-testable). */
import * as cheerio from "cheerio";

export type RobotsRules = {
  disallow: string[];
  allow: string[];
  sitemaps: string[];
};

export const ALLOW_ALL: RobotsRules = { disallow: [], allow: [], sitemaps: [] };

/**
 * Parse robots.txt for a given user-agent token (case-insensitive substring match).
 * Groups addressed to the specific UA win over `*` groups (per RFC 9309).
 */
export function parseRobots(txt: string, uaToken: string): RobotsRules {
  type Group = { agents: string[]; disallow: string[]; allow: string[] };
  const groups: Group[] = [];
  const sitemaps: string[] = [];
  let current: Group | null = null;
  let lastWasAgent = false;

  for (const raw of txt.split(/\r?\n/)) {
    const line = raw.replace(/#.*$/, "").trim();
    if (!line) continue;
    const idx = line.indexOf(":");
    if (idx < 0) continue;
    const field = line.slice(0, idx).trim().toLowerCase();
    const value = line.slice(idx + 1).trim();
    if (field === "sitemap") {
      if (value) sitemaps.push(value);
      continue;
    }
    if (field === "user-agent") {
      if (!current || !lastWasAgent) {
        current = { agents: [], disallow: [], allow: [] };
        groups.push(current);
      }
      current.agents.push(value.toLowerCase());
      lastWasAgent = true;
      continue;
    }
    lastWasAgent = false;
    if (!current) continue;
    if (field === "disallow" && value) current.disallow.push(value);
    if (field === "allow" && value) current.allow.push(value);
  }

  const token = uaToken.toLowerCase();
  const specific = groups.filter((g) => g.agents.some((a) => a !== "*" && (token.includes(a) || a.includes(token))));
  const chosen = specific.length > 0 ? specific : groups.filter((g) => g.agents.includes("*"));
  return {
    disallow: chosen.flatMap((g) => g.disallow),
    allow: chosen.flatMap((g) => g.allow),
    sitemaps,
  };
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/** Longest-match length of a robots pattern against a path, or -1. Supports `*` and trailing `$`. */
function matchLength(pattern: string, path: string): number {
  let p = pattern;
  let anchored = false;
  if (p.endsWith("$")) {
    anchored = true;
    p = p.slice(0, -1);
  }
  const re = new RegExp(`^${p.split("*").map(escapeRegex).join(".*")}${anchored ? "$" : ""}`);
  return re.test(path) ? p.length : -1;
}

/** Whether a path may be fetched under the parsed rules (most-specific rule wins; allow beats tie). */
export function isAllowed(rules: RobotsRules, path: string): boolean {
  let best = -1;
  for (const d of rules.disallow) best = Math.max(best, matchLength(d, path));
  if (best < 0) return true;
  let bestAllow = -1;
  for (const a of rules.allow) bestAllow = Math.max(bestAllow, matchLength(a, path));
  return bestAllow >= best;
}

/** Extract page URLs and child sitemap URLs from a sitemap.xml / sitemap-index document. */
export function parseSitemap(xml: string): { urls: string[]; sitemaps: string[] } {
  const urls: string[] = [];
  const sitemaps: string[] = [];
  try {
    const $ = cheerio.load(xml, { xmlMode: true });
    $("urlset > url > loc").each((_, el) => {
      const v = $(el).text().trim();
      if (v) urls.push(v);
    });
    $("sitemapindex > sitemap > loc").each((_, el) => {
      const v = $(el).text().trim();
      if (v) sitemaps.push(v);
    });
  } catch {
    // malformed XML → no seeds
  }
  return { urls, sitemaps };
}
