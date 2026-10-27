"use client";

import Link from "next/link";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Button } from "@/components/ui/button";
import { parseMoney } from "@/features/erp/shared/money";
import { th } from "@/features/erp/hr/strings";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { delta, monthLabel, num0, rp, rpCompact } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";
import { ChartEmpty, ComboChart, RangeBars, Waterfall } from "./charts";
import { SortTable } from "./sort-table";
import { useHrDrill, useHrScope } from "./drill";
import { deptPalette, Grid, type TabProps } from "./shared";

const c = thd.charts;
const k = thd.kpi;
const s = thd.series;

export function PayrollTab({ data }: TabProps) {
  const drill = useHrDrill();
  const { dept } = useHrScope();
  const year = data.to.slice(0, 4);
  const runs = hrLinks.payroll(year);
  const cur = data.current;
  const prev = data.previous;
  const depts = deptPalette(data);
  const m = (v: string) => parseMoney(v);

  const months = data.months.map((b) => {
    const gross = m(b.payrollGross);
    const net = m(b.payrollNet);
    const pph = m(b.payrollPph21);
    const bpjs = m(b.payrollBpjsEmployee);
    return { label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date, net, pph, bpjs, other: Math.max(0, gross - net - pph - bpjs) };
  });

  const sum = (pick: (b: (typeof data.buckets)[number]) => string) => data.buckets.reduce((a, b) => a + m(pick(b)), 0);
  const gross = sum((b) => b.payrollGross);
  const net = sum((b) => b.payrollNet);
  const pph = sum((b) => b.payrollPph21);
  const bpjs = sum((b) => b.payrollBpjsEmployee);
  const employer = sum((b) => b.payrollBpjsEmployer);

  return (
    <div className="page-stack">
      <StatStrip
        items={[
          { label: k.gross, value: rpCompact(cur.payrollGross), delta: delta(m(cur.payrollGross), m(prev.payrollGross)), lowerIsBetter: true, hint: k.exact(rp(cur.payrollGross)), href: runs },
          { label: k.net, value: rpCompact(cur.payrollNet), delta: delta(m(cur.payrollNet), m(prev.payrollNet)), lowerIsBetter: true, hint: k.exact(rp(cur.payrollNet)), href: runs },
          {
            label: k.employerCost,
            value: rpCompact(cur.payrollEmployerCost),
            delta: delta(m(cur.payrollEmployerCost), m(prev.payrollEmployerCost)),
            lowerIsBetter: true,
            hint: `${k.payrollHint} ${k.exact(rp(cur.payrollEmployerCost))}`,
            href: runs,
          },
          {
            label: k.paidEmployees,
            value: num0(cur.payrollEmployees),
            delta: delta(cur.payrollEmployees, prev.payrollEmployees),
            href: hrLinks.salaries({ dept }),
          },
          { label: k.loans, value: rpCompact(data.loansOutstanding), hint: k.exact(rp(data.loansOutstanding)), href: hrLinks.loans() },
        ]}
      />

      <Grid>
        <ChartCard title={c.payrollTrend} description={`${c.payrollTrendQ} ${thd.monthsWindow}.`} className="lg:col-span-2">
          <ComboChart
            data={months}
            stacked
            series={[
              { key: "net", label: s.net, color: slotColor(0), format: rp },
              { key: "pph", label: s.pph21, color: slotColor(1), format: rp },
              { key: "bpjs", label: s.bpjsEmployee, color: slotColor(2), format: rp },
              { key: "other", label: s.otherDeductions, color: slotColor(3), format: rp },
            ]}
            yFormat={(v) => rpCompact(v)}
            onSelect={(r) => drill("pay", { month: String(r.date).slice(0, 7) })}
            summary={c.payrollTrend}
            height={280}
          />
        </ChartCard>
        <ChartCard title={c.payrollDept} description={c.payrollDeptQ}>
          <HBarChart
            items={[...data.departments]
              .filter((d) => m(d.payrollEmployerCost) > 0)
              .sort((a, b) => m(b.payrollEmployerCost) - m(a.payrollEmployerCost))
              .map((d) => ({ label: depts.name(d.departmentId), value: m(d.payrollEmployerCost), key: d.departmentId }))}
            onSelect={(item) => item.key && drill("pay", { month: data.to.slice(0, 7), dept: item.key })}
            format={(v) => rpCompact(v)}
            showShare
            emptyLabel={thd.emptyPayroll}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.waterfall} description={c.waterfallQ} className="lg:col-span-2">
          {gross ? (
            <Waterfall
              steps={[
                { label: "Biaya PT", value: gross + employer, kind: "total" },
                { label: "Porsi PT", value: employer, kind: "minus" },
                { label: "Bruto", value: gross, kind: "total" },
                { label: "PPh 21", value: pph, kind: "minus" },
                { label: "BPJS", value: bpjs, kind: "minus" },
                { label: "Lain", value: Math.max(0, gross - net - pph - bpjs), kind: "minus" },
                { label: "Bersih", value: net, kind: "total" },
              ]}
              format={(v) => rpCompact(v)}
              summary={`${c.waterfall}: ${rp(gross)} ke ${rp(net)}`}
              onSelect={() => drill("pay", { month: data.to.slice(0, 7) })}
              height={280}
            />
          ) : (
            <ChartEmpty height={280}>{thd.emptyPayroll}</ChartEmpty>
          )}
        </ChartCard>
        <ChartCard title={c.salaryLevel} description={c.salaryLevelQ}>
          <RangeBars
            items={data.salaryByLevel.map((l) => ({
              key: l.level,
              label: th.masters.levels[l.level] ?? (l.level || "–"),
              min: m(l.minGross),
              max: m(l.maxGross),
              avg: m(l.avgGross),
              count: l.employees,
              href: l.level ? hrLinks.employees({ level: l.level, dept }) : undefined,
            }))}
            format={(v) => rpCompact(v)}
          />
        </ChartCard>
      </Grid>

      <ChartCard
        title={c.loans}
        description={c.loansQ}
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link href={hrLinks.loans()}>{thd.openList}</Link>
          </Button>
        }
      >
        <SortTable
          rows={data.loans}
          rowKey={(r) => r.loanId}
          rowHref={(r) => hrLinks.employee(r.employeeId)}
          initialSort={{ key: "remaining", desc: true }}
          empty={thd.emptyLoans}
          columns={[
            { key: "name", label: thd.cols.employee, value: (r) => r.fullName },
            { key: "type", label: thd.cols.type, value: (r) => r.loanType, cell: (r) => th.loans.types[r.loanType] ?? r.loanType },
            { key: "principal", label: thd.cols.principal, numeric: true, value: (r) => m(r.principalAmount), cell: (r) => rp(r.principalAmount), total: rp(data.loans.reduce((a, r) => a + m(r.principalAmount), 0)) },
            { key: "monthly", label: thd.cols.monthly, numeric: true, value: (r) => m(r.monthlyInstallment), cell: (r) => rp(r.monthlyInstallment), total: rp(data.loans.reduce((a, r) => a + m(r.monthlyInstallment), 0)) },
            { key: "remaining", label: thd.cols.remaining, numeric: true, value: (r) => m(r.remainingBalance), cell: (r) => rp(r.remainingBalance), total: rp(data.loansOutstanding) },
          ]}
        />
      </ChartCard>
    </div>
  );
}
