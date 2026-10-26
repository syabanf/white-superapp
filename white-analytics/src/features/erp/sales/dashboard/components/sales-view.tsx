"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatNumber, formatPercent } from "@/lib/format";
import { QUOTATION_STATUS, lbl } from "../../strings";
import { lineColor, lineLabel, sourceLabel } from "../colors";
import { rpCompact, sum, topN } from "../format";
import { drill } from "../params";
import { dd } from "../strings";
import type { DashboardQuery, DashboardSales, DashboardSlice } from "../types";
import { Donut, MonthlyChart } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const s = dd.sales;
const QUOTE_ORDER = ["diterima", "terkirim", "ditolak", "superseded", "draft"];

export function SalesView({ data, query }: { data: DashboardSales; query: DashboardQuery }) {
  const { period } = data;
  const series = (k: string) => data.monthly.find((m) => m.key === k) ?? { key: k, values: [], previous: [] };
  // A month without closed deals has no win rate; leave a gap instead of 0%.
  const rate = (pick: "values" | "previous") =>
    series("win_rate")[pick].map((v, i) => (series("won_count")[pick][i] || series("lost_count")[pick][i] ? v : null));
  const prevRate = rate("previous");
  const compareLabel = period.compare === "year" ? dd.compare.year : dd.compare.previous;
  const pipelineHref = drill("/erp/sales/pipeline", { pipeline: query.pipelineId, owner: query.ownerUserId });
  const totalWon = sum(data.serviceLines.map((x) => x.value));
  const quotes = [...data.quoteStatuses].sort((a, b) => QUOTE_ORDER.indexOf(a.key) - QUOTE_ORDER.indexOf(b.key));
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["won_value", "won_deals", "win_rate", "avg_discount", "quote_acceptance", "time_to_accept_days"]}
        compare={period.compare}
        hrefs={{ quote_acceptance: "/erp/sales/quotations?status=diterima" }}
      />
      <div className="grid gap-4 xl:grid-cols-2">
        <ChartCard title={s.wonLost} description={s.wonLostHint}>
          <MonthlyChart
            months={period.months}
            yFormat={(v) => formatNumber(v)}
            series={[
              { key: "won", label: s.won, values: series("won_count").values, format: (v) => dd.deals(v) },
              { key: "lost", label: s.lost, values: series("lost_count").values, format: (v) => dd.deals(v) },
            ]}
          />
        </ChartCard>
        <ChartCard title={s.winRate} description={s.winRateHint}>
          <MonthlyChart
            months={period.months}
            yFormat={(v) => formatPercent(v, 0)}
            yMax={100}
            series={[{ key: "rate", label: dd.kpi.win_rate![0], type: "line", values: rate("values"), format: (v) => formatPercent(v, 1) }]}
            previous={prevRate.length ? { label: compareLabel, values: prevRate, format: (v) => formatPercent(v, 1) } : undefined}
          />
        </ChartCard>
        <ChartCard title={s.byLine} description={s.byLineHint}>
          <HBarChart
            items={data.serviceLines.map((x) => ({ label: lineLabel(x.key), value: x.value, secondary: dd.deals(x.count), color: lineColor(x.key), href: drill("/erp/sales/pipeline", { pipeline: query.pipelineId, owner: query.ownerUserId }) }))}
            format={rpCompact}
            emptyLabel={dd.empty}
          />
        </ChartCard>
        <ChartCard title={s.byIndustry}>
          <HBarChart items={topN(data.industries, 8).map((x) => ({ label: x.label, value: x.value, secondary: dd.deals(x.count), muted: x.key === "__others__" }))} format={rpCompact} emptyLabel={dd.empty} />
        </ChartCard>
        <ChartCard title={s.lostReasons} description={s.lostReasonsHint}>
          <HBarChart
            items={data.lostReasons.map((x) => ({ label: x.label, value: x.count, secondary: rpCompact(x.value), color: slotColor(1), href: pipelineHref }))}
            format={(v) => dd.deals(v)}
            emptyLabel={dd.empty}
          />
        </ChartCard>
        <ChartCard title={s.quotes} description={s.quotesHint}>
          <Donut
            centerLabel={s.quoteUnit}
            items={quotes.map((q) => ({
              key: q.key,
              label: lbl(QUOTATION_STATUS, q.key),
              value: q.count,
              color: slotColor(QUOTE_ORDER.indexOf(q.key)),
              href: `/erp/sales/quotations?status=${q.key}`,
            }))}
          />
        </ChartCard>
      </div>
      <div className="grid gap-4 xl:grid-cols-3">
        <ChartCard className="xl:col-span-2" title={s.lineTable}>
          <DetailTable<DashboardSlice>
            rows={data.serviceLines}
            rowKey={(r) => r.key}
            initialSort={{ key: "value", desc: true }}
            columns={[
              { key: "label", label: s.line, render: (r) => lineLabel(r.key), sort: (r) => lineLabel(r.key) },
              { key: "count", label: dd.summary.deals, numeric: true, render: (r) => formatNumber(r.count), sort: (r) => r.count, total: (rs) => formatNumber(sum(rs.map((r) => r.count))) },
              { key: "value", label: dd.summary.value, numeric: true, render: (r) => formatRupiah(r.value), sort: (r) => r.value, total: (rs) => formatRupiah(sum(rs.map((r) => r.value))) },
              { key: "avg", label: s.avg, numeric: true, render: (r) => formatRupiah(r.count ? r.value / r.count : 0), sort: (r) => (r.count ? r.value / r.count : 0), optional: true },
              { key: "share", label: dd.summary.share, numeric: true, render: (r) => formatPercent(totalWon ? (r.value / totalWon) * 100 : 0, 0), optional: true },
            ]}
          />
        </ChartCard>
        <ChartCard title={s.bySource}>
          <HBarChart
            items={data.sources.map((x) => ({ label: sourceLabel(x.key), value: x.value, secondary: dd.deals(x.count), href: drill("/erp/sales/leads", { source: x.key, owner: query.ownerUserId }) }))}
            format={rpCompact}
            emptyLabel={dd.empty}
          />
        </ChartCard>
      </div>
    </>
  );
}
