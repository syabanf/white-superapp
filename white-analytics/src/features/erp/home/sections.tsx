import type { ReactNode } from "react";
import Link from "next/link";
import {
  ArrowUpRight,
  Building2,
  CalendarClock,
  Clock,
  HandCoins,
  Landmark,
  Receipt,
  Target,
  Timer,
  UserCircle,
  UserPlus,
  Wallet,
  type LucideIcon,
} from "lucide-react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { FunnelChart } from "@/components/dashboard/charts/funnel-chart";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { computeDelta } from "@/lib/metrics";
import { formatNumber, formatPercent } from "@/lib/format";
import type { ResolvedPeriod } from "@/features/erp/shared/dashboard-period";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { ErpLoadError } from "@/features/erp/shared/states";
import { FinanceKpi } from "@/features/erp/finance/dashboard/charts";
import { compactIdr, kpiMap, nums } from "@/features/erp/finance/dashboard/lib";
import { ds as financeStrings } from "@/features/erp/finance/dashboard/strings";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { delta as hrDelta, monthLabel, num1, pct1, rpCompact as hrRpCompact } from "@/features/erp/hr/dashboard/format";
import { href as perfHref, int, metricDelta, monthOf, pct, score } from "@/features/erp/performance/dashboard/lib";
import type { DashMetric } from "@/features/erp/performance/dashboard/types";
import { KpiCard } from "@/features/erp/sales/dashboard/components/kpi";
import { findKpi } from "@/features/erp/sales/dashboard/format";
import { dd as salesStrings } from "@/features/erp/sales/dashboard/strings";
import { acceptanceSeries, candidatesHref, dashboardHref as recruitmentHref } from "@/features/erp/recruitment/analytics";
import { FinanceTrend, HrTrend, PerformanceTrend, SalesTrend } from "./charts";
import {
  financeArAp,
  financeSpend,
  financeSummary,
  hrDashboard,
  performanceSummary,
  recruitmentDashboard,
  salesPipeline,
  salesSummary,
} from "./data";
import { dashboardHref } from "./links";
import { hs } from "./strings";

export type HomeProps = { period: ResolvedPeriod };
type PerfProps = HomeProps & { me: boolean };

/** A module block: title row with the dashboard link, a row of tiles, one chart. */
function ModuleSection({
  icon: Icon,
  title,
  href,
  extraAction,
  tiles,
  chart,
}: {
  icon: LucideIcon;
  title: string;
  href: string;
  extraAction?: ReactNode;
  tiles: ReactNode;
  chart: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3" aria-label={title}>
      <SectionTitle icon={Icon} title={title}>
        {extraAction}
        <Button size="sm" variant="ghost" asChild>
          <Link href={href}>
            {hs.openDashboard} <ArrowUpRight className="size-3.5" />
          </Link>
        </Button>
      </SectionTitle>
      <KpiGrid cols={4}>{tiles}</KpiGrid>
      {chart}
    </section>
  );
}

function SectionTitle({ icon: Icon, title, children }: { icon: LucideIcon; title: string; children?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2">
      <h2 className="flex items-center gap-2 text-base font-semibold tracking-[-0.02em]">
        <span className="flex size-7 items-center justify-center rounded-xl bg-brand text-white">
          <Icon className="size-4" />
        </span>
        {title}
      </h2>
      {children ? <div className="flex items-center gap-1">{children}</div> : null}
    </div>
  );
}

function SectionError({ icon, title, error }: { icon: LucideIcon; title: string; error: { code: string; message: string } }) {
  return (
    <section className="flex flex-col gap-3" aria-label={title}>
      <SectionTitle icon={icon} title={title} />
      <ErpLoadError error={error} compact />
    </section>
  );
}

/** Mirrors ModuleSection while its data streams in. */
export function ModuleSkeleton() {
  return (
    <div className="flex flex-col gap-3" aria-busy="true">
      <Skeleton className="h-7 w-40" />
      <div className="grid grid-cols-2 gap-3 sm:gap-5 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-32 rounded-xl" />
        ))}
      </div>
      <Skeleton className="h-80 rounded-xl" />
    </div>
  );
}

export function AttentionSkeleton() {
  return (
    <div className="flex flex-col gap-3" aria-busy="true">
      <Skeleton className="h-6 w-36" />
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-[repeat(auto-fit,minmax(8rem,1fr))]">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    </div>
  );
}

// ─── Keuangan ───────────────────────────────────────────────

