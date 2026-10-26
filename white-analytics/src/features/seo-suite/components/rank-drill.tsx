"use client";

import { DrillSheet, bucketLabel } from "@/features/erp/shared/drill";
import { DeltaText } from "@/components/dashboard/delta-badge";
import { getRankDayDrill } from "@/features/seo-suite/drill-actions";
import { s } from "@/features/seo-suite/strings";
import { formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";

export const RANK_DRILL = { day: "day" } as const;

/** Positions of every tracked keyword on the clicked trend day; lower is better. */
export function RankDrillSheet({ slug, listHref }: { slug: string; listHref: string }) {
  return (
    <DrillSheet
      kind={RANK_DRILL.day}
      title={({ key }) => `${t.seo.position} ${bucketLabel(key, "day")}`}
      description={() => s.rankDrillDesc}
      load={async ({ key }) => {
        const rows = await getRankDayDrill({ slug, date: key });
        const ranked = rows.filter((r) => r.position != null);
        return {
          summary: [
            { label: s.ranked, value: formatNumber(ranked.length) },
            { label: s.top10, value: formatNumber(ranked.filter((r) => r.position! <= 10).length) },
          ],
          items: rows.map((r) => ({
            id: r.id,
            title: r.keyword,
            subtitle: r.device === "MOBILE" ? t.seo.mobile : t.seo.desktop,
            value: r.position == null ? s.unranked : formatNumber(r.position),
            meta:
              r.position != null && r.previous != null ? (
                <DeltaText current={r.position} previous={r.previous} mode="abs" lowerIsBetter />
              ) : undefined,
          })),
        };
      }}
      fullListHref={() => listHref}
    />
  );
}
