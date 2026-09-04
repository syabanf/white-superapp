import "server-only";
import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";

/**
 * PageSpeed Insights v5 client (plain fetch, no SDK).
 * https://developers.google.com/speed/docs/insights/v5/get-started
 */

const ENDPOINT = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed";
const CATEGORIES = ["performance", "seo", "accessibility", "best-practices"] as const;
const DEFAULT_TIMEOUT_MS = 60_000;

export type PsiStrategy = "mobile" | "desktop";

export type PsiMetrics = {
  lcpMs: number | null;
  inpMs: number | null;
  cls: number | null;
  fcpMs: number | null;
  ttfbMs: number | null;
  tbtMs: number | null;
  speedIndexMs: number | null;
};

export type PsiResult = {
  strategy: PsiStrategy;
  finalUrl: string;
  scores: { performance: number; seo: number; accessibility: number; bestPractices: number };
  metrics: PsiMetrics;
  /** trimmed lighthouseResult (category scores + metric audits only) for SeoAudit.raw */
  raw: { categories: Record<string, number>; audits: Record<string, { numericValue?: number; displayValue?: string }> };
};

/** audit ids per metric — first match wins (INP has a legacy experimental id). */
const METRIC_AUDIT_IDS: Record<keyof PsiMetrics, string[]> = {
  lcpMs: ["largest-contentful-paint"],
  inpMs: ["interaction-to-next-paint", "experimental-interaction-to-next-paint"],
  cls: ["cumulative-layout-shift"],
  fcpMs: ["first-contentful-paint"],
  ttfbMs: ["server-response-time"],
  tbtMs: ["total-blocking-time"],
  speedIndexMs: ["speed-index"],
};

type RawPsiResponse = {
  lighthouseResult?: {
    finalDisplayedUrl?: string;
    finalUrl?: string;
    categories?: Record<string, { score?: number | null }>;
    audits?: Record<string, { numericValue?: number; displayValue?: string }>;
  };
  error?: { code?: number; message?: string };
};

export function isPagespeedConfigured(): boolean {
  return Boolean(process.env.PAGESPEED_API_KEY);
}

function codeForStatus(status: number): ProviderErrorCode {
  if (status === 429) return "RATE_LIMIT";
  if (status === 403) return "PERMISSION";
  if (status === 404) return "NOT_FOUND";
  return "UNKNOWN";
}

export async function runPagespeed(url: string, strategy: PsiStrategy, opts: { timeoutMs?: number; signal?: AbortSignal } = {}): Promise<PsiResult> {
  const params = new URLSearchParams({ url, strategy });
  for (const c of CATEGORIES) params.append("category", c);
  const key = process.env.PAGESPEED_API_KEY;
  if (key) params.set("key", key);

  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), opts.timeoutMs ?? DEFAULT_TIMEOUT_MS);
  opts.signal?.addEventListener("abort", () => controller.abort(), { once: true });

  let res: Response;
  try {
    res = await fetch(`${ENDPOINT}?${params.toString()}`, { signal: controller.signal });
  } catch (e) {
    const aborted = e instanceof Error && e.name === "AbortError";
    throw new ProviderError(
      "pagespeed",
      "NETWORK",
      aborted ? "PageSpeed Insights tidak merespons dalam 60 detik. Coba lagi." : "Gangguan jaringan saat memanggil PageSpeed Insights.",
      { cause: e },
    );
  } finally {
    clearTimeout(timer);
  }

  let json: RawPsiResponse;
  try {
    json = (await res.json()) as RawPsiResponse;
  } catch (e) {
    throw new ProviderError("pagespeed", "UNKNOWN", "Respons PageSpeed Insights tidak valid.", { cause: e });
  }

  if (!res.ok || !json.lighthouseResult) {
    const code = codeForStatus(res.status);
    const detail = json.error?.message ? ` (${json.error.message})` : "";
    const base =
      code === "RATE_LIMIT"
        ? "Kuota PageSpeed Insights tercapai. Coba lagi beberapa menit."
        : code === "PERMISSION"
          ? "Akses PageSpeed Insights ditolak. Periksa PAGESPEED_API_KEY."
          : "PageSpeed Insights gagal menganalisis URL ini.";
    throw new ProviderError("pagespeed", code, `${base}${detail}`);
  }

  const lh = json.lighthouseResult;
  const cat = (id: string) => Math.round(((lh.categories?.[id]?.score ?? 0) as number) * 100);
  const audits = lh.audits ?? {};
  const metric = (ids: string[]): number | null => {
    for (const id of ids) {
      const v = audits[id]?.numericValue;
      if (typeof v === "number" && Number.isFinite(v)) return v;
    }
    return null;
  };

  const metrics: PsiMetrics = {
    lcpMs: metric(METRIC_AUDIT_IDS.lcpMs),
    inpMs: metric(METRIC_AUDIT_IDS.inpMs),
    cls: metric(METRIC_AUDIT_IDS.cls),
    fcpMs: metric(METRIC_AUDIT_IDS.fcpMs),
    ttfbMs: metric(METRIC_AUDIT_IDS.ttfbMs),
    tbtMs: metric(METRIC_AUDIT_IDS.tbtMs),
    speedIndexMs: metric(METRIC_AUDIT_IDS.speedIndexMs),
  };

  const rawCategories: Record<string, number> = {};
  for (const c of CATEGORIES) rawCategories[c] = cat(c);
  const rawAudits: Record<string, { numericValue?: number; displayValue?: string }> = {};
  for (const ids of Object.values(METRIC_AUDIT_IDS)) {
    for (const id of ids) {
      const a = audits[id];
      if (a) rawAudits[id] = { numericValue: a.numericValue, displayValue: a.displayValue };
    }
  }

  return {
    strategy,
    finalUrl: lh.finalDisplayedUrl ?? lh.finalUrl ?? url,
    scores: { performance: cat("performance"), seo: cat("seo"), accessibility: cat("accessibility"), bestPractices: cat("best-practices") },
    metrics,
    raw: { categories: rawCategories, audits: rawAudits },
  };
}
