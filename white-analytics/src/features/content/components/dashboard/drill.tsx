"use client";

import * as React from "react";
import {
  DrillBreadcrumb,
  DrillSheet,
  useDrill,
  type DrillCrumb,
  type DrillResult,
} from "@/features/erp/shared/drill";
import { platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel, monthLabel } from "@/features/publishing/time";
import { describeBoardFilters, matchItem, parseBoardFilters } from "@/features/content/board-filters";
import { proxyRequest } from "@/features/content/client";
import { contentFiles, isOverdue, num } from "@/features/content/lib";
import { c, cbf, cd } from "@/features/content/strings";
import type { DashPost } from "@/features/content/dashboard";
import type { ContentFormat, Item } from "@/features/content/types";
import { formatCompact, formatNumber } from "@/lib/format";
import { ItemStatusBadge } from "../status";
import type { DashCtx } from "./shared";
import { overviewHref, scopedDrillQuery, unzoomPatch, zoomPatch, type ContentDrill } from "./links";

/** Drill kind of the content dashboard sheet; the key is a board filter query. */
export const CONTENT_DRILL = "konten";

const LIMIT = 50;

/** Opens the content drill sheet for the dashboard scope narrowed by `patch`. */
export function useContentDrill(ctx: DashCtx) {
  const { open } = useDrill();
  return React.useCallback(
    (patch: ContentDrill) => open(CONTENT_DRILL, scopedDrillQuery(ctx, patch).toString()),
    [ctx, open],
  );
}

/**
 * One sheet for every content mark: lists the items behind it (fetched
 * through the gateway proxy and filtered like the board), with engagement
 * for measured posts and a footer link to the board with the same filters.
 */
export function ContentDrillSheet({
  ctx,
  posts,
  granularity,
}: {
  ctx: DashCtx;
  /** measured posts of the period, for the engagement column */
  posts: Pick<DashPost, "itemId" | "hasMetrics" | "engagements">[];
  /** "month" offers a zoom into a month bucket */
  granularity: "week" | "month";
}) {
  const engagement = React.useMemo(
    () => new Map(posts.filter((p) => p.hasMetrics).map((p) => [p.itemId, num(p.engagements)])),
    [posts],
  );

  const load = React.useCallback(
    async ({ key }: { key: string }): Promise<DrillResult> => {
      const q = new URLSearchParams(key);
      const filters = parseBoardFilters(q);
      const api = new URLSearchParams();
      if (q.get("plan")) api.set("planId", q.get("plan")!);
      if (q.get("assignee")) api.set("assigneeId", q.get("assignee")!);
      const res = await proxyRequest<{ items?: Item[] }>(contentFiles.items(ctx.clientId, api.toString()));
      if (!res.ok) throw new Error(res.error);
      const matched = (res.data.items ?? [])
        .filter((it) => matchItem(it, filters, ctx.today))
        .sort((a, b) => b.plannedDate.localeCompare(a.plannedDate));
      const published = matched.filter((it) => it.status === "PUBLISHED").length;
      const overdue = matched.filter((it) => isOverdue(it, ctx.today)).length;
      const eng = matched.reduce((s, it) => s + (engagement.get(it.id) ?? 0), 0);
      return {
        total: matched.length,
        summary: [
          { label: cbf.drillCount, value: formatNumber(matched.length) },
          eng > 0
            ? { label: cbf.drillEngagement, value: formatCompact(eng) }
            : overdue > 0
              ? { label: cbf.drillOverdue, value: formatNumber(overdue) }
              : { label: cbf.drillPublished, value: formatNumber(published) },
        ],
        items: matched.slice(0, LIMIT).map((it) => {
          const e = engagement.get(it.id);
          return {
            id: it.id,
            title: it.title,
            subtitle: `${platformLabel(it.platform)} · ${c.format[it.format as ContentFormat]} · ${dayLabel(it.plannedDate, true)}`,
            badge: <ItemStatusBadge status={it.status} className="h-5 shrink-0 px-1.5 text-[10px]" />,
            value: e != null ? formatNumber(e) : undefined,
            meta:
              e != null ? (
                cd.engagement
              ) : isOverdue(it, ctx.today) ? (
                <span className="text-negative">{cbf.overdue}</span>
              ) : undefined,
            href: `${ctx.base}/board/${it.id}`,
          };
        }),
      };
    },
    [ctx.base, ctx.clientId, ctx.today, engagement],
  );

  return (
    <DrillSheet
      kind={CONTENT_DRILL}
      title={() => cbf.drillTitle}
      chips={({ key }) => {
        const q = new URLSearchParams(key);
        const chips = describeBoardFilters(parseBoardFilters(q)).map((x) => x.label);
        const assignee = q.get("assignee");
        if (assignee) chips.push(ctx.names[assignee] ?? cd.unknownPerson);
        return chips;
      }}
      actions={({ key }) => {
        // a month bucket can zoom the whole dashboard into that month (weekly buckets)
        const f = parseBoardFilters(new URLSearchParams(key));
        if (granularity !== "month" || !f.dateFrom || !f.dateTo || f.dateFrom === f.dateTo) return [];
        if (f.dateFrom === ctx.from && f.dateTo === ctx.to) return [];
        return [
          {
            label: cbf.zoomIn(monthLabel(f.dateFrom.slice(0, 7))),
            href: overviewHref(ctx, zoomPatch(ctx, f.dateFrom, f.dateTo)),
          },
        ];
      }}
      load={load}
      fullListHref={({ key }) => {
        const q = new URLSearchParams(key);
        if (!q.has("plan")) q.set("plan", "all");
        return `${ctx.base}/board?${q.toString()}`;
      }}
      fullListLabel={cbf.openOnBoard}
    />
  );
}

/**
 * "Semua > Instagram > Reel" trail for the dashboard dimension filters and a
 * zoomed period. Each crumb drops the filters after it.
 */
export function ContentDrillTrail({ ctx, pillarsLabel }: { ctx: DashCtx; pillarsLabel?: string }) {
  const q = new URLSearchParams(ctx.query);
  const levels: { param: string; label: string }[] = [];
  const zoom = q.get("up");
  if (zoom && q.get("from") && q.get("to")) levels.push({ param: "zoom", label: ctx.periodLabel });
  for (const param of ["platform", "format", "pillar", "assignee", "plan"] as const) {
    const v = q.get(param);
    if (!v) continue;
    const label =
      param === "platform"
        ? platformLabel(v)
        : param === "format"
          ? (c.format[v as ContentFormat] ?? v)
          : param === "assignee"
            ? (ctx.names[v] ?? cd.unknownPerson)
            : param === "plan"
              ? monthLabelOf(ctx, v)
              : `${pillarsLabel ?? cd.fPillar}: ${v}`;
    levels.push({ param, label });
  }
  if (levels.length === 0) return null;
  const clear = (from: number): Record<string, string | null> => {
    const patch: Record<string, string | null> = {};
    for (const l of levels.slice(from)) {
      if (l.param === "zoom") Object.assign(patch, unzoomPatch(zoom!));
      else patch[l.param] = null;
    }
    return patch;
  };
  const items: DrillCrumb[] = [
    { label: cbf.allContent, href: overviewHref(ctx, clear(0)) },
    ...levels.map((l, i) => ({
      label: l.label,
      href: i < levels.length - 1 ? overviewHref(ctx, clear(i + 1)) : undefined,
    })),
  ];
  return <DrillBreadcrumb items={items} />;
}

function monthLabelOf(ctx: DashCtx, planId: string): string {
  const month = ctx.plans.find((p) => p.id === planId)?.month;
  return month ? monthLabel(month) : planId;
}
