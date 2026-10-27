"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Button } from "@/components/ui/button";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { parseMoney } from "@/features/erp/shared/money";
import { kinds, th } from "@/features/erp/hr/strings";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { monthEndOf, ymOf } from "@/features/erp/hr/dashboard/range";
import { dayLabelLong, delta, monthLabel, num0, num1, rp, rpCompact } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";
import { BulletList, ComboChart } from "./charts";
import type { ChartRow } from "./charts";
import { SortTable } from "./sort-table";
import { useHrDrill, useHrScope } from "./drill";
import { deptFromKey, deptKey, deptPalette, Grid, type TabProps } from "./shared";

const c = thd.charts;
const k = thd.kpi;

export function LeaveTab({ data, leaveTypes }: TabProps) {
  const router = useRouter();
  const drill = useHrDrill();
  const { dept } = useHrScope();
  const month = ymOf(data.to);
  const cur = data.current;
  const prev = data.previous;
  const depts = deptPalette(data);

  // Fixed type order (the API's leave type list) so a type keeps its color.
  const typeOrder = Object.keys(leaveTypes).filter((t) => data.leaveTypeMonths.some((x) => x.type === t));
  for (const x of data.leaveTypeMonths) if (!typeOrder.includes(x.type)) typeOrder.push(x.type);
  const leaveStack = data.months.map((b) => {
    const row: ChartRow = { label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date };
    for (const x of data.leaveTypeMonths.filter((x) => x.month === b.date)) row[`t_${x.type}`] = x.days;
    return row;
  });

  const otStack = data.months.map((b) => {
    const row: ChartRow = { label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date };
    for (const d of data.deptMonths.filter((x) => x.month === b.date)) row[deptKey(d.departmentId)] = d.overtimeHours;
    return row;
  });

  const aging = data.pendingAging.map((a, i) => ({
    label: thd.agingBands[a.key] ?? a.key,
    tip: thd.agingBands[a.key] ?? a.key,
    count: a.count,
    color: i,
  }));

  return (
    <div className="page-stack">
      <StatStrip
        items={[
          {
            label: k.leaveDays,
            value: `${num0(cur.leaveDays)} ${thd.units.days}`,
            delta: delta(cur.leaveDays, prev.leaveDays),
            hint: k.leaveHint,
            href: hrLinks.leave({ from: data.from, to: data.to, status: "approved", dept }),
          },
          { label: k.pending, value: num0(data.pendingCount), caption: thd.openApprovals, href: hrLinks.leaveApprovals() },
          {
            label: k.overtimeHours,
            value: `${num1(cur.overtimeHours)} ${thd.units.hours}`,
            delta: delta(cur.overtimeHours, prev.overtimeHours),
            lowerIsBetter: true,
            hint: k.overtimeHint,
            href: hrLinks.overtime(month, { dept }),
          },
          {
            label: k.overtimeCost,
            value: rpCompact(cur.overtimeCost),
            delta: delta(parseMoney(cur.overtimeCost), parseMoney(prev.overtimeCost)),
            lowerIsBetter: true,
            hint: `${k.overtimeCostHint} ${k.exact(rp(cur.overtimeCost))}`,
            href: hrLinks.overtime(month, { dept }),
          },
        ]}
      />

      <Grid>
        <ChartCard title={c.leaveByType} description={`${c.leaveByTypeQ} ${thd.monthsWindow}.`} className="lg:col-span-2">
          <ComboChart
            data={leaveStack}
            stacked
            series={typeOrder.map((t, i) => ({ key: `t_${t}`, label: leaveTypes[t] ?? t, color: slotColor(i), format: num0 }))}
            yFormat={num0}
            onSelect={(r, _, key) =>
              drill("leave", { from: String(r.date), to: monthEndOf(String(r.date)), status: "approved", type: key?.slice(2) })
            }
            summary={c.leaveByType}
            height={260}
          />
        </ChartCard>
        <ChartCard title={c.leaveUtil} description={c.leaveUtilQ}>
          <BulletList
            items={data.departments
              .filter((d) => d.leaveQuota > 0)
              .map((d) => ({
                key: d.departmentId || "none",
                label: depts.name(d.departmentId),
                value: d.leaveUsed,
                target: d.leaveQuota,
                href: d.departmentId ? hrLinks.leave({ from: `${data.to.slice(0, 4)}-01-01`, to: data.to, type: "annual", status: "approved", dept: d.departmentId }) : undefined,
              }))}
            format={(v, t) => `${v} / ${t} ${thd.units.days}`}
          />
          <Button asChild variant="ghost" size="sm" className="mt-3 px-0">
            <Link href={hrLinks.leaveBalances()}>{thd.openList}</Link>
          </Button>
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.aging} description={c.agingQ}>
          <ComboChart
            data={aging}
            series={[{ key: "count", label: thd.series.count, color: slotColor(1), format: num0 }]}
            yFormat={num0}
            legend={false}
            onSelect={() => router.push(hrLinks.leaveApprovals())}
            summary={`${c.aging}: ${data.pendingCount}`}
            height={220}
          />
        </ChartCard>
        <ChartCard title={c.pendingList} className="lg:col-span-2">
          <SortTable
            rows={data.pending.slice(0, 12)}
            rowKey={(r) => r.id}
            rowHref={(r) => (r.kind === "leave" ? hrLinks.leaveApprovals() : hrLinks.overtimeApprovals())}
            initialSort={{ key: "age", desc: true }}
            showTotal={false}
            empty={thd.emptyPending}
            columns={[
              { key: "name", label: thd.cols.employee, value: (r) => r.fullName },
              { key: "age", label: thd.cols.age, numeric: true, value: (r) => r.ageDays, cell: (r) => thd.ageDays(r.ageDays) },
              { key: "kind", label: thd.cols.type, value: (r) => r.kind, cell: (r) => (r.kind === "leave" ? leaveTypes[r.type] ?? r.type : thd.kinds.overtime) },
              { key: "date", label: thd.cols.date, value: (r) => r.date, cell: (r) => dayLabelLong(r.date) },
              { key: "amount", label: thd.cols.amount, numeric: true, value: (r) => r.amount, cell: (r) => (r.kind === "leave" ? th.days(r.amount) : th.hours(r.amount)) },
              {
                key: "status",
                label: thd.cols.status,
                value: (r) => r.status,
                cell: (r) => <ErpStatusBadge status={r.status} labels={th.leave.statuses} kinds={kinds.leave} />,
              },
            ]}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.overtimeTrend} description={`${c.overtimeTrendQ} ${thd.monthsWindow}.`} className="lg:col-span-2">
          <ComboChart
            data={otStack}
            stacked
            series={depts.list.map((d) => ({ key: deptKey(d.id), label: d.name, color: d.color, format: num1 }))}
            yFormat={num0}
            onSelect={(r, _, key) => drill("ot", { month: ymOf(String(r.date)), dept: key ? deptFromKey(key) : undefined })}
            summary={c.overtimeTrend}
            height={260}
          />
        </ChartCard>
        <ChartCard title={c.overtimeDept} description={c.overtimeDeptQ}>
          <HBarChart
            items={[...data.departments]
              .sort((a, b) => b.overtimeHours - a.overtimeHours)
              .map((d) => ({
                label: depts.name(d.departmentId),
                value: d.overtimeHours,
                secondary: rpCompact(d.overtimeCost),
                href: d.departmentId ? hrLinks.overtime(month, { dept: d.departmentId }) : undefined,
              }))}
            format={(v) => `${num1(v)} ${thd.units.hours}`}
            emptyLabel={thd.empty}
          />
        </ChartCard>
      </Grid>

      <ChartCard title={c.topOvertime} description={c.topOvertimeQ}>
        <SortTable
          rows={data.topOvertime}
          rowKey={(r) => r.employeeId}
          rowHref={(r) => hrLinks.overtime(month, { employee: r.employeeId })}
          initialSort={{ key: "hours", desc: true }}
          columns={[
            { key: "name", label: thd.cols.employee, value: (r) => r.fullName },
            { key: "dept", label: thd.cols.department, value: (r) => r.departmentName },
            { key: "count", label: thd.cols.requests, numeric: true, value: (r) => r.attendanceRecords, total: num0(data.topOvertime.reduce((a, r) => a + r.attendanceRecords, 0)) },
            { key: "hours", label: thd.cols.overtime, numeric: true, value: (r) => r.overtimeHours, cell: (r) => num1(r.overtimeHours), total: num1(data.topOvertime.reduce((a, r) => a + r.overtimeHours, 0)) },
            {
              key: "cost",
              label: thd.cols.overtimeCost,
              numeric: true,
              value: (r) => parseMoney(r.overtimeCost),
              cell: (r) => rp(r.overtimeCost),
              total: rp(data.topOvertime.reduce((a, r) => a + parseMoney(r.overtimeCost), 0)),
            },
          ]}
        />
      </ChartCard>
    </div>
  );
}
