import "server-only";
import { ProviderError } from "@/lib/action-result";
import { analyzeHtml } from "./analyze";
import { ALLOW_ALL, isAllowed, parseRobots, parseSitemap, type RobotsRules } from "./robots";
import { CRAWLER_UA, CRAWLER_UA_TOKEN, ISSUE_DEFS, type CrawlIssue, type CrawlPageData, type CrawlResult } from "./types";

export type { CrawlIssue, CrawlPageData, CrawlResult } from "./types";
export { CRAWLER_UA } from "./types";
export { analyzeHtml } from "./analyze";

export type CrawlOptions = {
  /** default 30, hard cap 60 */
  maxPages?: number;
  /** wall-clock budget, default 45s */
  timeBudgetMs?: number;
  concurrency?: number;
  requestTimeoutMs?: number;
};

const HARD_CAP = 60;
const BINARY_RE = /\.(png|jpe?g|gif|webp|avif|svg|ico|css|js|mjs|json|pdf|zip|rar|7z|gz|tar|mp3|mp4|webm|mov|avi|wmv|woff2?|ttf|otf|eot|txt|xml|rss|atom|exe|dmg|apk)(\?.*)?$/i;
const MAX_REDIRECTS = 5;

function normalizeStart(input: string): string {
  let raw = input.trim();
  if (raw.startsWith("sc-domain:")) raw = `https://${raw.slice("sc-domain:".length)}/`;
  if (!/^https?:\/\//i.test(raw)) raw = `https://${raw}`;
  const u = new URL(raw);
  u.hash = "";
  return u.toString();
}

function normalizeUrl(raw: string): string | null {
  try {
    const u = new URL(raw);
    u.hash = "";
    return u.toString();
  } catch {
    return null;
  }
}

type FetchOutcome = {
  status: number;
  finalUrl: string;
  hops: number;
  loadMs: number;
  html: string | null;
  xRobotsTag: string | null;
  error?: string;
};

async function fetchPage(url: string, opts: { timeoutMs: number; method?: "GET" | "HEAD" }): Promise<FetchOutcome> {
  const started = Date.now();
  let current = url;
  let hops = 0;
  const method = opts.method ?? "GET";
  for (;;) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), opts.timeoutMs);
    let res: Response;
    try {
      res = await fetch(current, {
        method,
        redirect: "manual",
        signal: controller.signal,
        headers: { "user-agent": CRAWLER_UA, accept: "text/html,application/xhtml+xml,*/*;q=0.8" },
      });
    } catch (e) {
      clearTimeout(timer);
      return { status: 0, finalUrl: current, hops, loadMs: Date.now() - started, html: null, xRobotsTag: null, error: e instanceof Error ? e.message : String(e) };
    }
    clearTimeout(timer);

    if (res.status >= 300 && res.status < 400) {
      const loc = res.headers.get("location");
      res.body?.cancel().catch(() => undefined);
      if (!loc || hops >= MAX_REDIRECTS) {
        return { status: res.status, finalUrl: current, hops, loadMs: Date.now() - started, html: null, xRobotsTag: res.headers.get("x-robots-tag") };
      }
      try {
        current = new URL(loc, current).toString();
      } catch {
        return { status: res.status, finalUrl: current, hops, loadMs: Date.now() - started, html: null, xRobotsTag: null };
      }
      hops += 1;
      continue;
    }

    const contentType = res.headers.get("content-type") ?? "";
    const isHtml = contentType.includes("text/html") || contentType.includes("application/xhtml");
    let html: string | null = null;
    if (method === "GET" && isHtml) {
      try {
        html = await res.text();
      } catch {
        html = null;
      }
    } else {
      res.body?.cancel().catch(() => undefined);
    }
    return { status: res.status, finalUrl: current, hops, loadMs: Date.now() - started, html, xRobotsTag: res.headers.get("x-robots-tag") };
  }
}

async function fetchText(url: string, timeoutMs: number): Promise<string | null> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetch(url, { signal: controller.signal, headers: { "user-agent": CRAWLER_UA } });
    if (!res.ok) return null;
    return await res.text();
  } catch {
    return null;
  } finally {
    clearTimeout(timer);
  }
}

async function collectSeeds(origin: string, rules: RobotsRules, cap: number): Promise<string[]> {
  const candidates = [...new Set([...rules.sitemaps, `${origin}/sitemap.xml`])].slice(0, 4);
  const urls: string[] = [];
  let docsFetched = 0;
  const queue = [...candidates];
  while (queue.length > 0 && docsFetched < 4 && urls.length < cap) {
    const smUrl = queue.shift()!;
    docsFetched += 1;
    const xml = await fetchText(smUrl, 5000);
    if (!xml) continue;
    const parsed = parseSitemap(xml);
    for (const child of parsed.sitemaps.slice(0, 3)) queue.push(child);
    for (const u of parsed.urls) {
      const n = normalizeUrl(u);
      if (n && n.startsWith(origin)) urls.push(n);
      if (urls.length >= cap) break;
    }
  }
  return urls;
}

/**
 * BFS crawl of a site: robots-aware, sitemap-seeded, same-origin only.
 * Stops at maxPages (default 30, cap 60) or the wall-clock budget (default 45s).
 */