export async function FinanceSection({ period }: HomeProps) {
  const res = await financeSummary(period.from, period.to, period.compare);
  if (res.error) return <SectionError icon={Landmark} title={hs.finance.title} error={res.error} />;
  const d = res.data;
  const k = kpiMap(d.kpis);
  const dash = dashboardHref("/erp/finance", period);
  const series = (key: string) => d.pnl.find((s) => s.key === key);
  const rev = series("revenue");
  return (
    <ModuleSection
      icon={Landmark}
      title={hs.finance.title}
      href={dash}
      tiles={
        <>
          <FinanceKpi kpi={k.revenue} href={dash} />
          <FinanceKpi kpi={k.expense} lowerIsBetter href={dashboardHref("/erp/finance", period, { tab: "profit" })} />
          <FinanceKpi kpi={k.net_income} href={dashboardHref("/erp/finance", period, { tab: "profit" })} />
          <FinanceKpi kpi={k.cash} href={dashboardHref("/erp/finance", period, { tab: "cash" })} />
        </>
      }
      chart={
        <ChartCard title={hs.finance.chart} description={hs.finance.chartHint}>
          <FinanceTrend
            months={d.window.months}
            compareMonths={d.window.compareMonths}
            revenue={nums(rev?.values ?? [])}
            revenueCompare={nums(rev?.compareValues ?? [])}
            expense={nums(series("expense")?.values ?? [])}
            net={nums(series("net_income")?.values ?? [])}
            labels={{ revenue: financeStrings.kpi.revenue!, expense: financeStrings.kpi.expense!, net: financeStrings.kpi.net_income! }}
          />
        </ChartCard>
      }
    />
  );
}

// ─── Penjualan ──────────────────────────────────────────────

export async function SalesSection({ period }: HomeProps) {
  const res = await salesSummary(period.from, period.to, period.compare);
  if (res.error) return <SectionError icon={HandCoins} title={hs.sales.title} error={res.error} />;
  const d = res.data;
  const dash = dashboardHref("/erp/sales", period);
  const won = d.trend.find((t) => t.key === "won_value");
  const target = d.trend.find((t) => t.key === "target");
  const tile = (key: string, tab?: string) => (
    <KpiCard kpi={findKpi(d.kpis, key)} compare={d.period.compare} href={dashboardHref("/erp/sales", period, { tab })} />
  );
  return (
    <ModuleSection
      icon={HandCoins}
      title={hs.sales.title}
      href={dash}
      tiles={
        <>
          {tile("won_value", "penjualan")}
          {tile("pipeline_value", "pipeline")}
          {tile("win_rate", "penjualan")}
          {tile("avg_deal", "penjualan")}
        </>
      }
      chart={
        <ChartCard title={hs.sales.chart} description={hs.sales.chartHint}>
          <SalesTrend
            months={d.period.months}
            won={won?.values ?? []}
            target={target?.values ?? []}
            previous={won?.previous ?? []}
            labels={{
              won: salesStrings.summary.won,
              target: salesStrings.summary.target,
              previous: d.period.compare === "year" ? salesStrings.compare.year : salesStrings.compare.previous,
            }}
          />
        </ChartCard>
      }
    />
  );
}

// ─── SDM ────────────────────────────────────────────────────

