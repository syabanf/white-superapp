/**
 * Campaign builder — pure logic (no I/O). Objective defaults, option lists,
 * budget/schedule validation, completeness, AI merge and the "perkiraan"
 * reach heuristic. Tested in tests/campaigns.test.ts. Components and actions
 * import from here; never re-derive these rules elsewhere.
 */
import type { WriterAudience } from "@/lib/providers/meta-ads-writer/types";
import { c } from "./strings";

export const OBJECTIVES = ["OUTCOME_AWARENESS", "OUTCOME_TRAFFIC", "OUTCOME_ENGAGEMENT", "OUTCOME_LEADS", "OUTCOME_SALES"] as const;
export type Objective = (typeof OBJECTIVES)[number];
export const isObjective = (v: unknown): v is Objective => typeof v === "string" && (OBJECTIVES as readonly string[]).includes(v);

export const DRAFT_STATUSES = ["DRAFT", "READY", "SUBMITTED"] as const;
export type DraftStatus = (typeof DRAFT_STATUSES)[number];

export const CTA_OPTIONS = ["LEARN_MORE", "SHOP_NOW", "SIGN_UP", "SEND_MESSAGE", "WHATSAPP_MESSAGE", "CONTACT_US", "BOOK_TRAVEL", "GET_OFFER"] as const;
export type Cta = (typeof CTA_OPTIONS)[number];
export const isCta = (v: unknown): v is Cta => typeof v === "string" && (CTA_OPTIONS as readonly string[]).includes(v);

export const PLACEMENTS = [
  "instagram_feed",
  "instagram_stories",
  "instagram_reels",
  "instagram_explore",
  "facebook_feed",
  "facebook_stories",
  "facebook_reels",
  "facebook_marketplace",
  "audience_network",
] as const;
export type Placement = (typeof PLACEMENTS)[number];
export const isPlacement = (v: unknown): v is Placement => typeof v === "string" && (PLACEMENTS as readonly string[]).includes(v);
export const placementPlatform = (p: Placement): "instagram" | "facebook" | "audience_network" =>
  p.startsWith("instagram") ? "instagram" : p.startsWith("facebook") ? "facebook" : "audience_network";

/** Sensible starting point per objective (CTA + placements). */
export const OBJECTIVE_DEFAULTS: Record<Objective, { cta: Cta; placements: Placement[] }> = {
  OUTCOME_AWARENESS: { cta: "LEARN_MORE", placements: ["instagram_feed", "instagram_stories", "instagram_reels", "facebook_feed"] },
  OUTCOME_TRAFFIC: { cta: "LEARN_MORE", placements: ["instagram_feed", "instagram_stories", "facebook_feed"] },
  OUTCOME_ENGAGEMENT: { cta: "SEND_MESSAGE", placements: ["instagram_feed", "instagram_reels", "facebook_feed"] },
  OUTCOME_LEADS: { cta: "SIGN_UP", placements: ["instagram_feed", "instagram_stories", "facebook_feed"] },
  OUTCOME_SALES: { cta: "SHOP_NOW", placements: ["instagram_feed", "instagram_stories", "instagram_reels", "facebook_feed"] },
};

export type Audience = WriterAudience;
export type Creative = { headline: string; primaryText: string; cta: string; mediaUrl: string | null };

export const CREATIVE_SLOTS = 3;
export const AGE_MIN = 18;
export const AGE_MAX = 65;
export const MAX_INTERESTS = 10;
export const MAX_LOCATIONS = 10;
/** Meta's recommended lengths (longer text gets truncated in feed). */
export const HEADLINE_RECOMMENDED = 40;
export const PRIMARY_TEXT_RECOMMENDED = 125;
export const HEADLINE_MAX = 80;
export const PRIMARY_TEXT_MAX = 600;

export const DEFAULT_AUDIENCE: Audience = { locations: [], ageMin: AGE_MIN, ageMax: AGE_MAX, genders: "ALL", interests: [] };
export const emptyCreative = (): Creative => ({ headline: "", primaryText: "", cta: "", mediaUrl: null });

// ── parsing JSON columns ─────────────────────────────────────

const strList = (v: unknown, max: number): string[] =>
  Array.isArray(v) ? Array.from(new Set(v.filter((x): x is string => typeof x === "string").map((x) => x.trim()).filter(Boolean))).slice(0, max) : [];
const intOr = (v: unknown, fallback: number) => (typeof v === "number" && Number.isFinite(v) ? Math.round(v) : fallback);
const clampAge = (n: number) => Math.min(AGE_MAX, Math.max(AGE_MIN, n));

export function parseAudience(json: unknown): Audience {
  const o = (json && typeof json === "object" ? json : {}) as Record<string, unknown>;
  const ageMin = clampAge(intOr(o.ageMin, AGE_MIN));
  const ageMax = clampAge(intOr(o.ageMax, AGE_MAX));
  return {
    locations: strList(o.locations, MAX_LOCATIONS),
    ageMin: Math.min(ageMin, ageMax),
    ageMax: Math.max(ageMin, ageMax),
    genders: o.genders === "MALE" || o.genders === "FEMALE" ? o.genders : "ALL",
    interests: strList(o.interests, MAX_INTERESTS),
  };
}