export async function crawlSite(startUrl: string, options: CrawlOptions = {}): Promise<CrawlResult> {
  const started = Date.now();
  const maxPages = Math.min(options.maxPages ?? 30, HARD_CAP);
  const budget = options.timeBudgetMs ?? 45_000;
  const concurrency = options.concurrency ?? 4;
  const requestTimeout = options.requestTimeoutMs ?? 10_000;
  const timeLeft = () => budget - (Date.now() - started);

  let start: string;
  try {
    start = normalizeStart(startUrl);
  } catch {
    throw new ProviderError("crawler", "UNKNOWN", `URL situs tidak valid: ${startUrl}`);
  }
  const origin = new URL(start).origin;

  // robots.txt (best effort)
  const robotsTxt = await fetchText(`${origin}/robots.txt`, 5000);
  const rules = robotsTxt ? parseRobots(robotsTxt, CRAWLER_UA_TOKEN) : ALLOW_ALL;

  // sitemap seeds (best effort)
  const seeds = timeLeft() > 5000 ? await collectSeeds(origin, rules, maxPages * 2) : [];

  const queue: string[] = [];
  const queued = new Set<string>();
  const enqueue = (u: string) => {
    if (queued.has(u) || queue.length > maxPages * 6) return;
    queued.add(u);
    queue.push(u);
  };
  enqueue(start);
  for (const sd of seeds) enqueue(sd);

  const pages: CrawlPageData[] = [];
  const issues: CrawlIssue[] = [];
  const issueKeys = new Set<string>();
  const pushIssue = (i: CrawlIssue) => {
    const k = `${i.code}|${i.url}|${i.message}`;
    if (issueKeys.has(k)) return;
    issueKeys.add(k);
    issues.push(i);
  };
  const statusByUrl = new Map<string, number>();
  /** internal link target → first page that references it */
  const linkTargets = new Map<string, string>();
  let active = 0;

  const processUrl = async (url: string) => {
    const u = new URL(url);
    if (u.origin !== origin) return;
    if (BINARY_RE.test(u.pathname)) return;
    if (!isAllowed(rules, `${u.pathname}${u.search}`)) return;

    const out = await fetchPage(url, { timeoutMs: requestTimeout });
    if (out.status === 0) return; // network error on this URL — skip silently
    statusByUrl.set(url, out.status);
    statusByUrl.set(out.finalUrl, out.status);

    const finalU = normalizeUrl(out.finalUrl);
    if (!finalU || !finalU.startsWith(origin)) return; // redirected off-site

    if (pages.length >= maxPages) return;

    if (out.html == null) {
      // non-HTML or bodyless response: record status-only page for error codes
      if (out.status >= 400) {
        pages.push({
          url: finalU,
          statusCode: out.status,
          title: null,
          metaDescription: null,
          h1Count: 0,
          wordCount: 0,
          canonical: null,
          indexable: false,
          imagesTotal: 0,
          imagesMissingAlt: 0,
          internalLinks: 0,
          externalLinks: 0,
          loadMs: Math.round(out.loadMs),
          hasJsonLd: false,
          hasHreflang: false,
        });
      }
      return;
    }

    const analyzed = analyzeHtml({
      url: finalU,
      html: out.html,
      statusCode: out.status,
      loadMs: out.loadMs,
      xRobotsTag: out.xRobotsTag,
      redirectHops: out.hops,
    });
    pages.push(analyzed.page);
    for (const i of analyzed.issues) pushIssue(i);
    for (const link of analyzed.internalUrls) {
      if (!linkTargets.has(link)) linkTargets.set(link, finalU);
      if (!BINARY_RE.test(link)) enqueue(link);
    }
  };

  const worker = async () => {
    for (;;) {
      if (timeLeft() < 1500 || pages.length >= maxPages) return;
      const next = queue.shift();
      if (next == null) {
        if (active === 0) return;
        await new Promise((r) => setTimeout(r, 60));
        continue;
      }
      active += 1;
      try {
        await processUrl(next);
      } catch {
        // per-URL failures never abort the crawl
      } finally {
        active -= 1;
      }
    }
  };

  await Promise.all(Array.from({ length: concurrency }, () => worker()));

  if (pages.length === 0) {
    throw new ProviderError("crawler", "NETWORK", `Tidak ada halaman yang bisa di-crawl dari ${origin}. Periksa URL dan koneksi.`);
  }

  // ── broken internal links ─────────────────────────────────
  // 1) targets we already crawled and saw 4xx/5xx; 2) HEAD-check a few uncrawled targets.
  const brokenFor = (target: string, referrer: string, status: number) =>
    pushIssue({
      code: "BROKEN_INTERNAL_LINK",
      severity: ISSUE_DEFS.BROKEN_INTERNAL_LINK.severity,
      message: `Tautan internal mengarah ke halaman ${status}: ${target}`,
      url: referrer,
    });

  const unchecked: Array<[string, string]> = [];
  for (const [target, referrer] of linkTargets) {
    const known = statusByUrl.get(target);
    if (known != null) {
      if (known >= 400) brokenFor(target, referrer, known);
    } else if (!BINARY_RE.test(target)) {
      unchecked.push([target, referrer]);
    }
  }
  const toCheck = unchecked.slice(0, 15);
  let idx = 0;
  const checkWorker = async () => {
    while (idx < toCheck.length && timeLeft() > 1500) {
      const [target, referrer] = toCheck[idx++]!;
      let out = await fetchPage(target, { timeoutMs: 5000, method: "HEAD" });
      if (out.status === 405 || out.status === 501) out = await fetchPage(target, { timeoutMs: 5000, method: "GET" });
      if (out.status >= 400) brokenFor(target, referrer, out.status);
    }
  };
  await Promise.all(Array.from({ length: Math.min(4, toCheck.length) }, () => checkWorker()));

  return { pages, issues, pagesCrawled: pages.length, durationMs: Date.now() - started };
}
