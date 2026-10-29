/**
 * Meta Ads writer — the write side of the Marketing API (create campaign →
 * ad set → creatives → ads). Shared contract for the real Graph adapter and
 * the simulation mock. Plain, serializable shapes only: the caller (features/
 * campaigns) maps a `CampaignDraft` row into `WriterDraft` and decrypts the
 * token; the adapter never touches the database.
 *
 * Real writes need the `ads_management` permission (Meta App Review). Until a
 * connection has it, `pickAdsWriter()` returns the mock and every submission is
 * recorded as `simulated: true` — nothing is created in Meta, nothing is spent.
 */

export type WriterObjective = "OUTCOME_AWARENESS" | "OUTCOME_TRAFFIC" | "OUTCOME_ENGAGEMENT" | "OUTCOME_LEADS" | "OUTCOME_SALES";

export type WriterAudience = {
  /** Free text: country names ("Indonesia") or city/region names ("Jakarta"). */
  locations: string[];
  ageMin: number;
  ageMax: number;
  genders: "ALL" | "MALE" | "FEMALE";
  /** Free-text interest names, resolved to Meta interest ids by the real adapter. */
  interests: string[];
};

export type WriterCreative = {
  headline: string;
  primaryText: string;
  /** Meta call_to_action type, e.g. LEARN_MORE, SHOP_NOW. */
  cta: string;
  /** Public image URL (Meta fetches it). Optional: link ads fall back to the page's link preview. */
  mediaUrl?: string | null;
};

export type WriterDraft = {
  /** Our draft id, used for idempotent naming and the simulated external id. */
  id: string;
  name: string;
  objective: WriterObjective;
  /** Daily budget in MAJOR units of `currency` (e.g. 50000 = Rp50.000). */
  dailyBudget: number;
  currency: string;
  startDate: Date;
  endDate: Date | null;
  audience: WriterAudience;
  /** Placement keys from features/campaigns/lib.ts (empty = Advantage+ placements). */
  placements: string[];
  creatives: WriterCreative[];
};

export type WriterContext = {
  /** Decrypted user/system-user token carrying `ads_management`. Ignored by the mock. */
  accessToken: string;
  /** Ad account id, with or without the `act_` prefix. */
  adAccountExternalId: string;
  /** Facebook Page that "speaks" in the ad (object_story_spec.page_id). Required by the real adapter. */
  pageId?: string | null;
  /** Destination URL for link ads (client website). Required by the real adapter. */
  linkUrl?: string | null;
};

export type CreateCampaignResult = {
  /** Meta campaign id, or `sim_<…>` for a simulation. */
  externalId: string;
  simulated: boolean;
  adSetId: string | null;
  adIds: string[];
};

export interface MetaAdsWriter {
  readonly id: "meta" | "mock";
  createCampaign(draft: WriterDraft, ctx: WriterContext): Promise<CreateCampaignResult>;
}

/** Why a submission falls back to simulation. Order = order shown in the UI. */
export type WriterGateReason = "META_NOT_CONFIGURED" | "DEMO_CLIENT" | "NO_CONNECTION" | "MISSING_ADS_MANAGEMENT";

export type WriterGateInput = {
  metaConfigured: boolean;
  demoClient: boolean;
  /** Ad account linked to a real (non-demo) Meta connection with a usable token. */
  realConnection: boolean;
  /** `Connection.scopes` of that connection. */
  scopes: string[];
};

export type WriterGate = { mode: "real" | "simulation"; reasons: WriterGateReason[] };

export const ADS_MANAGEMENT_SCOPE = "ads_management";
