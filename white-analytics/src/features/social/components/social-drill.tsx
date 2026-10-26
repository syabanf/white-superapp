"use client";

import { usePathname, useSearchParams } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { DrillSheet, bucketLabel, withParams, type DrillResult } from "@/features/erp/shared/drill";
import { filterPosts, type PostFilter } from "@/features/social/post-filters";
import type { GrowthPoint, ReachPoint, SocialPostRow } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { formatCompact, formatDateTime, formatNumber, formatPercent, truncate } from "@/lib/format";
import { mean } from "@/lib/metrics";
import { t } from "@/i18n/id";

/** Drill kinds on the social dashboard: a day (growth/reach point), a content type, a weekday-hour slot. */
export const SOCIAL_DRILL = { day: "day", type: "type", slot: "slot" } as const;

const hourLabel = (h: number) => `${String(h).padStart(2, "0")}:00`;

function parseSlot(key: string): { day: number; hour: number } {
  const [d, h] = key.split("-").map(Number);
  return { day: d ?? 0, hour: h ?? 0 };
}

function filterFor(kind: string, key: string): PostFilter {
  if (kind === SOCIAL_DRILL.type) return { type: key };
  if (kind === SOCIAL_DRILL.slot) return parseSlot(key);
  return { date: key };
}

function postsResult(posts: SocialPostRow[], extra: DrillResult["summary"] = []): DrillResult {
  const sorted = [...posts].sort((a, b) => b.engagements - a.engagements);
  return {
    summary: [
      { label: t.social.posts, value: formatNumber(posts.length) },
      { label: s.avgEr, value: posts.length ? formatPercent(mean(posts.map((p) => p.er))) : "–" },
      ...extra,
    ],
    items: sorted.map((p) => ({
      id: p.id,
      title: truncate(p.caption || "–", 60),
      subtitle: formatDateTime(p.publishedAt),
      badge: <Badge variant="outline">{p.type}</Badge>,
      value: formatCompact(p.engagements),
      meta: `${t.social.er} ${formatPercent(p.er)}`,
      href: p.permalink ?? undefined,
    })),
  };
}

/**
 * One sheet for every social drill (`?drill=day:2026-10-01`, `type:Reels`, `slot:0-19`).
 * Records come from the posts already on the page, so it never fetches.
 */
export function SocialDrillSheet({
  posts,
  growth,
  reach,
}: {
  posts: SocialPostRow[];
  growth: GrowthPoint[];
  reach: ReachPoint[];
}) {
  const pathname = usePathname();
  const params = useSearchParams();
  const list = (patch: Record<string, string | number | null>) =>
    `${withParams(pathname, params.toString(), patch, ["drill", "type", "day", "hour", "sort"])}#posts`;

  return (
    <DrillSheet
      kind={Object.values(SOCIAL_DRILL)}
      title={({ kind, key }) => {
        if (kind === SOCIAL_DRILL.type) return `${t.social.type}: ${key}`;
        if (kind === SOCIAL_DRILL.slot) {
          const { day, hour } = parseSlot(key);
          return `${s.days[day] ?? day}, ${hourLabel(hour)}`;
        }
        return `${t.social.posts} ${bucketLabel(key, "day")}`;
      }}
      description={({ kind }) => (kind === SOCIAL_DRILL.day ? s.drillDayDesc : s.drillPostsDesc)}
      actions={({ kind, key }) =>
        kind === SOCIAL_DRILL.day
          ? [{ label: s.drillZoomDay, href: withParams(pathname, params.toString(), { from: key, to: key, preset: null }, ["drill"]) }]
          : []
      }
      load={({ kind, key }) => {
        const matched = filterPosts(posts, filterFor(kind, key));
        if (kind !== SOCIAL_DRILL.day) return postsResult(matched);
        const g = growth.find((p) => p.date === key);
        const r = reach.find((p) => p.date === key);
        return postsResult(matched, [
          { label: t.social.followers, value: g?.followers == null ? "–" : formatNumber(g.followers) },
          { label: t.social.reach, value: r ? formatCompact(r.reach) : "–" },
        ]);
      }}
      fullListHref={({ kind, key }) => {
        if (kind === SOCIAL_DRILL.type) return list({ type: key });
        if (kind === SOCIAL_DRILL.slot) return list(parseSlot(key));
        return list({ from: key, to: key, preset: null });
      }}
      emptyLabel={s.noPosts}
    />
  );
}

