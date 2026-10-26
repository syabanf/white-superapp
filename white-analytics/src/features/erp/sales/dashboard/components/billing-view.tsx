"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatDate, formatPercent } from "@/lib/format";
import { SEQ } from "../colors";
import { rpCompact, sum } from "../format";
import { dd } from "../strings";
import type { DashboardBilling, DashboardInvoice, DashboardQuery, DashboardTerm } from "../types";
import { ColumnChart, MonthlyChart } from "./charts";
import { DetailTable } from "./detail-table";
import { KpiRow } from "./kpi";

const b = dd.billing;

function Progress({ paid, invoiced, total }: { paid: number; invoiced: number; total: number }) {
  const pct = (v: number) => `${total ? Math.min(100, (v / total) * 100) : 0}%`;
  return (
    <span className="relative block h-2 w-24 overflow-hidden rounded-full bg-muted" aria-hidden>
      <span className="absolute inset-y-0 left-0 bg-[var(--seq-250)]" style={{ width: pct(invoiced) }} />
      <span className="absolute inset-y-0 left-0 bg-[var(--chart-1)]" style={{ width: pct(paid) }} />
    </span>
  );
}

export function BillingView({ data }: { data: DashboardBilling; query: DashboardQuery }) {
  const { period } = data;
  const series = (k: string) => data.monthly.find((m) => m.key === k) ?? { key: k, values: [], previous: [] };
  const collected = series("collected");
  const compareLabel = period.compare === "year" ? dd.compare.year : dd.compare.previous;
  return (
    <>
      <KpiRow
        kpis={data.kpis}
        keys={["invoiced", "collected", "outstanding", "overdue", "collection_rate", "days_to_pay"]}
        compare={period.compare}
        hrefs={{ invoiced: "/erp/sales/invoices", outstanding: "/erp/sales/invoices?status=terkirim", overdue: "/erp/sales/invoices?status=terkirim" }}
      />
      <div className="grid gap-4 xl:grid-cols-3">
        <ChartCard className="xl:col-span-2" title={b.monthly} description={b.monthlyHint}>
          <MonthlyChart
            months={period.months}
            yFormat={rpCompact}
            series={[
              { key: "invoiced", label: b.invoiced, values: series("invoiced").values, format: formatRupiah },
              { key: "collected", label: b.collected, values: collected.values, format: formatRupiah },
            ]}
            previous={collected.previous.length ? { label: `${b.collected} · ${compareLabel}`, values: collected.previous, format: formatRupiah } : undefined}
          />
        </ChartCard>
        <ChartCard title={b.aging} description={b.agingHint}>
          <ColumnChart
            seriesLabel={b.left}
            format={rpCompact}
            items={data.aging.map((a, i) => ({ key: a.key, label: b.bucketShort[a.key] ?? a.key, value: a.value, color: SEQ[i], href: "/erp/sales/invoices?status=terkirim" }))}
          />
        </ChartCard>
      </div>
      <div className="grid gap-4 xl:grid-cols-2">
        <ChartCard title={b.overdue} description={b.overdueHint}>
          <DetailTable<DashboardInvoice>
            rows={data.overdue}
            rowKey={(r) => r.id}
            href={(r) => `/erp/sales/pipeline/${r.dealId}`}
            empty={b.noOverdue}
            initialSort={{ key: "late", desc: true }}
            columns={[
              { key: "no", label: b.invoice, render: (r) => <span className="whitespace-nowrap">{r.invoiceNumber}</span>, sort: (r) => r.invoiceNumber },
              { key: "client", label: b.client, optional: true, render: (r) => <span className="line-clamp-1 min-w-28">{r.accountName || r.dealTitle}</span> },
              { key: "left", label: b.left, numeric: true, render: (r) => formatRupiah(r.amount - r.paid), sort: (r) => r.amount - r.paid, total: (rs) => formatRupiah(sum(rs.map((r) => r.amount - r.paid))) },
              { key: "due", label: b.due, numeric: true, optional: true, wide: true, render: (r) => (r.dueDate ? formatDate(r.dueDate) : "–") },
              { key: "late", label: b.late, numeric: true, render: (r) => dd.days(String(r.daysOverdue)), sort: (r) => r.daysOverdue },
            ]}
          />
        </ChartCard>
        <ChartCard title={b.terms} description={b.termsHint}>
          <DetailTable<DashboardTerm>
            rows={data.terms}
            rowKey={(r) => r.dealId}
            href={(r) => `/erp/sales/pipeline/${r.dealId}`}
            limit={12}
            initialSort={{ key: "contract", desc: true }}
            columns={[
              { key: "client", label: b.client, render: (r) => <span className="line-clamp-1 min-w-28">{r.accountName || r.dealTitle}</span>, sort: (r) => r.accountName },
              { key: "contract", label: b.contract, numeric: true, optional: true, render: (r) => rpCompact(r.contractValue), sort: (r) => r.contractValue, total: (rs) => rpCompact(sum(rs.map((r) => r.contractValue))) },
              { key: "left", label: b.paid, numeric: true, render: (r) => formatPercent(r.contractValue ? (r.paid / r.contractValue) * 100 : 0, 0), sort: (r) => (r.contractValue ? r.paid / r.contractValue : 0), total: (rs) => rpCompact(sum(rs.map((r) => r.paid))) },
              { key: "terms", label: b.termCount, numeric: true, render: (r) => `${r.termsInvoiced}/${r.terms}` },
              { key: "bar", label: "", render: (r) => <Progress paid={r.paid} invoiced={r.invoiced} total={r.contractValue} />, optional: true },
            ]}
          />
        </ChartCard>
      </div>
    </>
  );
}