/** Always returns exactly CREATIVE_SLOTS entries (missing slots are empty). */
export function parseCreatives(json: unknown): Creative[] {
  const arr = Array.isArray(json) ? json : [];
  return Array.from({ length: CREATIVE_SLOTS }, (_, i) => {
    const o = (arr[i] && typeof arr[i] === "object" ? arr[i] : {}) as Record<string, unknown>;
    return {
      headline: typeof o.headline === "string" ? o.headline : "",
      primaryText: typeof o.primaryText === "string" ? o.primaryText : "",
      cta: typeof o.cta === "string" ? o.cta : "",
      mediaUrl: typeof o.mediaUrl === "string" && o.mediaUrl ? o.mediaUrl : null,
    };
  });
}

export const parsePlacements = (v: unknown): Placement[] => (Array.isArray(v) ? Array.from(new Set(v.filter(isPlacement))) : []);

export const isCreativeEmpty = (cr: Creative) => !cr.headline.trim() && !cr.primaryText.trim() && !cr.mediaUrl;
/** Creatives that will be sent (non-empty slots). */
export const filledCreatives = (list: Creative[]) => list.filter((cr) => !isCreativeEmpty(cr));

// ── validation ───────────────────────────────────────────────

export function minDailyBudget(currency: string): number {
  return currency.toUpperCase() === "IDR" ? 10_000 : 1;
}

/** null = valid; otherwise the message. */
export function validateBudget(amount: number, currency: string): string | null {
  if (!Number.isFinite(amount) || amount <= 0) return c.issues.budgetRequired;
  const min = minDailyBudget(currency);
  if (amount < min) return c.issues.budgetMin(min, currency);
  return null;
}

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;
export const isDateKey = (v: unknown): v is string => typeof v === "string" && DATE_RE.test(v) && !Number.isNaN(Date.parse(`${v}T00:00:00Z`));

/** Dates are `YYYY-MM-DD` keys (string compare is chronological). */
export function validateSchedule(start: string, end: string | null, today: string): string | null {
  if (!isDateKey(start)) return c.issues.startRequired;
  if (start < today) return c.issues.startPast;
  if (end != null && end !== "") {
    if (!isDateKey(end)) return c.issues.endInvalid;
    if (end <= start) return c.issues.endBeforeStart;
  }
  return null;
}

export type DraftInput = {
  name: string;
  objective: string;
  dailyBudget: number;
  currency: string;
  startDate: string;
  endDate: string | null;
  adAccountId: string | null;
  audience: Audience;
  placements: string[];
  creatives: Creative[];
};

export type IssueField = "name" | "objective" | "budget" | "schedule" | "adAccount" | "audience" | "placements" | "creatives" | `creative-${number}`;
export type DraftIssue = { field: IssueField; message: string };

export function draftIssues(d: DraftInput, opts: { today: string; requireAdAccount: boolean }): DraftIssue[] {
  const out: DraftIssue[] = [];
  if (!d.name.trim()) out.push({ field: "name", message: c.issues.nameRequired });
  if (!isObjective(d.objective)) out.push({ field: "objective", message: c.issues.objectiveRequired });
  const budget = validateBudget(d.dailyBudget, d.currency);
  if (budget) out.push({ field: "budget", message: budget });
  const schedule = validateSchedule(d.startDate, d.endDate, opts.today);
  if (schedule) out.push({ field: "schedule", message: schedule });
  if (opts.requireAdAccount && !d.adAccountId) out.push({ field: "adAccount", message: c.issues.adAccountRequired });
  if (!d.audience.locations.length) out.push({ field: "audience", message: c.issues.locationRequired });
  if (d.audience.ageMin > d.audience.ageMax) out.push({ field: "audience", message: c.issues.ageOrder });
  // Empty placements are allowed: they mean Advantage+ (automatic) placements.
  if (d.placements.some((p) => !isPlacement(p))) out.push({ field: "placements", message: c.issues.placementUnknown });
  const filled = d.creatives.map((cr, i) => ({ cr, i })).filter(({ cr }) => !isCreativeEmpty(cr));
  if (!filled.length) out.push({ field: "creatives", message: c.issues.creativeRequired });
  for (const { cr, i } of filled) {
    const n = i + 1;
    if (!cr.headline.trim()) out.push({ field: `creative-${i}`, message: c.issues.headlineRequired(n) });
    else if (cr.headline.length > HEADLINE_MAX) out.push({ field: `creative-${i}`, message: c.issues.headlineTooLong(n, HEADLINE_MAX) });
    if (!cr.primaryText.trim()) out.push({ field: `creative-${i}`, message: c.issues.primaryRequired(n) });
    else if (cr.primaryText.length > PRIMARY_TEXT_MAX) out.push({ field: `creative-${i}`, message: c.issues.primaryTooLong(n, PRIMARY_TEXT_MAX) });
    if (!isCta(cr.cta)) out.push({ field: `creative-${i}`, message: c.issues.ctaRequired(n) });
  }
  return out;
}

