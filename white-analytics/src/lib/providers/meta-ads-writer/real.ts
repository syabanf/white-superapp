/**
 * Real Meta Ads writer (Marketing API, Graph version from meta-graph/version.ts),
 * plain fetch, no SDK. NOT exercised yet: `pickAdsWriter()` only returns it when
 * the connection was granted `ads_management`, which needs Meta App Review.
 *
 * Sequence (everything created PAUSED, so nothing spends until a human turns
 * it on in Ads Manager):
 *   1. resolve free-text cities/regions and interests to Meta ids (search API)
 *   2. POST /act_<id>/campaigns     objective, special_ad_categories: []
 *   3. POST /act_<id>/adsets        daily_budget in Meta units, targeting, schedule
 *   4. per creative: POST /act_<id>/adcreatives (link ad on the client's Page)
 *                    POST /act_<id>/ads (creative → ad set)
 * If step 3 or 4 fails, the campaign from step 2 is deleted (best effort) so
 * a retry does not leave half-built campaigns behind.
 *
 * Errors are normalised to ProviderError (caught at the action boundary).
 */
import { ProviderError, type ProviderErrorCode } from "@/lib/action-result";
import { META_GRAPH_BASE } from "@/lib/providers/meta-graph/version";
import {
  buildAdSetPayload,
  buildCampaignPayload,
  buildCreativePayload,
  buildTargeting,
  normalizeAdAccountId,
  splitLocations,
  toFormBody,
  type ResolvedTargeting,
} from "./payload";
import type { CreateCampaignResult, MetaAdsWriter, WriterContext, WriterDraft } from "./types";

const PROVIDER = "meta-ads-writer";

function mapErrorCode(code: number | undefined): ProviderErrorCode {
  if (code === 190) return "TOKEN_EXPIRED";
  if (code === 4 || code === 17 || code === 32 || code === 613 || code === 80004) return "RATE_LIMIT";
  if (code === 10 || code === 200 || code === 272 || code === 294) return "PERMISSION";
  if (code === 803 || code === 100) return "NOT_FOUND";
  return "UNKNOWN";
}

type GraphError = { error?: { message?: string; code?: number; error_user_msg?: string } };

async function graph<T>(method: "GET" | "POST" | "DELETE", path: string, opts: { token: string; body?: Record<string, unknown>; query?: Record<string, string> }): Promise<T> {
  const url = new URL(`${META_GRAPH_BASE}/${path}`);
  let init: RequestInit = { method, cache: "no-store" };
  if (method === "POST") {
    init = { ...init, body: toFormBody(opts.body ?? {}, opts.token), headers: { "Content-Type": "application/x-www-form-urlencoded" } };
  } else {
    for (const [k, v] of Object.entries(opts.query ?? {})) url.searchParams.set(k, v);
    url.searchParams.set("access_token", opts.token);
  }
  let res: Response;
  try {
    res = await fetch(url, init);
  } catch (cause) {
    throw new ProviderError(PROVIDER, "NETWORK", "Gangguan jaringan saat menghubungi Meta.", { cause });
  }
  let json: (T & GraphError) | null = null;
  try {
    json = (await res.json()) as T & GraphError;
  } catch {
    // body bukan JSON: jatuh ke pemeriksaan status
  }
  if (!res.ok || json?.error) {
    const err = json?.error;
    throw new ProviderError(PROVIDER, mapErrorCode(err?.code), err?.error_user_msg ?? err?.message ?? `Meta API ${res.status}`);
  }
  return (json ?? {}) as T;
}

/** Look up Meta ids for free-text places and interests. Unmatched names are dropped (not guessed). */
async function resolveTargeting(draft: WriterDraft, token: string): Promise<ResolvedTargeting> {
  const { places } = splitLocations(draft.audience.locations);
  const cities: { key: string }[] = [];
  const regions: { key: string }[] = [];
  for (const q of places.slice(0, 10)) {
    const res = await graph<{ data?: { key: string; type: string }[] }>("GET", "search", {
      token,
      query: { type: "adgeolocation", q, location_types: JSON.stringify(["city", "region"]), limit: "1" },
    });
    const hit = res.data?.[0];
    if (hit?.type === "city") cities.push({ key: hit.key });
    else if (hit?.type === "region") regions.push({ key: hit.key });
  }
  const interests: { id: string; name: string }[] = [];
  for (const q of draft.audience.interests.slice(0, 10)) {
    const res = await graph<{ data?: { id: string; name: string }[] }>("GET", "search", { token, query: { type: "adinterest", q, limit: "1" } });
    const hit = res.data?.[0];
    if (hit && !interests.some((i) => i.id === hit.id)) interests.push({ id: hit.id, name: hit.name });
  }
  return { cities, regions, interests };
}

export async function realCreateCampaign(draft: WriterDraft, ctx: WriterContext): Promise<CreateCampaignResult> {
  const token = ctx.accessToken;
  const act = normalizeAdAccountId(ctx.adAccountExternalId);
  // Fail fast before creating anything: a link ad needs a Page and a destination.
  if (!ctx.pageId) throw new ProviderError(PROVIDER, "NOT_FOUND", "Halaman Facebook klien belum terhubung; iklan butuh Page sebagai identitas.", { requiresReconnect: false });
  if (!ctx.linkUrl) throw new ProviderError(PROVIDER, "NOT_FOUND", "Website klien belum diisi; iklan butuh URL tujuan.", { requiresReconnect: false });
  if (!draft.creatives.length) throw new ProviderError(PROVIDER, "NOT_FOUND", "Draf belum punya materi iklan.", { requiresReconnect: false });

  const resolved = await resolveTargeting(draft, token);
  const campaign = await graph<{ id: string }>("POST", `${act}/campaigns`, { token, body: buildCampaignPayload(draft) });

  try {
    const targeting = buildTargeting(draft.audience, draft.placements, resolved);
    const adSet = await graph<{ id: string }>("POST", `${act}/adsets`, { token, body: buildAdSetPayload(draft, campaign.id, targeting) });
    const adIds: string[] = [];
    for (let i = 0; i < draft.creatives.length; i++) {
      const creative = await graph<{ id: string }>("POST", `${act}/adcreatives`, {
        token,
        body: buildCreativePayload(draft, i, { pageId: ctx.pageId, linkUrl: ctx.linkUrl }),
      });
      const ad = await graph<{ id: string }>("POST", `${act}/ads`, {
        token,
        body: { name: `${draft.name} · Iklan ${i + 1}`, adset_id: adSet.id, creative: { creative_id: creative.id }, status: "PAUSED" },
      });
      adIds.push(ad.id);
    }
    return { externalId: campaign.id, simulated: false, adSetId: adSet.id, adIds };
  } catch (err) {
    await graph("DELETE", campaign.id, { token }).catch(() => undefined);
    throw err;
  }
}

export const realAdsWriter: MetaAdsWriter = {
  id: "meta",
  createCampaign: realCreateCampaign,
};
