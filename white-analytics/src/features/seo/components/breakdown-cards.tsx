"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { formatNumber, formatPercent } from "@/lib/format";
import type { PositionBucket } from "@/lib/metrics";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import { countryLabel } from "@/features/seo/aggregate";
import { BUCKET_COLORS, BUCKET_LABELS } from "./buckets";

type DistributionRow = { bucket: PositionBucket; queries: number; clicks: number; impressions: number };

const right: ColMeta = { align: "right" };

/** (c) Distribusi posisi — ordinal seq ramp, jumlah kata kunci per bucket. */
export function PositionDistributionCard({ distribution }: { distribution: DistributionRow[] }) {
  const columns: ColumnDef<DistributionRow, unknown>[] = [
    { accessorKey: "bucket", header: t.seo.positionDist, cell: ({ row }) => BUCKET_LABELS[row.original.bucket] },
    { accessorKey: "queries", header: t.seo.keywords, meta: right, cell: ({ row }) => formatNumber(row.original.queries) },
    { accessorKey: "clicks", header: t.seo.clicks, meta: right, cell: ({ row }) => formatNumber(row.original.clicks) },
    { accessorKey: "impressions", header: t.seo.impressions, meta: right, cell: ({ row }) => formatNumber(row.original.impressions) },
  ];
  return (
    <ChartCard
      title={t.seo.positionDist}
      description={t.seo.positionDistDesc}
      table={<DataTable bare dense data={distribution} columns={columns} paginate={false} />}
    >
      <HBarChart
        items={distribution.map((d) => ({
          label: BUCKET_LABELS[d.bucket],
          value: d.queries,
          color: BUCKET_COLORS[d.bucket],
          secondary: `${formatNumber(d.clicks)} ${t.seo.clicks.toLowerCase()}`,
        }))}
        format={(v) => `${formatNumber(v)} ${s.queriesUnit}`}
        className="py-1"
      />
    </ChartCard>
  );
}

type ShareRow = { key: string; clicks: number; impressions: number; ctr: number; position: number };

function shareColumns(nameHeader: string, labelFor: (key: string) => string): ColumnDef<ShareRow, unknown>[] {
  return [
    { accessorKey: "key", header: nameHeader, cell: ({ row }) => labelFor(row.original.key) },
    { accessorKey: "clicks", header: t.seo.clicks, meta: right, cell: ({ row }) => formatNumber(row.original.clicks) },
    { accessorKey: "ctr", header: t.seo.ctr, meta: right, cell: ({ row }) => formatPercent(row.original.ctr) },
    { accessorKey: "position", header: t.seo.position, meta: right, cell: ({ row }) => formatNumber(row.original.position, 1) },
  ];
}

function deviceLabel(key: string): string {
  const k = key.toUpperCase();
  if (k === "MOBILE") return t.seo.mobile;
  if (k === "DESKTOP") return t.seo.desktop;
  if (k === "TABLET") return s.tablet;
  return key;
}

/** (d) Perangkat & Negara — clicks share per category. */
export function DeviceCountryCards({ devices, countries }: { devices: ShareRow[]; countries: ShareRow[] }) {
  return (
    <>
      <ChartCard
        title={t.seo.devices}
        description={s.deviceDesc}
        table={<DataTable bare dense data={devices} columns={shareColumns(t.seo.devices, deviceLabel)} paginate={false} />}
      >
        <HBarChart items={devices.map((d) => ({ label: deviceLabel(d.key), value: d.clicks }))} showShare className="py-1" />
      </ChartCard>
      <ChartCard
        title={t.seo.countries}
        description={s.countryDesc}
        table={<DataTable bare dense data={countries} columns={shareColumns(t.seo.countries, countryLabel)} paginate={false} />}
      >
        <HBarChart items={countries.map((c) => ({ label: countryLabel(c.key), value: c.clicks }))} showShare maxItems={8} className="py-1" />
      </ChartCard>
    </>
  );
}
