/**
 * Content dashboard: JSON shapes of ReportService.GetDashboard and the pure
 * helpers the overview page uses (URL filters, deltas, labels). Tested in
 * tests/content-dashboard.test.ts.
 */
import { computeDelta, type Delta } from "@/lib/metrics";
import { FORMATS, PLATFORMS, num } from "./lib";
import type { AssetKind, ContentFormat, Platform } from "./types";

export type DashKpis = {
  planned: number;
  published: number;
  failed: number;
  inReview: number;
  overdue: number;
  deadlinesMet: number;
  deadlinesTotal: number;
  onTimeRate: number;
  revisionsTotal: number;
  revisionAvg: number;
  engagements: string;
  reach: string;
  engagementRate: number;
  measuredPosts: number;
  cycleDaysAvg: number;
  designTasks: number;
  designApproved: number;
  designTurnaroundDays: number;
  firstPassRate: number;
  assetUploads: number;
};

export type DashBucket = {
  start: string;
  planned: number;
  published: number;
  engagements: string;
  reach: string;
  deadlinesMet: number;
  deadlinesTotal: number;
  onTimeRate: number;
  revisionAvg: number;
  stageCopy: number;
  stageDesign: number;
  stageReview: number;
  stageReady: number;
  designTurnaroundDays: number;
  assetUploads: number;
};

export type DashCount = { key: string; count: number };
export type DashStageCycle = { stage: Stage4; avgDays: number; medianDays: number; count: number };
export type Stage4 = "COPY" | "DESIGN" | "REVIEW" | "PUBLISH";

export type DashPerson = {
  userId: string;
  role: "COPYWRITER" | "DESIGNER";
  assigned: number;
  published: number;
  overdue: number;
  overdueCopy: number;
  overdueDesign: number;
  overdueReview: number;
  revisionAvg: number;
  onTimeRate: number;
  designTasks: number;
  designApproved: number;
  turnaroundDays: number;
  firstPassRate: number;
};

export type DashDay = { date: string; planned: number; published: number };
export type DashHeatCell = { weekday: number; hour: number; posts: number; engagements: string };
export type DashBreakdown = { key: string; planned: number; published: number; engagements: string; reach: string; measured: number };

export type DashPost = {
  itemId: string;
  title: string;
  platform: Platform;
  format: ContentFormat;
  pillar: string;
  plannedDate: string;
  publishedAt?: string;
  permalink?: string;
  thumbnailFileId: string;
  hasMetrics: boolean;
  likes: string;
  comments: string;
  shares: string;
  saves: string;
  reach: string;
  views: string;
  engagements: string;
};

export type DashAsset = { assetId: string; name: string; kind: AssetKind; uploads: number; currentVersion: number; lastUploadedAt: string };

export type DashWindow = {
  granularity: "week" | "month";
  kpis: DashKpis;
  buckets?: DashBucket[];
  funnel?: DashCount[];
  statuses?: DashCount[];
  stageCycles?: DashStageCycle[];
  overdueByStage?: DashCount[];
  revisionRounds?: DashCount[];
  designVersions?: DashCount[];
  people?: DashPerson[];
  days?: DashDay[];
  postingHeat?: DashHeatCell[];
  byPlatform?: DashBreakdown[];
  byFormat?: DashBreakdown[];
  byPillar?: DashBreakdown[];
  posts?: DashPost[];
  topAssets?: DashAsset[];
  assetUploadsByKind?: DashCount[];
};

export type ContentDashboard = {
  from: string;
  to: string;
  prevFrom: string;
  prevTo: string;
  compare: CompareMode;
  current: DashWindow;
  previous: DashWindow;
  pillars?: string[];
};

export type CompareMode = "previous" | "year";

export const DASH_TABS = ["ringkasan", "produksi", "kalender", "performa", "aset"] as const;
export type DashTab = (typeof DASH_TABS)[number];

