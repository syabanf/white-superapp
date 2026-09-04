/**
 * Adapter Meta Marketing API (Graph v21.0) — fetch polos, tanpa SDK.
 * Error dinormalkan menjadi ProviderError (ditangkap di boundary action).
 */
import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";
import type { InsightsQuery, MetaInsightRow, MetaMarketingAdapter } from "./types";

const GRAPH = "https://graph.facebook.com/v21.0";
const PROVIDER = "meta-marketing";

/** Kode error Graph API → ProviderErrorCode. */
function mapErrorCode(code: number | undefined): ProviderErrorCode {
  if (code === 190) return "TOKEN_EXPIRED";
  if (code === 4 || code === 17 || code === 32 || code === 613) return "RATE_LIMIT";
  if (code === 10 || code === 200 || code === 272) return "PERMISSION";
  if (code === 803 || code === 100) return "NOT_FOUND";
  return "UNKNOWN";
}

type GraphError = { error?: { message?: string; code?: number; error_subcode?: number } };
type Paged<T> = { data?: T[]; paging?: { next?: string } };

async function graphFetch<T>(url: string): Promise<Paged<T> & GraphError> {
  let res: Response;
  try {
    res = await fetch(url, { cache: "no-store" });
  } catch (cause) {
    throw new ProviderError(PROVIDER, "NETWORK", "Gangguan jaringan saat menghubungi Meta.", { cause });
  }
  let json: (Paged<T> & GraphError) | null = null;
  try {
    json = (await res.json()) as Paged<T> & GraphError;
  } catch {
    // body bukan JSON — jatuh ke pemeriksaan status di bawah
  }
  if (!res.ok || json?.error) {
    const err = json?.error;
    throw new ProviderError(PROVIDER, mapErrorCode(err?.code), err?.message ?? `Meta API ${res.status}`);
  }
  return json ?? {};
}

/** Ikuti paging.next hingga habis (dibatasi maxPages agar aman). */
async function fetchAll<T>(firstUrl: string, maxPages = 25): Promise<T[]> {
  const out: T[] = [];
  let url: string | undefined = firstUrl;
  for (let page = 0; url && page < maxPages; page++) {
    const body: Paged<T> & GraphError = await graphFetch<T>(url);
    if (body.data) out.push(...body.data);
    url = body.paging?.next;
  }
  return out;
}

function num(v: unknown): number {
  const n = typeof v === "string" ? Number(v) : typeof v === "number" ? v : 0;
  return Number.isFinite(n) ? n : 0;
}

type RawAction = { action_type?: string; value?: string };

/** Prioritas pemilihan "hasil" dari daftar actions Meta. */
const RESULT_PRIORITY: Array<{ type: string; match: (t: string) => boolean }> = [
  { type: "purchase", match: (t) => t.includes("purchase") },
  { type: "lead", match: (t) => t === "lead" || t.includes(".lead") || t.includes("lead_grouped") },
  { type: "messaging_conversation_started", match: (t) => t.includes("messaging_conversation_started") },
  { type: "link_click", match: (t) => t === "link_click" },
];

function pickResult(actions: RawAction[] | undefined): { results: number; resultType: string | null } {
  if (!actions?.length) return { results: 0, resultType: null };
  for (const p of RESULT_PRIORITY) {
    const hit = actions.filter((a) => a.action_type != null && p.match(a.action_type));
    if (hit.length > 0) return { results: hit.reduce((s, a) => s + num(a.value), 0), resultType: p.type };
  }
  return { results: 0, resultType: null };
}

function pickPurchaseValue(actionValues: RawAction[] | undefined): number | null {
  if (!actionValues?.length) return null;
  const hits = actionValues.filter((a) => a.action_type?.includes("purchase"));
  if (hits.length === 0) return null;
  return hits.reduce((s, a) => s + num(a.value), 0);
}

