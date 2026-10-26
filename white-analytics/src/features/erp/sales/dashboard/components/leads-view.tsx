"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { formatNumber, formatPercent } from "@/lib/format";
import { SEQ, SOURCE_ORDER, sourceColor, sourceGroup, sourceLabel } from "../colors";
import { sum } from "../format";
import { drill } from "../params";
import { dd } from "../strings";
import type { DashboardLeads, DashboardQuery, DashboardSlice } from "../types";
import { ColumnChart, Donut, MonthlyChart, StackedBars } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const l = dd.leads;

export function LeadsView({ data, query }: { data: DashboardLeads; query: DashboardQuery }) {
  const { period } = data;
  const leadsHref = (source?: string) => drill("/erp/sales/leads", { source, owner: query.ownerUserId });
  const groups = [...SOURCE_ORDER, "lainnya"];
  const stacked = groups.map((g) => ({
    key: g,
    label: g === "lainnya" ? dd.others : sourceLabel(g),
    color: sourceColor(g),
    values: period.months.map((m) => sum(data.monthlySources.filter((x) => x.key === m && sourceGroup(x.group) === g).map((x) => x.count))),
    format: (v: number) => formatNumber(v),
  }));
  const response = (k: string) => data.response.find((r) => r.key === k)?.values ?? [];
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["leads", "conversion_rate", "converted", "first_response_minutes", "sla_breaches", "submissions"]}
        compare={period.compare}
        hrefs={{ leads: leadsHref(), sla_breaches: "/erp/sales/chatbot", submissions: "/erp/sales/forms" }}
      />
      <ChartCard title={l.bySourceMonth} description={l.bySourceMonthHint}>
        <MonthlyChart months={period.months} stacked yFormat={(v) => formatNumber(v)} series={stacked.filter((s) => s.values.some(Boolean))} />
      </ChartCard>
      <div className="grid gap-4 xl:grid-cols-2">
        <ChartCard title={l.conversion} description={l.conversionHint}>
          <StackedBars
            rows={data.sources.map((s) => ({
              key: s.key,
              label: `${sourceLabel(s.key)} · ${formatPercent(s.count ? (s.value / s.count) * 100 : 0, 0)}`,
              href: leadsHref(s.key),
              segments: [
                { key: "won", value: s.extra },
                { key: "deal", value: s.value - s.extra },
                { key: "lead", value: s.count - s.value },
              ],
            }))}
            groups={[
              { key: "won", label: l.toWon, color: slotColor(0) },
              { key: "deal", label: l.toDeal, color: "var(--seq-250)" },
              { key: "lead", label: l.leads, color: "var(--chart-emphasis-muted)" },
            ]}
            format={(v) => formatNumber(v)}
          />
        </ChartCard>
        <ChartCard title={l.scores} description={l.scoresHint}>
          <ColumnChart items={data.scores.map((s, i) => ({ key: s.key, label: s.label, value: s.count, color: SEQ[i] }))} seriesLabel={l.leads} />
        </ChartCard>
        <ChartCard title={l.response} description={l.responseHint}>
          <MonthlyChart
            months={period.months}
            yFormat={(v) => dd.minutes(formatNumber(v))}
            series={[
              { key: "minutes", label: dd.kpi.first_response_minutes![0], type: "line", values: response("response_minutes"), format: (v) => dd.minutes(formatNumber(v, 1)) },
            ]}
          />
        </ChartCard>
        <ChartCard title={l.channels}>
          <Donut
            centerLabel={l.convUnit}
            items={data.channels.map((c, i) => ({ key: c.key, label: l.channel[c.key] ?? c.key, value: c.count, color: slotColor(i), href: `/erp/sales/chatbot?channel=${c.key}` }))}
          />
          <ul className="mt-4 space-y-1 text-xs text-muted-foreground">
            {data.channels.map((c) => (
              <li key={c.key} className="flex justify-between gap-2">
                <span>{l.channel[c.key] ?? c.key}</span>
                <span className="tabular">
                  {dd.minutes(formatNumber(c.value, 1))} · {formatNumber(c.extra)} {l.breaches.toLowerCase()}
                </span>
              </li>
            ))}
          </ul>
        </ChartCard>
      </div>
      <div className="grid gap-4 xl:grid-cols-3">
        <ChartCard className="xl:col-span-2" title={l.sourceTable}>
          <DetailTable<DashboardSlice>
            rows={data.sources}
            rowKey={(r) => r.key}
            href={(r) => leadsHref(r.key)}
            initialSort={{ key: "count", desc: true }}
            columns={[
              { key: "src", label: l.source, render: (r) => sourceLabel(r.key), sort: (r) => sourceLabel(r.key) },
              { key: "count", label: l.leads, numeric: true, render: (r) => formatNumber(r.count), sort: (r) => r.count, total: (rs) => formatNumber(sum(rs.map((r) => r.count))) },
              { key: "deal", label: l.toDeal, numeric: true, render: (r) => formatNumber(r.value), sort: (r) => r.value, total: (rs) => formatNumber(sum(rs.map((r) => r.value))) },
              { key: "won", label: l.toWon, numeric: true, optional: true, render: (r) => formatNumber(r.extra), sort: (r) => r.extra, total: (rs) => formatNumber(sum(rs.map((r) => r.extra))) },
              {
                key: "rate",
                label: l.rate,
                numeric: true,
                render: (r) => formatPercent(r.count ? (r.value / r.count) * 100 : 0, 0),
                sort: (r) => (r.count ? r.value / r.count : 0),
                total: (rs) => formatPercent(sum(rs.map((r) => r.count)) ? (sum(rs.map((r) => r.value)) / sum(rs.map((r) => r.count))) * 100 : 0, 0),
              },
            ]}
          />
        </ChartCard>
        <ChartCard title={l.forms} description={l.formsHint}>
          <HBarChart
            items={data.forms.map((f) => ({ label: f.label, value: f.count, secondary: `${formatNumber(f.extra)} ${l.rejected.toLowerCase()}`, href: `/erp/sales/forms?form=${f.key}` }))}
            format={(v) => formatNumber(v)}
            emptyLabel={dd.empty}
          />
        </ChartCard>
      </div>
    </>
  );
}
