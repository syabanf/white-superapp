"use client";

import { CalendarClock, Clock, Plane, Timer, TrendingDown, UserCheck, Users, Wallet } from "lucide-react";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { ChartCard } from "@/components/dashboard/chart-card";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { parseMoney } from "@/features/erp/shared/money";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { bucketEndOf, ymOf } from "@/features/erp/hr/dashboard/range";
import { bucketLabel, dayLabelLong, delta, monthLabel, num0, num1, pct1, rp, rpCompact } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";
import { ComboChart, Donut } from "./charts";
import type { ChartRow } from "./charts";
import { SortTable } from "./sort-table";
import { useHrDrill, useHrScope } from "./drill";
import { SignedPair } from "./signed";
import { deptFromKey, deptKey, deptPalette, Grid, type TabProps } from "./shared";

const k = thd.kpi;
const c = thd.charts;

export function SummaryTab({ data, leaveTypes }: TabProps) {
  const drill = useHrDrill();
  const { dept } = useHrScope();
  const range = { from: data.from, to: data.to, dept };
  const cur = data.current;
  const prev = data.previous;
  const m = data.months;
  const depts = deptPalette(data);
  const payroll = data.payrollVisible;
  const curPay = parseMoney(cur.payrollEmployerCost);
  const prevPay = parseMoney(prev.payrollEmployerCost);

  const trend = data.buckets.map((b, i) => ({
    label: bucketLabel(b.date, data.bucket),
    tip: dayLabelLong(b.date),
    current: b.attendanceRecords ? b.attendanceRate : null,
    previous: data.previousBuckets[i]?.attendanceRecords ? data.previousBuckets[i]!.attendanceRate : null,
    date: b.date,
  }));

  const stack = m.map((b) => {
    const row: ChartRow = { label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date };
    for (const d of data.deptMonths.filter((x) => x.month === b.date)) row[deptKey(d.departmentId)] = d.headcount;
    return row;
  });
  const movement = m.map((b) => ({ label: monthLabel(b.date), tip: monthLabel(b.date), date: b.date, joiners: b.joiners, leavers: -b.leavers }));

  const reasons = data.absenceReasons.map((r) => ({
    key: r.key,
    label: thd.absenceKeys[r.key] ?? leaveTypes[r.key] ?? r.key,
    value: r.value,
  }));

  return (
    <div className="space-y-5">
      <KpiGrid cols={4}>
        <KpiTile
          label={k.headcount}
          hint={k.headcountHint}
          icon={<Users />}
          value={num0(cur.headcount)}
          delta={delta(cur.headcount, prev.headcount)}
          previous={num0(prev.headcount)}
          spark={m.map((b) => b.headcount)}
          href={hrLinks.employees({ dept })}
        />
        <KpiTile
          label={k.attendance}
          hint={k.attendanceHint}
          icon={<UserCheck />}
          value={pct1(cur.attendanceRate)}
          delta={delta(cur.attendanceRate, prev.attendanceRate)}
          previous={pct1(prev.attendanceRate)}
          spark={m.map((b) => b.attendanceRate)}
          href={hrLinks.attendance(range)}
        />
        <KpiTile
          label={k.late}
          hint={k.lateHint}
          icon={<Clock />}
          value={pct1(cur.lateRate)}
          delta={delta(cur.lateRate, prev.lateRate)}
          lowerIsBetter
          previous={pct1(prev.lateRate)}
          spark={m.map((b) => b.lateRate)}
          href={hrLinks.attendance({ ...range, late: true })}
        />
        <KpiTile
          label={k.leaveDays}
          hint={k.leaveHint}
          icon={<Plane />}
          value={`${num0(cur.leaveDays)} ${thd.units.days}`}
          delta={delta(cur.leaveDays, prev.leaveDays)}
          previous={`${num0(prev.leaveDays)} ${thd.units.days}`}
          spark={m.map((b) => b.leaveDays)}
          href={hrLinks.leave({ ...range, status: "approved" })}
        />
        <KpiTile
          label={k.overtimeHours}
          hint={k.overtimeHint}
          icon={<Timer />}
          value={`${num1(cur.overtimeHours)} ${thd.units.hours}`}
          delta={delta(cur.overtimeHours, prev.overtimeHours)}
          lowerIsBetter
          previous={`${num1(prev.overtimeHours)} ${thd.units.hours}`}
          spark={m.map((b) => b.overtimeHours)}
          href={hrLinks.overtime(ymOf(data.to), { dept })}
          size="sm"
        />
        <KpiTile
          label={k.overtimeCost}
          hint={`${k.overtimeCostHint} ${k.exact(rp(cur.overtimeCost))}`}
          value={rpCompact(cur.overtimeCost)}
          delta={delta(parseMoney(cur.overtimeCost), parseMoney(prev.overtimeCost))}
          lowerIsBetter
          previous={rpCompact(prev.overtimeCost)}
          spark={m.map((b) => parseMoney(b.overtimeCost))}
          href={hrLinks.overtime(ymOf(data.to), { dept })}
          size="sm"
        />
        <KpiTile
          label={k.turnover}
          hint={k.turnoverHint}
          icon={<TrendingDown />}
          value={pct1(cur.turnoverRate)}
          delta={delta(cur.turnoverRate, prev.turnoverRate)}
          lowerIsBetter
          caption={`${num0(cur.leavers)} ${thd.series.leavers.toLowerCase()} · ${num0(cur.joiners)} ${thd.series.joiners.toLowerCase()}`}
          spark={m.map((b) => b.turnoverRate)}
          href={hrLinks.employees({ dept, inactive: true })}
          size="sm"
        />
        {payroll ? (
          <KpiTile
            label={k.payroll}
            hint={`${k.payrollHint} ${k.exact(rp(curPay))}`}
            icon={<Wallet />}
            value={curPay ? rpCompact(curPay) : "–"}
            delta={curPay ? delta(curPay, prevPay) : null}
            lowerIsBetter
            caption={curPay ? undefined : k.noPayroll}
            previous={prevPay ? rpCompact(prevPay) : undefined}
            spark={m.map((b) => parseMoney(b.payrollGross) + parseMoney(b.payrollBpjsEmployer))}
            href={hrLinks.payroll(data.to.slice(0, 4))}
            size="sm"
          />
        ) : (
          <KpiTile
            label={k.avgHours}
            icon={<CalendarClock />}
            value={`${num1(cur.avgWorkHours)} ${thd.units.hours}`}
            delta={delta(cur.avgWorkHours, prev.avgWorkHours)}
            previous={`${num1(prev.avgWorkHours)} ${thd.units.hours}`}
            spark={m.map((b) => b.avgWorkHours)}
            href={hrLinks.attendance(range)}
            size="sm"
          />
        )}
      </KpiGrid>

      <Grid>
        <ChartCard
          title={c.attendanceTrend}
          description={c.attendanceTrendQ}
          className="lg:col-span-2"
          table={
            <SortTable
              rows={trend}
              rowKey={(r) => r.date}
              showTotal={false}
              initialSort={{ key: "date", desc: false }}
              columns={[
                { key: "date", label: thd.cols.date, value: (r) => r.date, cell: (r) => r.tip },
                { key: "cur", label: thd.series.current, numeric: true, value: (r) => r.current ?? -1, cell: (r) => (r.current == null ? "–" : pct1(r.current)) },
                { key: "prev", label: thd.series.previous, numeric: true, value: (r) => r.previous ?? -1, cell: (r) => (r.previous == null ? "–" : pct1(r.previous)) },
              ]}
            />
          }
        >
          <ComboChart
            data={trend}
            series={[
              { key: "current", label: thd.series.current, type: "line", color: slotColor(0), format: pct1 },
              { key: "previous", label: thd.series.previous, type: "line", color: MUTED_SERIES, format: pct1 },
            ]}
            yFormat={(v) => `${v}%`}
            yDomain="percent"
            onSelect={(r) => drill("att", { from: String(r.date), to: bucketEndOf(String(r.date), data.bucket, data.to) })}
            summary={`${c.attendanceTrend}: ${pct1(cur.attendanceRate)}`}
          />
        </ChartCard>
        <ChartCard title={c.absence} description={c.absenceQ}>
          <Donut
            slices={reasons}
            centerLabel={thd.units.days}
            format={num0}
            onSelect={(s) =>
              s.key === "absent"
                ? drill("att", { from: data.from, to: data.to, status: "absent" })
                : drill("leave", { from: data.from, to: data.to, type: s.key, status: "approved" })
            }
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.headcountByDept} description={`${c.headcountByDeptQ} ${thd.monthsWindow}.`} className="lg:col-span-2">
          <ComboChart
            data={stack}
            stacked
            series={depts.list.map((d) => ({ key: deptKey(d.id), label: d.name, color: d.color, format: num0 }))}
            yFormat={num0}
            onSelect={(_, __, key) => drill("emp", key ? { dept: deptFromKey(key) } : {})}
            summary={`${c.headcountByDept}: ${num0(cur.headcount)}`}
            height={260}
          />
        </ChartCard>
        <ChartCard title={c.movement} description={c.movementQ}>
          <ComboChart
            data={movement}
            series={[
              { key: "joiners", label: thd.series.joiners, color: "var(--positive)", signed: true, format: num0 },
              { key: "leavers", label: thd.series.leavers, color: "var(--negative)", signed: true, format: (v) => num0(Math.abs(v)) },
            ]}
            stacked
            onSelect={(r, _, key) => drill("emp", { [key === "leavers" ? "left" : "joined"]: String(r.date).slice(0, 7) })}
            yDomain={["auto", "auto"]}
            yFormat={(v) => num0(Math.abs(v))}
            summary={`${c.movement}: ${cur.joiners} / ${cur.leavers}`}
            height={260}
          />
        </ChartCard>
      </Grid>

      <ChartCard title={c.departments} description={c.departmentsQ}>
        <SortTable
          rows={data.departments}
          rowKey={(r) => r.departmentId || "none"}
          rowHref={(r) => (r.departmentId ? hrLinks.employees({ dept: r.departmentId }) : undefined)}
          initialSort={{ key: "headcount", desc: true }}
          columns={[
            { key: "name", label: thd.cols.department, value: (r) => depts.name(r.departmentId) },
            { key: "headcount", label: thd.cols.headcount, numeric: true, value: (r) => r.headcount, total: num0(cur.headcount) },
            { key: "moves", label: `${thd.cols.joiners}/${thd.cols.leavers}`, numeric: true, value: (r) => r.joiners - r.leavers, cell: (r) => <SignedPair plus={r.joiners} minus={r.leavers} />, total: <SignedPair plus={cur.joiners} minus={cur.leavers} /> },
            { key: "att", label: thd.cols.attendance, numeric: true, value: (r) => r.attendanceRate, cell: (r) => pct1(r.attendanceRate), total: pct1(cur.attendanceRate) },
            { key: "late", label: thd.cols.late, numeric: true, value: (r) => r.late, total: num0(cur.lateCount) },
            { key: "leave", label: thd.cols.leave, numeric: true, value: (r) => r.leaveDays, cell: (r) => num0(r.leaveDays), total: num0(cur.leaveDays) },
            { key: "ot", label: thd.cols.overtime, numeric: true, value: (r) => r.overtimeHours, cell: (r) => num1(r.overtimeHours), total: num1(cur.overtimeHours) },
            ...(payroll
              ? [
                  {
                    key: "pay",
                    label: thd.cols.payroll,
                    numeric: true,
                    value: (r: (typeof data.departments)[number]) => parseMoney(r.payrollEmployerCost),
                    cell: (r: (typeof data.departments)[number]) => rpCompact(r.payrollEmployerCost),
                    total: rpCompact(curPay),
                  },
                ]
              : []),
          ]}
        />
      </ChartCard>
    </div>
  );
}