export const isDraftComplete = (d: DraftInput, opts: { today: string; requireAdAccount: boolean }) => draftIssues(d, opts).length === 0;

/** Saving never promotes to READY (that is "Tandai siap"), but an edit that breaks completeness demotes. */
export function statusAfterSave(prev: DraftStatus | null, complete: boolean): DraftStatus {
  return prev === "READY" && complete ? "READY" : "DRAFT";
}

export const canEditDraft = (status: DraftStatus) => status !== "SUBMITTED";
export const canSubmitDraft = (status: DraftStatus) => status === "READY";

// ── AI merge ─────────────────────────────────────────────────

export type Suggestion = {
  audience: Audience;
  placements: string[];
  creatives: { headline: string; primaryText: string; cta: string }[];
  notes: string;
};

/**
 * Fill what the user left empty with the AI suggestion; never overwrite what
 * they typed. Defaults (age 18–65, gender ALL) count as "not chosen".
 * Interests are unioned (user first). Placements are taken only when none are
 * picked. Creatives merge per field; the user's image stays.
 */
export function mergeSuggestion(
  current: { audience: Audience; placements: string[]; creatives: Creative[] },
  s: Suggestion,
  objective: Objective,
): { audience: Audience; placements: Placement[]; creatives: Creative[]; aiNotes: string } {
  const ai = parseAudience(s.audience);
  const cur = current.audience;
  const ageUntouched = cur.ageMin === DEFAULT_AUDIENCE.ageMin && cur.ageMax === DEFAULT_AUDIENCE.ageMax;
  const audience: Audience = {
    locations: cur.locations.length ? cur.locations : ai.locations,
    ageMin: ageUntouched ? ai.ageMin : cur.ageMin,
    ageMax: ageUntouched ? ai.ageMax : cur.ageMax,
    genders: cur.genders === "ALL" ? ai.genders : cur.genders,
    interests: Array.from(new Set([...cur.interests, ...ai.interests])).slice(0, MAX_INTERESTS),
  };
  const aiPlacements = parsePlacements(s.placements);
  const placements = current.placements.length
    ? parsePlacements(current.placements)
    : aiPlacements.length
      ? aiPlacements
      : OBJECTIVE_DEFAULTS[objective].placements;
  const creatives = Array.from({ length: CREATIVE_SLOTS }, (_, i) => {
    const mine = current.creatives[i] ?? emptyCreative();
    const theirs = s.creatives[i];
    const aiCta = theirs && isCta(theirs.cta) ? theirs.cta : OBJECTIVE_DEFAULTS[objective].cta;
    return {
      headline: mine.headline.trim() ? mine.headline : (theirs?.headline ?? "").slice(0, HEADLINE_MAX),
      primaryText: mine.primaryText.trim() ? mine.primaryText : (theirs?.primaryText ?? "").slice(0, PRIMARY_TEXT_MAX),
      cta: isCta(mine.cta) ? mine.cta : aiCta,
      mediaUrl: mine.mediaUrl,
    };
  });
  return { audience, placements, creatives, aiNotes: s.notes.trim() };
}

// ── estimates ────────────────────────────────────────────────

/** Rough CPM band per currency (major units per 1.000 impressions). Heuristic, not a Meta forecast. */
const CPM_BAND: Record<string, [number, number]> = { IDR: [12_000, 35_000], USD: [2, 8] };
const AVG_FREQUENCY = 1.4;

/** "Perkiraan" daily reach range, or null when we have no CPM band for the currency. */
export function estimateDailyReach(dailyBudget: number, currency: string): { low: number; high: number } | null {
  const band = CPM_BAND[currency.toUpperCase()];
  if (!band || !Number.isFinite(dailyBudget) || dailyBudget <= 0) return null;
  const reach = (cpm: number) => Math.round(((dailyBudget / cpm) * 1000) / AVG_FREQUENCY / 10) * 10;
  return { low: reach(band[1]), high: reach(band[0]) };
}

/** Number of days between two date keys, inclusive; null when open-ended. */
export function scheduleDays(start: string, end: string | null): number | null {
  if (!isDateKey(start) || !end || !isDateKey(end)) return null;
  return Math.round((Date.parse(`${end}T00:00:00Z`) - Date.parse(`${start}T00:00:00Z`)) / 86_400_000) + 1;
}

/** Today's `YYYY-MM-DD` in a time zone (server-side, for validation). */
export function todayKey(now: Date, timeZone: string): string {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(now);
  const get = (t: string) => parts.find((p) => p.type === t)?.value ?? "";
  return `${get("year")}-${get("month")}-${get("day")}`;
}

export const dateKeyOf = (d: Date) => d.toISOString().slice(0, 10);
export const dateFromKey = (key: string) => new Date(`${key}T00:00:00Z`);
