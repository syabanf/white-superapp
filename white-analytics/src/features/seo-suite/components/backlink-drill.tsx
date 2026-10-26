"use client";

import { DrillSheet, bucketLabel, bucketRange, type DrillTarget } from "@/features/erp/shared/drill";
import { StatusBadge } from "@/components/dashboard/status-badge";
import type { BacklinkRow } from "@/features/seo-suite/queries";
import { s } from "@/features/seo-suite/strings";
import { formatDate, formatNumber, truncate } from "@/lib/format";

/** `week:<monday>` from the new/lost bars, `day:<date>` from the referring-domain trend. */
export const BACKLINK_DRILL = { week: "week", day: "day" } as const;

const within = (iso: string, from: string, to: string) => iso >= from && iso <= to;

/** Backlinks first seen (and lost) in the clicked week or day, from the rows already on the page. */
export function BacklinkDrillSheet({ rows, listHref }: { rows: BacklinkRow[]; listHref: string }) {
  const range = ({ kind, key }: DrillTarget) => bucketRange(key, kind === BACKLINK_DRILL.week ? "week" : "day");
  return (
    <DrillSheet
      kind={Object.values(BACKLINK_DRILL)}
      title={({ kind, key }) => `${s.backlinksKpi} ${bucketLabel(key, kind === BACKLINK_DRILL.week ? "week" : "day")}`}
      description={() => s.drillDesc}
      load={(target) => {
        const { from, to } = range(target);
        const gained = rows.filter((r) => within(r.firstSeen, from, to));
        const lost = rows.filter((r) => r.isLost && within(r.lastSeen, from, to));
        return {
          summary: [
            { label: s.newBacklinks, value: formatNumber(gained.length) },
            { label: s.lostBacklinks, value: formatNumber(lost.length) },
          ],
          items: [...gained, ...lost].map((r) => ({
            id: r.id,
            title: r.sourceDomain,
            subtitle: truncate(r.anchor || r.sourceUrl, 60),
            badge: r.isLost ? <StatusBadge kind="critical">{s.tabLost}</StatusBadge> : <StatusBadge kind="good">{s.tabNew}</StatusBadge>,
            value: r.domainRank == null ? "–" : formatNumber(r.domainRank),
            meta: formatDate(r.isLost ? r.lastSeen : r.firstSeen),
            href: r.sourceUrl,
          })),
        };
      }}
      fullListHref={() => listHref}
    />
  );
}
