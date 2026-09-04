import "server-only";
import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";
import type { Ga4ReportRequest, Ga4ReportRow, GoogleSearchProvider, GscQueryRequest, GscRow } from "./types";

const GSC_BASE = "https://www.googleapis.com/webmasters/v3";
const GA4_BASE = "https://analyticsdata.googleapis.com/v1beta";
const TIMEOUT_MS = 30_000;

function codeForStatus(status: number): ProviderErrorCode {
  if (status === 401) return "TOKEN_EXPIRED";
  if (status === 403) return "PERMISSION";
  if (status === 429) return "RATE_LIMIT";
  if (status === 404) return "NOT_FOUND";
  return "UNKNOWN";
}

function messageForCode(code: ProviderErrorCode, api: string): string {
  switch (code) {
    case "TOKEN_EXPIRED":
      return `Token Google kedaluwarsa saat memanggil ${api}. Hubungkan ulang akun Google.`;
    case "PERMISSION":
      return `Akun Google tidak punya izin untuk ${api}. Periksa akses properti.`;
    case "RATE_LIMIT":
      return `Batas kuota ${api} tercapai. Coba lagi beberapa menit.`;
    case "NOT_FOUND":
      return `Properti tidak ditemukan di ${api}.`;
    default:
      return `Terjadi kesalahan saat memanggil ${api}.`;
  }
}

async function googleFetch<T>(url: string, accessToken: string, body: unknown, api: string): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), TIMEOUT_MS);
  let res: Response;
  try {
    res = await fetch(url, {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: controller.signal,
    });
  } catch (e) {
    throw new ProviderError("google", "NETWORK", `Gangguan jaringan saat memanggil ${api}. Periksa koneksi dan coba lagi.`, { cause: e });
  } finally {
    clearTimeout(timer);
  }
  if (!res.ok) {
    const code = codeForStatus(res.status);
    let detail = "";
    try {
      const j = (await res.json()) as { error?: { message?: string } };
      detail = j.error?.message ?? "";
    } catch {
      // ignore body parse errors
    }
    throw new ProviderError("google", code, detail ? `${messageForCode(code, api)} (${detail})` : messageForCode(code, api));
  }
  return (await res.json()) as T;
}

type RawGscResponse = { rows?: Array<{ keys?: string[]; clicks?: number; impressions?: number; ctr?: number; position?: number }> };
type RawGa4Response = { rows?: Array<{ dimensionValues?: Array<{ value?: string }>; metricValues?: Array<{ value?: string }> }> };

export const realGoogleSearchProvider: GoogleSearchProvider = {
  /** GSC Search Analytics — https://developers.google.com/webmaster-tools/v1/searchanalytics/query */
  async querySearchAnalytics(accessToken: string, req: GscQueryRequest): Promise<GscRow[]> {
    const url = `${GSC_BASE}/sites/${encodeURIComponent(req.siteUrl)}/searchAnalytics/query`;
    const json = await googleFetch<RawGscResponse>(
      url,
      accessToken,
      {
        startDate: req.startDate,
        endDate: req.endDate,
        dimensions: req.dimensions,
        rowLimit: req.rowLimit ?? 5000,
        startRow: req.startRow ?? 0,
      },
      "Google Search Console",
    );
    return (json.rows ?? []).map((r) => ({
      keys: r.keys ?? [],
      clicks: r.clicks ?? 0,
      impressions: r.impressions ?? 0,
      // Google returns ctr as a 0–1 fraction; the app convention is percentage 0–100.
      ctr: (r.ctr ?? 0) * 100,
      position: r.position ?? 0,
    }));
  },

  /** GA4 Data API runReport — https://developers.google.com/analytics/devguides/reporting/data/v1 */
  async runGa4Report(accessToken: string, req: Ga4ReportRequest): Promise<Ga4ReportRow[]> {
    const property = req.propertyId.startsWith("properties/") ? req.propertyId : `properties/${req.propertyId}`;
    const url = `${GA4_BASE}/${property}:runReport`;
    const json = await googleFetch<RawGa4Response>(
      url,
      accessToken,
      {
        dateRanges: [{ startDate: req.startDate, endDate: req.endDate }],
        metrics: req.metrics.map((name) => ({ name })),
        dimensions: req.dimensions.map((name) => ({ name })),
        limit: req.limit ?? 10000,
      },
      "Google Analytics 4",
    );
    return (json.rows ?? []).map((r) => ({
      dimensions: (r.dimensionValues ?? []).map((d) => d.value ?? ""),
      metrics: (r.metricValues ?? []).map((m) => Number(m.value ?? 0)),
    }));
  },
};