type RawInsight = {
  campaign_id?: string;
  adset_id?: string;
  ad_id?: string;
  spend?: string;
  impressions?: string;
  reach?: string;
  clicks?: string;
  inline_link_clicks?: string;
  frequency?: string;
  actions?: RawAction[];
  action_values?: RawAction[];
  date_start?: string;
  age?: string;
  gender?: string;
};

export function realMetaMarketing(accessToken: string): MetaMarketingAdapter {
  const qs = (params: Record<string, string>) => new URLSearchParams({ ...params, access_token: accessToken, limit: "200" }).toString();

  return {
    async listCampaigns(actId) {
      type Raw = { id: string; name: string; objective?: string; status?: string; daily_budget?: string; start_time?: string };
      const rows = await fetchAll<Raw>(`${GRAPH}/${actId}/campaigns?${qs({ fields: "id,name,objective,status,daily_budget,start_time" })}`);
      return rows.map((c) => ({
        id: c.id,
        name: c.name,
        objective: c.objective ?? "OUTCOME_TRAFFIC",
        status: c.status ?? "ACTIVE",
        // daily_budget dalam satuan minor (sen); IDR tanpa desimal → /100 aman utk mayoritas mata uang
        dailyBudget: c.daily_budget != null ? num(c.daily_budget) / 100 : null,
        startTime: c.start_time ?? null,
      }));
    },

    async listAdSets(actId) {
      type Raw = { id: string; campaign_id: string; name: string; status?: string; daily_budget?: string };
      const rows = await fetchAll<Raw>(`${GRAPH}/${actId}/adsets?${qs({ fields: "id,name,campaign_id,status,daily_budget" })}`);
      return rows.map((s) => ({
        id: s.id,
        campaignId: s.campaign_id,
        name: s.name,
        status: s.status ?? "ACTIVE",
        dailyBudget: s.daily_budget != null ? num(s.daily_budget) / 100 : null,
      }));
    },

    async listAds(actId) {
      type Raw = { id: string; name: string; status?: string; adset_id: string; campaign_id: string; creative?: { thumbnail_url?: string } };
      const rows = await fetchAll<Raw>(
        `${GRAPH}/${actId}/ads?${qs({ fields: "id,name,status,adset_id,campaign_id,creative{thumbnail_url}" })}`,
      );
      return rows.map((a) => ({
        id: a.id,
        adSetId: a.adset_id,
        campaignId: a.campaign_id,
        name: a.name,
        status: a.status ?? "ACTIVE",
        thumbnailUrl: a.creative?.thumbnail_url ?? null,
      }));
    },

    async getInsights(actId, query: InsightsQuery): Promise<MetaInsightRow[]> {
      const params: Record<string, string> = {
        level: query.level ?? "ad",
        time_increment: String(query.timeIncrement ?? 1),
        time_range: JSON.stringify({ since: query.since, until: query.until }),
        fields: "campaign_id,adset_id,ad_id,spend,impressions,reach,clicks,inline_link_clicks,actions,action_values,frequency,date_start",
      };
      if (query.breakdowns?.length) params.breakdowns = query.breakdowns.join(",");
      const rows = await fetchAll<RawInsight>(`${GRAPH}/${actId}/insights?${qs(params)}`, 100);
      return rows.map((r) => {
        const { results, resultType } = pickResult(r.actions);
        return {
          campaignId: r.campaign_id ?? "",
          adSetId: r.adset_id ?? "",
          adId: r.ad_id ?? "",
          date: r.date_start ?? query.since,
          spend: num(r.spend),
          impressions: num(r.impressions),
          reach: num(r.reach),
          clicks: num(r.clicks),
          linkClicks: num(r.inline_link_clicks),
          frequency: num(r.frequency),
          results,
          resultType,
          purchaseValue: pickPurchaseValue(r.action_values),
          ...(r.age != null ? { age: r.age } : {}),
          ...(r.gender != null ? { gender: r.gender } : {}),
        };
      });
    },
  };
}
