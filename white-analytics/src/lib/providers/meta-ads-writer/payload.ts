/**
 * Pure builders for the Marketing API write payloads + the real/simulation
 * gate. No I/O, no `server-only`, so they are unit-tested directly
 * (tests/campaigns.test.ts).
 */
import { ADS_MANAGEMENT_SCOPE, type WriterAudience, type WriterDraft, type WriterGate, type WriterGateInput, type WriterGateReason, type WriterObjective } from "./types";

/**
 * Decide real vs simulation. ALL must hold for a real write: app credentials
 * present, client not in demo mode, ad account on a real connection, and that
 * connection was granted `ads_management`.
 */
export function writerGate(input: WriterGateInput): WriterGate {
  const reasons: WriterGateReason[] = [];
  if (!input.metaConfigured) reasons.push("META_NOT_CONFIGURED");
  if (input.demoClient) reasons.push("DEMO_CLIENT");
  if (!input.realConnection) reasons.push("NO_CONNECTION");
  if (!input.scopes.includes(ADS_MANAGEMENT_SCOPE)) reasons.push("MISSING_ADS_MANAGEMENT");
  return { mode: reasons.length === 0 ? "real" : "simulation", reasons };
}

/**
 * Currencies Meta bills without a minor unit (offset 1): amounts are sent as
 * whole units. Every other currency uses offset 100 (cents).
 * Source: Marketing API "Currencies" reference.
 */
const OFFSET_ONE = new Set(["CLP", "COP", "CRC", "HUF", "ISK", "IDR", "JPY", "KRW", "PYG", "TWD", "VND"]);

/** Major units → the integer Meta expects in `daily_budget`. */
export function toMetaBudget(amount: number, currency: string): number {
  const factor = OFFSET_ONE.has(currency.toUpperCase()) ? 1 : 100;
  return Math.round(amount * factor);
}

export function normalizeAdAccountId(id: string): string {
  const bare = id.trim().replace(/^act_/, "");
  return `act_${bare}`;
}

/**
 * Ad set optimisation per objective. Conversion optimisation (leads/sales)
 * needs a promoted_object (pixel / lead form) the app does not store yet, so
 * those objectives optimise for link clicks until it does.
 */
export const OPTIMIZATION_GOAL: Record<WriterObjective, string> = {
  OUTCOME_AWARENESS: "REACH",
  OUTCOME_TRAFFIC: "LINK_CLICKS",
  OUTCOME_ENGAGEMENT: "POST_ENGAGEMENT",
  OUTCOME_LEADS: "LINK_CLICKS",
  OUTCOME_SALES: "LINK_CLICKS",
};

/** Placement key → publisher platform + position (Meta targeting vocabulary). */
export const PLACEMENT_MAP: Record<string, { platform: "facebook" | "instagram" | "audience_network"; position: string }> = {
  facebook_feed: { platform: "facebook", position: "feed" },
  facebook_stories: { platform: "facebook", position: "story" },
  facebook_reels: { platform: "facebook", position: "facebook_reels" },
  facebook_marketplace: { platform: "facebook", position: "marketplace" },
  instagram_feed: { platform: "instagram", position: "stream" },
  instagram_stories: { platform: "instagram", position: "story" },
  instagram_reels: { platform: "instagram", position: "reels" },
  instagram_explore: { platform: "instagram", position: "explore" },
  audience_network: { platform: "audience_network", position: "classic" },
};

/** Country names users type → ISO codes. Anything else is treated as a city/region and resolved via search. */
const COUNTRY_CODES: Record<string, string> = {
  indonesia: "ID",
  malaysia: "MY",
  singapura: "SG",
  singapore: "SG",
  thailand: "TH",
  filipina: "PH",
  philippines: "PH",
  vietnam: "VN",
  australia: "AU",
  "amerika serikat": "US",
  "united states": "US",
};

export function splitLocations(locations: string[]): { countries: string[]; places: string[] } {
  const countries: string[] = [];
  const places: string[] = [];
  for (const raw of locations) {
    const name = raw.trim();
    if (!name) continue;
    const code = COUNTRY_CODES[name.toLowerCase()] ?? (/^[A-Z]{2}$/.test(name) ? name : null);
    if (code) {
      if (!countries.includes(code)) countries.push(code);
    } else places.push(name);
  }
  return { countries, places };
}

export type ResolvedTargeting = {
  /** Meta geo keys for cities/regions found by search. */
  cities?: { key: string }[];
  regions?: { key: string }[];
  /** Meta interest ids found by search. */
  interests?: { id: string; name: string }[];
};