export async function HrSection({ period }: HomeProps) {
  const res = await hrDashboard(period.from, period.to, period.compare);
  if (res.error) return <SectionError icon={Building2} title={hs.hr.title} error={res.error} />;
  const d = res.data;
  const cur = d.current;
  const prev = d.previous;
  const m = d.months;
  const dash = dashboardHref("/erp/hr", period);
  const payroll = parseMoney(cur.payrollEmployerCost);
  const prevPayroll = parseMoney(prev.payrollEmployerCost);
  return (
    <ModuleSection
      icon={Building2}
      title={hs.hr.title}
      href={dash}
      tiles={
        <>
          <KpiTile
            label={hs.hr.headcount}
            hint={hs.hr.headcountHint}
            value={formatNumber(cur.headcount)}
            delta={hrDelta(cur.headcount, prev.headcount)}
            previous={formatNumber(prev.headcount)}
            spark={m.map((b) => b.headcount)}
            href={hrLinks.employees()}
          />
          <KpiTile
            label={hs.hr.attendance}
            hint={hs.hr.attendanceHint}
            value={pct1(cur.attendanceRate)}
            delta={hrDelta(cur.attendanceRate, prev.attendanceRate)}
            previous={pct1(prev.attendanceRate)}
            spark={m.map((b) => b.attendanceRate)}
            href={hrLinks.attendance({ from: d.from, to: d.to })}
          />
          <KpiTile
            label={hs.hr.late}
            hint={hs.hr.lateHint}
            value={pct1(cur.lateRate)}
            delta={hrDelta(cur.lateRate, prev.lateRate)}
            lowerIsBetter
            previous={pct1(prev.lateRate)}
            spark={m.map((b) => b.lateRate)}
            href={hrLinks.attendance({ from: d.from, to: d.to, late: true })}
          />
          {d.payrollVisible ? (
            <KpiTile
              label={hs.hr.payroll}
              hint={`${hs.hr.payrollHint} ${formatRupiah(payroll)}.`}
              icon={<Wallet />}
              value={payroll ? hrRpCompact(payroll) : "–"}
              delta={payroll ? hrDelta(payroll, prevPayroll) : null}
              previous={prevPayroll ? hrRpCompact(prevPayroll) : undefined}
              lowerIsBetter
              spark={m.map((b) => parseMoney(b.payrollGross) + parseMoney(b.payrollBpjsEmployer))}
              href={hrLinks.payroll(d.to.slice(0, 4))}
            />
          ) : (
            <KpiTile
              label={hs.hr.overtime}
              hint={hs.hr.overtimeHint}
              icon={<Timer />}
              value={`${num1(cur.overtimeHours)} ${hs.hr.hours}`}
              delta={hrDelta(cur.overtimeHours, prev.overtimeHours)}
              lowerIsBetter
              previous={`${num1(prev.overtimeHours)} ${hs.hr.hours}`}
              spark={m.map((b) => b.overtimeHours)}
              href={hrLinks.overtime(d.to.slice(0, 7))}
            />
          )}
        </>
      }
      chart={
        <ChartCard title={hs.hr.chart} description={hs.hr.chartHint}>
          <HrTrend
            to={d.to}
            months={m.map((b) => ({
              date: b.date,
              label: monthLabel(b.date),
              headcount: b.headcount,
              attendanceRate: b.attendanceRecords ? b.attendanceRate : null,
            }))}
          />
        </ChartCard>
      }
    />
  );
}

// ─── Kinerja ────────────────────────────────────────────────

const MAX_LINES = 8;

function metricTile(label: string, hint: string, metric: DashMetric, format: (v: number | undefined) => string, href: string, lowerIsBetter = false) {
  return (
    <KpiTile
      label={label}
      hint={hint}
      value={format(metric.value)}
      delta={metricDelta(metric)}
      lowerIsBetter={lowerIsBetter}
      previous={metric.previous != null ? format(metric.previous) : undefined}
      spark={metric.spark}
      href={href}
    />
  );
}

export async function PerformanceSection({ period, me }: PerfProps) {
  const title = me ? hs.me.title : hs.performance.title;
  const res = await performanceSummary(period.from, period.to, period.compare, me ? "me" : "");
  if (res.error) return <SectionError icon={me ? UserCircle : Target} title={title} error={res.error} />;
  const d = res.data;
  const scope = d.window.scope === "me" ? "me" : "all";
  const month = monthOf(d.window.to);
  const dash = dashboardHref("/erp/performance", period, { scope: me ? "me" : undefined });
  const tasks = me ? "/erp/me/tasks" : perfHref("/erp/performance/tasks", { tab: "monitor", month });
  const ids = [...new Set(d.departmentTrend.map((x) => x.departmentId))].slice(0, MAX_LINES);
  const names = Object.fromEntries(d.departmentTrend.map((x) => [x.departmentId, x.departmentName]));
  const rows = d.months.map((mo) => {
    const row: Record<string, string | number | null> = { month: mo };
    for (const x of d.departmentTrend) if (x.period === mo) row[x.departmentId] = x.score ?? null;
    return row;
  });
  const p = hs.performance;
  return (
    <ModuleSection
      icon={me ? UserCircle : Target}
      title={title}
      href={dash}
      extraAction={
        me ? (
          <Button size="sm" variant="ghost" asChild>
            <Link href="/erp/me">{hs.me.open}</Link>
          </Button>
        ) : undefined
      }
      tiles={
        <>
          {metricTile(p.kpi, p.kpiHint, d.kpiScore, score, perfHref("/erp/performance/kpi", { month, scope }))}
          {metricTile(p.completion, p.completionHint, d.taskCompletion, pct, tasks)}
          {metricTile(p.onTime, p.onTimeHint, d.onTimeRate, pct, tasks)}
          {metricTile(p.overdue, p.overdueHint, d.overdueTasks, int, tasks, true)}
        </>
      }
      chart={
        <ChartCard title={me ? p.chartMe : p.chart} description={p.chartHint}>
          <PerformanceTrend
            months={d.months}
            lines={ids.map((id) => ({ id, name: names[id] ?? id }))}
            rows={rows}
            scope={scope}
          />
        </ChartCard>
      }
    />
  );
}

