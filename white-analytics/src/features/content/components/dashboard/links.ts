/**
 * Link builders for the content dashboard. Pure (no "use client"), so the
 * server page and the client tabs share them.
 */
import { boardFilterQuery, parseBoardFilters, type BoardFilters } from "@/features/content/board-filters";
import { DRILL_PARAM } from "@/features/erp/shared/drill/href";
import type { DashCtx } from "./shared";

export type ContentDrill = Partial<BoardFilters> & { assignee?: string; plan?: string };

/** Overview URL with one filter changed (drill-down within the page); an open drill sheet is dropped. */
export function overviewHref(ctx: DashCtx, patch: Record<string, string | null>, tab?: string): string {
  const q = new URLSearchParams(ctx.query);
  q.delete(DRILL_PARAM);
  for (const [k, v] of Object.entries(patch)) {
    if (v) q.set(k, v);
    else q.delete(k);
  }
  if (tab) q.set("tab", tab);
  const s = q.toString();
  return s ? `${ctx.base}?${s}` : ctx.base;
}

/** Dashboard dimension filters (platform, format, pillar, assignee, plan) as a drill base. */
function dashScope(ctx: DashCtx): ContentDrill {
  const q = new URLSearchParams(ctx.query);
  const f = parseBoardFilters(q);
  return {
    platform: f.platform,
    format: f.format,
    pillar: f.pillar,
    assignee: q.get("assignee") ?? "",
    plan: q.get("plan") ?? "",
  };
}

function drillQuery(drill: ContentDrill): URLSearchParams {
  const q = boardFilterQuery(drill);
  if (drill.plan) q.set("plan", drill.plan);
  if (drill.assignee) q.set("assignee", drill.assignee);
  return q;
}

/** Drill query: the dashboard scope narrowed by `patch`. */
export function scopedDrillQuery(ctx: DashCtx, patch: ContentDrill = {}): URLSearchParams {
  return drillQuery({ ...dashScope(ctx), ...patch });
}

/** Board URL with the dashboard scope plus `patch` (plan defaults to all plans). */
export function contentBoardHref(ctx: DashCtx, patch: ContentDrill = {}): string {
  const q = scopedDrillQuery(ctx, patch);
  if (!q.has("plan")) q.set("plan", "all");
  return `${ctx.base}/board?${q.toString()}`;
}

/** The period of the dashboard as planned-date bounds. */
export function periodRange(ctx: DashCtx): Pick<BoardFilters, "dateFrom" | "dateTo"> {
  return { dateFrom: ctx.from, dateTo: ctx.to };
}

/** `up` holds the period before a zoom: "preset:28d" or "range:2026-01-01..2026-06-30". */
export function zoomPatch(ctx: DashCtx, from: string, to: string): Record<string, string | null> {
  const q = new URLSearchParams(ctx.query);
  const preset = q.get("preset");
  const up =
    q.get("up") ??
    (q.get("from") && q.get("to") ? `range:${q.get("from")}..${q.get("to")}` : `preset:${preset ?? ""}`);
  return { from, to, preset: null, up };
}

export function unzoomPatch(up: string): Record<string, string | null> {
  if (up.startsWith("range:")) {
    const [from, to] = up.slice(6).split("..");
    return { from: from ?? null, to: to ?? null, up: null };
  }
  return { from: null, to: null, preset: up.slice(7) || null, up: null };
}
