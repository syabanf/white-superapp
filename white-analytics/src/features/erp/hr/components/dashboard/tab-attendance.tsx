"use client";

import { useRouter } from "next/navigation";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { bucketEndOf } from "@/features/erp/hr/dashboard/range";
import { bucketLabel, dayLabelLong, delta, monthLabel, num0, num1, pct1 } from "@/features/erp/hr/dashboard/format";
import { thd } from "@/features/erp/hr/dashboard/strings";
import { ComboChart, HeatGrid } from "./charts";
import { SortTable } from "./sort-table";
import { deptPalette, Grid, type TabProps } from "./shared";

const c = thd.charts;
const k = thd.kpi;
const TOLERANCE = 15;

function offsetLabel(bin: number): string {
  if (bin <= -30) return "≤ -30";
  if (bin >= 60) return "≥ 60";
  return bin > 0 ? `+${bin}` : String(bin);
}

export function AttendanceTab({ data }: TabProps) {
  const router = useRouter();
  const cur = data.current;
  const prev = data.previous;
  const depts = deptPalette(data);

  const mix = data.buckets.map((b) => ({
    label: bucketLabel(b.date, data.bucket),
    tip: dayLabelLong(b.date),
    date: b.date,
    onTime: b.present - b.late,
    late: b.late,
    absent: b.absent,
  }));

  const weekdays = [1, 2, 3, 4, 5, 6, 7].filter((d) => d <= 5 || data.lateByWeekday.some((x) => x.col === d && x.total > 0));
  const heatRows = depts.list.filter((d) => data.lateByWeekday.some((x) => x.row === d.id && x.total > 0));

  const byDay = data.lateByDay.map((d) => ({
    label: String(d.col),
    tip: `Tanggal ${d.col}`,
    share: d.total ? Math.round((d.value / d.total) * 1000) / 10 : 0,
    late: d.value,
  }));

  const offsets = data.clockInOffsets.map((o) => {
    const bin = Number(o.key);
    const late = bin >= TOLERANCE;
    return { label: offsetLabel(bin), tip: `${offsetLabel(bin)} ${thd.units.minutes}`, onTime: late ? 0 : o.count, late: late ? o.count : 0 };
  });

  return (
    <div className="space-y-5">
      <StatStrip
        items={[
          { label: k.records, value: num0(cur.attendanceRecords), delta: delta(cur.attendanceRecords, prev.attendanceRecords) },
          { label: k.attendance, value: pct1(cur.attendanceRate), delta: delta(cur.attendanceRate, prev.attendanceRate), hint: k.attendanceHint },
          { label: k.lateCount, value: num0(cur.lateCount), delta: delta(cur.lateCount, prev.lateCount), lowerIsBetter: true },
          {
            label: k.avgLate,
            value: `${num1(cur.avgLateMinutes)} ${thd.units.minutes}`,
            delta: delta(cur.avgLateMinutes, prev.avgLateMinutes),
            lowerIsBetter: true,
          },
          { label: k.absent, value: num0(cur.absentCount), delta: delta(cur.absentCount, prev.absentCount), lowerIsBetter: true },
          { label: k.avgHours, value: `${num1(cur.avgWorkHours)} ${thd.units.hours}`, delta: delta(cur.avgWorkHours, prev.avgWorkHours) },
        ]}
      />

      <Grid>
        <ChartCard title={c.attendanceMix} description={c.attendanceMixQ} className="lg:col-span-2">
          <ComboChart
            data={mix}
            stacked
            series={[
              { key: "onTime", label: thd.series.onTime, color: slotColor(0), format: num0 },
              { key: "late", label: thd.series.late, color: slotColor(3), format: num0 },
              { key: "absent", label: thd.series.absent, color: slotColor(7), format: num0 },
            ]}
            yFormat={num0}
            onSelect={(r) => router.push(hrLinks.attendance({ from: String(r.date), to: bucketEndOf(String(r.date), data.bucket, data.to) }))}
            summary={`${c.attendanceMix}: ${num0(cur.attendanceRecords)}`}
            height={260}
          />
        </ChartCard>
        <ChartCard title={c.clockIn} description={c.clockInQ}>
          <ComboChart
            data={offsets}
            stacked
            series={[
              { key: "onTime", label: thd.series.onTime, color: slotColor(0), format: num0 },
              { key: "late", label: thd.series.late, color: slotColor(3), format: num0 },
            ]}
            yFormat={num0}
            reference={{ x: `+${TOLERANCE}`, label: "15 mnt" }}
            onSelect={(r) => r.late && router.push(hrLinks.attendance({ from: data.from, to: data.to, late: true }))}
            summary={c.clockIn}
            height={260}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.lateHeat} description={c.lateHeatQ} className="lg:col-span-2">
          <HeatGrid
            rows={heatRows.map((d) => ({ key: d.id, label: d.name }))}
            cols={weekdays.map((d) => ({ key: d, label: thd.weekdays[d - 1]! }))}
            cell={(row, col) => data.lateByWeekday.find((x) => x.row === row && x.col === col)}
            valueLabel={thd.series.late}
            onSelect={() => router.push(hrLinks.attendance({ from: data.from, to: data.to, late: true }))}
          />
        </ChartCard>
        <ChartCard title={c.lateByDay} description={c.lateByDayQ}>
          <ComboChart
            data={byDay}
            series={[{ key: "share", label: thd.series.late, color: slotColor(3), format: pct1 }]}
            yFormat={(v) => `${v}%`}
            summary={c.lateByDay}
            legend={false}
            height={232}
          />
        </ChartCard>
      </Grid>

      <Grid>
        <ChartCard title={c.deptAttendance} description={c.deptAttendanceQ} className="lg:col-span-2">
          <HeatGrid
            rows={depts.list.map((d) => ({ key: d.id, label: d.name }))}
            cols={data.months.map((b, i) => ({ key: i, label: monthLabel(b.date) }))}
            cell={(row, col) => {
              const d = data.deptMonths.find((x) => x.month === data.months[col]?.date && x.departmentId === row);
              return d?.attendanceRecords ? { value: Math.round((d.attendanceRate * d.attendanceRecords) / 100), total: d.attendanceRecords } : undefined;
            }}
            valueLabel={k.attendance}
          />
        </ChartCard>
        <ChartCard title={c.topLate} description={c.topLateQ}>
          <SortTable
            rows={data.topLate}
            rowKey={(r) => r.employeeId}
            rowHref={(r) => hrLinks.employee(r.employeeId)}
            initialSort={{ key: "late", desc: true }}
            columns={[
              { key: "name", label: thd.cols.employee, value: (r) => r.fullName, cell: (r) => <span className="block max-w-36 truncate">{r.fullName}</span> },
              { key: "late", label: thd.cols.late, numeric: true, value: (r) => r.late, total: num0(data.topLate.reduce((a, r) => a + r.late, 0)) },
              {
                key: "min",
                label: thd.cols.lateMinutes,
                numeric: true,
                value: (r) => r.lateMinutes,
                total: num0(data.topLate.reduce((a, r) => a + r.lateMinutes, 0)),
              },
            ]}
          />
        </ChartCard>
      </Grid>
    </div>
  );
}