/** Build the ad set `targeting` object. `resolved` holds ids looked up by the real adapter. */
export function buildTargeting(audience: WriterAudience, placements: string[], resolved: ResolvedTargeting = {}) {
  const { countries } = splitLocations(audience.locations);
  const geo: Record<string, unknown> = {};
  if (countries.length) geo.countries = countries;
  if (resolved.cities?.length) geo.cities = resolved.cities;
  if (resolved.regions?.length) geo.regions = resolved.regions;
  if (!geo.countries && !geo.cities && !geo.regions) geo.countries = ["ID"];

  const targeting: Record<string, unknown> = {
    geo_locations: geo,
    age_min: Math.max(18, Math.min(65, Math.round(audience.ageMin))),
    age_max: Math.max(18, Math.min(65, Math.round(audience.ageMax))),
    // Explicit opt-out of Advantage+ audience so our age/interest choices are respected.
    targeting_automation: { advantage_audience: 0 },
  };
  if (audience.genders === "MALE") targeting.genders = [1];
  if (audience.genders === "FEMALE") targeting.genders = [2];
  if (resolved.interests?.length) targeting.flexible_spec = [{ interests: resolved.interests }];

  const known = placements.map((p) => PLACEMENT_MAP[p]).filter((p): p is NonNullable<typeof p> => Boolean(p));
  if (known.length) {
    const platforms = Array.from(new Set(known.map((p) => p.platform)));
    targeting.publisher_platforms = platforms;
    const positions = (platform: string) => known.filter((p) => p.platform === platform).map((p) => p.position);
    if (platforms.includes("facebook")) targeting.facebook_positions = positions("facebook");
    if (platforms.includes("instagram")) targeting.instagram_positions = positions("instagram");
    if (platforms.includes("audience_network")) targeting.audience_network_positions = positions("audience_network");
  }
  return targeting;
}

/** POST /act_<id>/campaigns body. Always PAUSED: nothing runs until someone turns it on in Ads Manager. */
export function buildCampaignPayload(draft: WriterDraft): Record<string, unknown> {
  return {
    name: draft.name,
    objective: draft.objective,
    status: "PAUSED",
    special_ad_categories: [],
    buying_type: "AUCTION",
    // Budget lives on the ad set (no CBO); newer Graph versions require this flag to be explicit.
    is_adset_budget_sharing_enabled: false,
  };
}

/** POST /act_<id>/adsets body (targeting passed in so the caller can resolve ids first). */
export function buildAdSetPayload(draft: WriterDraft, campaignId: string, targeting: Record<string, unknown>): Record<string, unknown> {
  const body: Record<string, unknown> = {
    name: `${draft.name} · Ad set`,
    campaign_id: campaignId,
    daily_budget: toMetaBudget(draft.dailyBudget, draft.currency),
    billing_event: "IMPRESSIONS",
    optimization_goal: OPTIMIZATION_GOAL[draft.objective],
    bid_strategy: "LOWEST_COST_WITHOUT_CAP",
    targeting,
    start_time: draft.startDate.toISOString(),
    status: "PAUSED",
  };
  if (draft.endDate) body.end_time = draft.endDate.toISOString();
  return body;
}

/** POST /act_<id>/adcreatives body for one link ad. */
export function buildCreativePayload(
  draft: WriterDraft,
  index: number,
  ctx: { pageId: string; linkUrl: string },
): Record<string, unknown> {
  const c = draft.creatives[index]!;
  const linkData: Record<string, unknown> = {
    link: ctx.linkUrl,
    message: c.primaryText,
    name: c.headline,
    call_to_action: { type: c.cta || "LEARN_MORE", value: { link: ctx.linkUrl } },
  };
  if (c.mediaUrl) linkData.picture = c.mediaUrl;
  return {
    name: `${draft.name} · Materi ${index + 1}`,
    object_story_spec: { page_id: ctx.pageId, link_data: linkData },
  };
}

/** Graph POST bodies are form-encoded; object/array values are JSON strings. */
export function toFormBody(payload: Record<string, unknown>, accessToken: string): URLSearchParams {
  const form = new URLSearchParams();
  for (const [key, value] of Object.entries(payload)) {
    if (value === undefined || value === null) continue;
    form.set(key, typeof value === "object" ? JSON.stringify(value) : String(value));
  }
  form.set("access_token", accessToken);
  return form;
}