// ─── Rekrutmen ──────────────────────────────────────────────

export async function RecruitmentSection({ period }: HomeProps) {
  const res = await recruitmentDashboard(period.from, period.to, period.compare);
  if (res.error) return <SectionError icon={UserPlus} title={hs.recruitment.title} error={res.error} />;
  const d = res.data;
  const k = d.kpis;
  const prev = d.previousKpis;
  const t = d.trend;
  const ctx = { from: period.from, to: period.to };
  const r = hs.recruitment;
  const days = (v: number | null | undefined) => (v == null ? "–" : `${formatNumber(v, 1)} ${r.days}`);
  const rate = (v: number | null | undefined) => (v == null ? "–" : formatPercent(v, 0));
  const change = (a: number | null | undefined, b: number | null | undefined) => (a != null && b ? computeDelta(a, b) : null);
  // A rate's change reads in points, as the finance tiles do.
  const pointsDelta = (a: number | null | undefined, b: number | null | undefined) => {
    const c = a != null && b != null ? computeDelta(a, b) : null;
    return c ? { ...c, pct: c.abs } : null;
  };
  return (
    <ModuleSection
      icon={UserPlus}
      title={r.title}
      href={dashboardHref("/erp/hr/recruitment", period)}
      tiles={
        <>
          <KpiTile
            label={r.applicants}
            hint={r.applicantsHint}
            value={formatNumber(k.applicants)}
            delta={change(k.applicants, prev.applicants)}
            previous={formatNumber(prev.applicants)}
            spark={t.map((x) => x.applications)}
            href={candidatesHref(ctx)}
          />
          <KpiTile
            label={r.hires}
            hint={r.hiresHint}
            value={formatNumber(k.hires)}
            delta={change(k.hires, prev.hires)}
            previous={formatNumber(prev.hires)}
            spark={t.map((x) => x.hires)}
            href={candidatesHref(ctx, { status: "hired" })}
          />
          <KpiTile
            label={r.daysToHire}
            hint={r.daysToHireHint}
            value={days(k.avgDaysToHire)}
            delta={change(k.avgDaysToHire, prev.avgDaysToHire)}
            lowerIsBetter
            previous={prev.avgDaysToHire != null ? days(prev.avgDaysToHire) : undefined}
            spark={t.map((x) => x.avgDaysToHire ?? 0)}
            href={candidatesHref(ctx, { status: "hired" })}
          />
          <KpiTile
            label={r.acceptance}
            hint={r.acceptanceHint}
            value={rate(k.offerAcceptanceRate)}
            delta={pointsDelta(k.offerAcceptanceRate, prev.offerAcceptanceRate)}
            previous={prev.offerAcceptanceRate != null ? rate(prev.offerAcceptanceRate) : undefined}
            spark={acceptanceSeries(t)}
            href={recruitmentHref({ ...ctx, compare: period.compare }, { tab: "offer" })}
          />
        </>
      }
      chart={
        <ChartCard title={r.chart} description={r.chartHint}>
          <FunnelChart
            exitedLabel={r.exited}
            steps={d.funnel.map((f) => ({
              key: f.stage,
              label: f.label,
              value: f.reached,
              exited: f.exited,
              href: candidatesHref(ctx, { status: f.stage }),
            }))}
          />
        </ChartCard>
      }
    />
  );
}

// ─── Perlu perhatian ────────────────────────────────────────

type AttentionItem = { key: string; label: string; count: number; caption: string; href: string; icon: LucideIcon };

export type AttentionAccess = { finance: boolean; hr: boolean; performance: boolean; perfMe: boolean; sales: boolean; recruitment: boolean };

