"use client";

import { usePathname, useSearchParams } from "next/navigation";
import { DrillSheet, bucketLabel, withParams } from "@/features/erp/shared/drill";
import { getSeoDayDrill } from "@/features/seo/drill-actions";
import { s } from "@/features/seo/strings";
import { formatNumber, formatPercent, truncate } from "@/lib/format";
import { t } from "@/i18n/id";

export const SEO_DRILL = { day: "day" } as const;

/** Sheet for a daily SEO point (`?drill=day:2026-10-01`): that day's queries by clicks. */
export function SeoDrillSheet({ slug }: { slug: string }) {
  const pathname = usePathname();
  const params = useSearchParams();
  const base = `/clients/${slug}/seo`;
  const period = (path: string, date: string, extra: Record<string, string> = {}) =>
    withParams(path, params.toString(), { from: date, to: date, preset: null, ...extra }, ["drill", "page"]);

  return (
    <DrillSheet
      kind={SEO_DRILL.day}
      title={({ key }) => `${t.seo.keywords} ${bucketLabel(key, "day")}`}
      description={() => s.drillDayDesc}
      actions={({ key }) => [
        { label: s.drillZoomDay, href: period(pathname, key) },
        { label: t.seo.pages, href: period(`${base}/pages`, key) },
      ]}
      load={async ({ key }) => {
        const d = await getSeoDayDrill({ slug, date: key });
        return {
          summary: [
            { label: t.seo.clicks, value: formatNumber(d.totals.clicks) },
            { label: t.seo.impressions, value: formatNumber(d.totals.impressions) },
            { label: t.seo.position, value: d.totals.position == null ? "–" : formatNumber(d.totals.position, 1) },
            { label: t.seo.keywords, value: formatNumber(d.total) },
          ],
          total: d.total,
          items: d.queries.map((q) => ({
            id: q.key,
            title: truncate(q.key, 56),
            subtitle: `${t.seo.ctr} ${formatPercent(q.ctr)} · ${t.seo.position} ${formatNumber(q.position, 1)}`,
            value: formatNumber(q.clicks),
            meta: `${formatNumber(q.impressions)} ${t.seo.impressions.toLowerCase()}`,
            href: period(`${base}/keywords`, key, { q: q.key }),
          })),
        };
      }}
      fullListHref={({ key }) => period(`${base}/keywords`, key)}
      emptyLabel={t.common.noData}
    />
  );
}
