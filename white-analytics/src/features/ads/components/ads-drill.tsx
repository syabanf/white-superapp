"use client";

import { usePathname, useSearchParams } from "next/navigation";
import { DrillSheet, bucketLabel, withParams, type DrillTarget } from "@/features/erp/shared/drill";
import { getAdsDayDrill } from "@/features/ads/drill-actions";
import type { DemographicRow } from "@/features/ads/components/demographics-card";
import { genderLabel, s } from "@/features/ads/strings";
import { formatCurrency, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";

/** Drill kinds on the ads dashboard: a day (daily bars), an age group, a gender. */
export const ADS_DRILL = { day: "day", age: "age", gender: "gender" } as const;

/**
 * One sheet for every ads drill. Days load campaigns of that day through a
 * server action; age and gender rows come from the demographics already on the page.
 */
export function AdsDrillSheet({ slug, currency, matrix }: { slug: string; currency: string; matrix: DemographicRow[] }) {
  const pathname = usePathname();
  const params = useSearchParams();
  const campaign = params.get("campaign");
  const money = (v: number) => formatCurrency(v, currency);
  const filterHref = (patch: Record<string, string | null>) => withParams(pathname, params.toString(), patch, ["drill"]);

  const demographic = ({ kind, key }: DrillTarget) => {
    const rows = matrix.filter((m) => (kind === ADS_DRILL.age ? m.age === key : m.gender === key));
    const spend = rows.reduce((a, r) => a + r.spend, 0);
    const results = rows.reduce((a, r) => a + r.results, 0);
    return {
      summary: [
        { label: t.ads.spend, value: money(spend) },
        { label: t.ads.results, value: formatNumber(results) },
        { label: t.ads.cpr, value: results > 0 ? money(spend / results) : "–" },
      ],
      items: rows
        .sort((a, b) => b.spend - a.spend)
        .map((r) => ({
          id: `${r.age}|${r.gender}`,
          title: kind === ADS_DRILL.age ? genderLabel(r.gender) : r.age,
          value: money(r.spend),
          meta: `${formatNumber(r.results)} ${t.ads.results.toLowerCase()}`,
        })),
    };
  };

  return (
    <DrillSheet
      kind={Object.values(ADS_DRILL)}
      title={({ kind, key }) =>
        kind === ADS_DRILL.day
          ? `${t.ads.campaign} ${bucketLabel(key, "day")}`
          : kind === ADS_DRILL.age
            ? `${t.ads.age} ${key}`
            : genderLabel(key)
      }
      description={({ kind }) => (kind === ADS_DRILL.day ? s.drillDayDesc : s.drillDemoDesc)}
      actions={({ kind, key }) =>
        kind === ADS_DRILL.day
          ? [{ label: s.drillZoomDay, href: filterHref({ from: key, to: key, preset: null }) }]
          : []
      }
      load={async (target) => {
        if (target.kind !== ADS_DRILL.day) return demographic(target);
        const d = await getAdsDayDrill({ slug, date: target.key, campaignId: campaign });
        return {
          summary: [
            { label: t.ads.spend, value: money(d.spend) },
            { label: t.ads.results, value: formatNumber(d.results) },
            { label: t.ads.cpr, value: d.results > 0 ? money(d.cpr) : "–" },
          ],
          items: d.campaigns.map((c) => ({
            id: c.id,
            title: c.name,
            subtitle: `${t.ads.ctr} ${formatPercent(c.ctr)} · ${t.ads.cpr} ${c.results > 0 ? money(c.cpr) : "–"}`,
            value: money(c.spend),
            meta: `${formatNumber(c.results)} ${t.ads.results.toLowerCase()}`,
            href: c.id ? filterHref({ campaign: c.id }) : undefined,
          })),
        };
      }}
      fullListHref={({ kind, key }) =>
        kind === ADS_DRILL.day ? `${filterHref({ from: key, to: key, preset: null })}#breakdown` : undefined
      }
      emptyLabel={t.common.noData}
    />
  );
}