/** Counts that wait on someone, across the modules the user can open; a failed module drops out. */
export async function AttentionStrip({ period, can }: HomeProps & { can: AttentionAccess }) {
  const { from, to, compare } = period;
  const [spend, arap, hr, perf, pipeline, rec] = await Promise.all([
    can.finance ? financeSpend(from, to, compare) : null,
    can.finance ? financeArAp(from, to, compare) : null,
    can.hr ? hrDashboard(from, to, compare) : null,
    can.performance ? performanceSummary(from, to, compare, can.perfMe ? "me" : "") : null,
    can.sales ? salesPipeline(from, to, compare) : null,
    can.recruitment ? recruitmentDashboard(from, to, compare) : null,
  ]);
  const a = hs.attention;
  const items: AttentionItem[] = [];

  if (spend?.data) {
    const pending = kpiMap(spend.data.kpis).reimburse_pending?.value ?? "";
    items.push({
      key: "reimburse",
      label: a.reimburse,
      count: spend.data.pending.length,
      caption: a.amount(compactIdr(parseMoney(pending))),
      href: "/erp/finance/reimbursements",
      icon: Receipt,
    });
  }
  if (hr?.data) {
    const pending = hr.data.pending;
    const kinds = [
      { kind: "leave", label: a.leave, href: hrLinks.leaveApprovals(), icon: CalendarClock },
      { kind: "overtime", label: a.overtime, href: hrLinks.overtimeApprovals(), icon: Timer },
    ] as const;
    for (const k of kinds) {
      const rows = pending.filter((p) => p.kind === k.kind);
      const oldest = rows.reduce((max, p) => Math.max(max, p.ageDays), 0);
      items.push({ key: k.kind, label: k.label, count: rows.length, caption: rows.length ? a.olderThan(oldest) : a.none, href: k.href, icon: k.icon });
    }
  }
  if (perf?.data) {
    items.push({
      key: "tasks",
      label: can.perfMe ? a.myOverdueTasks : a.overdueTasks,
      count: perf.data.overdueTasks.value ?? 0,
      caption: a.inPeriod,
      href: can.perfMe ? "/erp/me/tasks" : perfHref("/erp/performance/tasks", { tab: "monitor", month: monthOf(perf.data.window.to) }),
      icon: Clock,
    });
  }
  if (pipeline?.data) {
    items.push({
      key: "deals",
      label: a.stuckDeals,
      count: pipeline.data.stuck.length,
      caption: a.pastThreshold,
      href: dashboardHref("/erp/sales", period, { tab: "pipeline" }),
      icon: HandCoins,
    });
  }
  if (arap?.data) {
    const overdue = arap.data.aging.filter((r) => r.kind === "AR" && r.bucket !== "current");
    const amount = kpiMap(arap.data.kpis).overdue_receivable?.value ?? "";
    items.push({
      key: "ar",
      label: a.overdueAr,
      count: overdue.reduce((s, r) => s + r.count, 0),
      caption: a.amount(compactIdr(parseMoney(amount))),
      href: "/erp/finance/receivables?tab=aging",
      icon: Wallet,
    });
  }
  if (rec?.data) {
    items.push({
      key: "candidates",
      label: a.stuckCandidates,
      count: rec.data.stuck.length,
      caption: a.olderThan(rec.data.stuckAfterDays),
      href: recruitmentHref({ from, to, compare }, { tab: "pipeline" }),
      icon: UserPlus,
    });
  }
  if (items.length === 0) return null;

  return (
    <section className="flex flex-col gap-3" aria-label={a.title}>
      <h2 className="text-base font-semibold tracking-[-0.02em]">{a.title}</h2>
      {/* A swipeable row on phones, a grid from sm up. */}
      <ul className="-mx-4 flex snap-x gap-3 overflow-x-auto px-4 pb-1 scrollbar-thin sm:mx-0 sm:grid sm:grid-cols-[repeat(auto-fit,minmax(8rem,1fr))] sm:overflow-visible sm:px-0 sm:pb-0">
        {items.map((it) => (
          <li key={it.key} className="w-40 shrink-0 snap-start sm:w-auto sm:min-w-0">
            <Card className="lift h-full gap-0 py-0 hover:ring-brand/25">
              <Link
                href={it.href}
                className="flex h-full min-w-0 flex-col gap-2 rounded-xl p-4 outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span className="flex items-start justify-between gap-2 text-muted-foreground">
                  <span className="label-mono line-clamp-2">{it.label}</span>
                  <it.icon className="size-3.5 shrink-0" />
                </span>
                <span className="text-[26px] leading-none font-bold tracking-[-0.035em] tabular">{formatNumber(it.count)}</span>
                <StatusBadge kind={it.count > 0 ? "warning" : "good"} className="w-fit">
                  {it.count > 0 ? a.act : a.clear}
                </StatusBadge>
                <span className="truncate text-xs text-muted-foreground">{it.caption}</span>
              </Link>
            </Card>
          </li>
        ))}
      </ul>
    </section>
  );
}