/** Dimension filters the overview keeps in the URL (the period comes from the top bar picker). */
export const FILTER_PARAMS = ["cmp", "platform", "format", "pillar", "assignee", "plan"] as const;

export type DashFilters = {
  tab: DashTab;
  compareMode: CompareMode;
  platform: Platform | "";
  format: ContentFormat | "";
  pillar: string;
  assignee: string;
  plan: string;
};

type SP = Record<string, string | string[] | undefined>;
const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export function parseDashFilters(sp: SP): DashFilters {
  const tab = one(sp.tab) as DashTab;
  const platform = one(sp.platform) as Platform;
  const format = one(sp.format) as ContentFormat;
  return {
    tab: DASH_TABS.includes(tab) ? tab : "ringkasan",
    compareMode: one(sp.cmp) === "year" ? "year" : "previous",
    platform: PLATFORMS.includes(platform) ? platform : "",
    format: FORMATS.includes(format) ? format : "",
    pillar: one(sp.pillar).slice(0, 80),
    assignee: one(sp.assignee).slice(0, 64),
    plan: one(sp.plan).slice(0, 64),
  };
}

/** Query string for GET /v1/content/dashboard/clients/{id}. */
export function dashboardQuery(from: string, to: string, f: DashFilters): string {
  const q = new URLSearchParams({ from, to, compare: f.compareMode });
  if (f.platform) q.set("platform", f.platform);
  if (f.format) q.set("format", f.format);
  if (f.pillar) q.set("pillar", f.pillar);
  if (f.assignee) q.set("assigneeId", f.assignee);
  if (f.plan) q.set("planId", f.plan);
  return q.toString();
}

/** Delta between two KPI values, int64 strings included. */
export function kpiDelta(cur: number | string, prev: number | string): Delta {
  return computeDelta(num(cur), num(prev));
}

/** Engagement per measured post (0 when none). */
export function perPost(engagements: number | string, measured: number): number {
  return measured > 0 ? num(engagements) / measured : 0;
}

/** Engagement rate in percent: engagements / reach. */
export function engagementRate(engagements: number | string, reach: number | string): number {
  const r = num(reach);
  return r > 0 ? (num(engagements) / r) * 100 : 0;
}

/** Share of planned items that were published, percent. */
export function publishRate(k: Pick<DashKpis, "planned" | "published">): number {
  return k.planned > 0 ? (k.published / k.planned) * 100 : 0;
}

/** Step-to-step conversion in a funnel, percent of the previous step. */
export function funnelSteps(counts: DashCount[]): { key: string; count: number; ofFirst: number; ofPrev: number }[] {
  const first = counts[0]?.count ?? 0;
  return counts.map((c, i) => {
    const prev = i === 0 ? c.count : counts[i - 1]!.count;
    return { key: c.key, count: c.count, ofFirst: first ? (c.count / first) * 100 : 0, ofPrev: prev ? (c.count / prev) * 100 : 0 };
  });
}

/** Ranked rows with the tail folded into one "others" row. */
export function topWithOthers<T>(rows: T[], n: number, value: (r: T) => number): { top: T[]; others: number; othersCount: number } {
  const sorted = rows.toSorted((a, b) => value(b) - value(a));
  const rest = sorted.slice(n);
  return { top: sorted.slice(0, n), others: rest.reduce((s, r) => s + value(r), 0), othersCount: rest.length };
}

export type WaterfallStep = { label: string; value: number; kind: "total" | "minus" };

/** Floating bars: totals from the baseline, each loss hanging from the running total. */
export function waterfallBars(steps: WaterfallStep[]) {
  const out: { label: string; base: number; value: number; kind: WaterfallStep["kind"] }[] = [];
  let running = 0;
  for (const s of steps) {
    running = s.kind === "total" ? s.value : running - s.value;
    out.push({ label: s.label, base: s.kind === "total" ? 0 : running, value: s.value, kind: s.kind });
  }
  return out;
}
