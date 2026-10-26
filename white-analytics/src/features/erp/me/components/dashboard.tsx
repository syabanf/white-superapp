"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Download } from "lucide-react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { MUTED_SERIES, slotColor } from "@/components/dashboard/charts/chart-primitives";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { DrillSheet, useDrill, useDrillParams, type DrillResult } from "@/features/erp/shared/drill";
import { parseMoney } from "@/features/erp/shared/money";
import { ChartEmpty, ComboChart } from "@/features/erp/hr/components/dashboard/charts";
import { dayLabel, dayLabelLong, delta, monthLabel, num0, pct1, rp, rpCompact } from "@/features/erp/hr/dashboard/format";
import { kinds, th } from "@/features/erp/hr/strings";
import { tm } from "@/features/erp/me/strings";
import type { MyDashboard, MyDashboardDay } from "@/features/erp/me/types";

const d = tm.dash;

export function MyKpis({ data }: { data: MyDashboard }) {
  const s = data.stats;
  const p = data.previous;
  const b = data.leaveBalance;
  const ym = data.month.slice(0, 7);
  const attendance = `/erp/me/attendance?month=${ym}`;
  return (
    <StatStrip
      scrollOnMobile
      minWidth={150}
      items={[
        {
          label: d.kpi.present,
          value: d.days(s.presentDays),
          delta: delta(s.presentDays, p.presentDays),
          caption: `${d.vsPrev}: ${p.presentDays}`,
          href: attendance,
        },
        {
          label: d.kpi.onTime,
          value: pct1(s.onTimeRate),
          delta: delta(s.onTimeRate, p.onTimeRate),
          caption: `${d.vsPrev}: ${pct1(p.onTimeRate)}`,
          href: attendance,
        },
        {
          label: d.kpi.late,
          value: d.days(s.lateDays),
          delta: delta(s.lateDays, p.lateDays),
          lowerIsBetter: true,
          caption: `${s.lateMinutes} mnt`,
          href: attendance,
        },
        { label: d.kpi.absent, value: d.days(s.absentDays), delta: delta(s.absentDays, p.absentDays), lowerIsBetter: true, href: attendance },
        { label: d.kpi.hours, value: d.hours(s.workHours), delta: delta(s.workHours, p.workHours), href: attendance },
        {
          label: d.kpi.overtime,
          value: d.hours(s.overtimeHours),
          delta: delta(s.overtimeHours, p.overtimeHours),
          lowerIsBetter: true,
          href: `/erp/me/overtime?month=${ym}`,
        },
        {
          label: d.kpi.avgIn,
          value: s.avgClockIn || "–",
          caption: p.avgClockIn ? `${d.vsPrev}: ${p.avgClockIn}` : undefined,
          href: attendance,
        },
        { label: d.kpi.leaveLeft, value: d.days(b.annualLeaveRemaining), caption: `${b.annualLeaveUsed} / ${b.annualLeaveTotal}`, href: "/erp/me/leave" },
      ]}
    />
  );
}

const STATUS_COLOR: Record<string, string> = {
  present: "var(--status-good)",
  remote: "var(--status-good)",
  late: "var(--status-warning)",
  absent: "var(--status-critical)",
  missing: "var(--status-critical)",
  leave: "var(--chart-1)",
  holiday: MUTED_SERIES,
  off: "transparent",
  upcoming: "transparent",
};
const LEGEND = ["present", "late", "absent", "leave", "holiday"];

/** Month grid (Monday first), each day tinted by its attendance status. */
export function AttendanceCalendar({ data, leaveTypes }: { data: MyDashboard; leaveTypes: Record<string, string> }) {
  const { open } = useDrill();
  const first = data.days[0];
  const lead = first ? (new Date(`${first.date}T00:00:00Z`).getUTCDay() + 6) % 7 : 0;
  return (
    <ChartCard title={d.calendar} description={d.calendarQ}>
      <div className="grid grid-cols-7 gap-1.5 text-center">
        {d.weekdays.map((w) => (
          <div key={w} className="pb-1 text-[11px] text-muted-foreground">
            {w}
          </div>
        ))}
        {Array.from({ length: lead }, (_, i) => (
          <div key={`lead-${i}`} />
        ))}
        {data.days.map((day) => (
          <DayCell key={day.date} day={day} today={day.date === data.today} leaveTypes={leaveTypes} onOpen={() => open("day", day.date)} />
        ))}
      </div>
      <ul className="mt-4 flex flex-wrap gap-x-4 gap-y-1.5 text-xs text-muted-foreground">
        {LEGEND.map((k) => (
          <li key={k} className="flex items-center gap-1.5">
            <span aria-hidden className="size-2.5 rounded-full" style={{ background: STATUS_COLOR[k] }} />
            {d.status[k]}
          </li>
        ))}
      </ul>
      <MyDaysDrill data={data} leaveTypes={leaveTypes} />
    </ChartCard>
  );
}

function dayItems(days: MyDashboardDay[], leaveTypes: Record<string, string>): DrillResult {
  const worked = days.reduce((a, x) => a + x.workHours, 0);
  const late = days.reduce((a, x) => a + x.lateMinutes, 0);
  return {
    summary: [
      { label: d.kpi.hours, value: d.hours(Math.round(worked * 10) / 10) },
      { label: d.kpi.late, value: d.lateBy(late) },
    ],
    items: days.map((x) => ({
      id: x.date,
      title: dayLabelLong(x.date),
      subtitle: [
        x.clockIn ? `${x.clockIn}${x.clockOut ? `–${x.clockOut}` : ""}` : "",
        x.lateMinutes ? d.lateBy(x.lateMinutes) : "",
        x.overtimeHours ? `${d.kpi.overtime} ${d.hours(x.overtimeHours)}` : "",
        x.status === "leave" ? (leaveTypes[x.note] ?? x.note) : x.note,
      ]
        .filter(Boolean)
        .join(" · "),
      value: x.workHours ? d.hours(x.workHours) : undefined,
      meta: d.status[x.status],
      href: x.status === "leave" ? "/erp/me/leave" : x.overtimeHours ? `/erp/me/overtime?month=${x.date.slice(0, 7)}` : undefined,
    })),
  };
}

/** Day (calendar cell) and week (weekly hours bar) sheets, from the dashboard payload. */
function MyDaysDrill({ data, leaveTypes }: { data: MyDashboard; leaveTypes: Record<string, string> }) {
  const ym = data.month.slice(0, 7);
  const week = (start: string) => {
    const end = new Date(Date.parse(`${start}T00:00:00Z`) + 6 * 86_400_000).toISOString().slice(0, 10);
    return data.days.filter((x) => x.date >= start && x.date <= end);
  };
  return (
    <DrillSheet
      kind={["day", "week"]}
      title={({ kind, key }) => (kind === "week" ? d.week(dayLabel(key)) : dayLabelLong(key))}
      load={({ kind, key }) => dayItems(kind === "week" ? week(key) : data.days.filter((x) => x.date === key), leaveTypes)}
      fullListHref={() => `/erp/me/attendance?month=${ym}`}
    />
  );
}

function DayCell({
  day,
  today,
  leaveTypes,
  onOpen,
}: {
  day: MyDashboardDay;
  today: boolean;
  leaveTypes: Record<string, string>;
  onOpen: () => void;
}) {
  const color = STATUS_COLOR[day.status] ?? "transparent";
  const tinted = color !== "transparent";
  const note = day.status === "leave" ? (leaveTypes[day.note] ?? day.note) : day.note;
  const label = [
    dayLabelLong(day.date),
    d.status[day.status],
    day.clockIn ? `${day.clockIn}${day.clockOut ? `–${day.clockOut}` : ""}` : "",
    day.lateMinutes ? d.lateBy(day.lateMinutes) : "",
    note,
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          onClick={day.status === "upcoming" ? undefined : onOpen}
          className={cn(
            "relative flex aspect-square min-h-9 cursor-pointer flex-col items-center justify-center rounded-xl text-sm tabular transition-transform hover:scale-105 focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none",
            day.status === "upcoming" && "cursor-default",
            day.status === "upcoming" && "border border-dashed text-muted-foreground",
            day.status === "off" && "text-muted-foreground/60",
            today && "ring-2 ring-brand ring-offset-1 ring-offset-card",
          )}
          style={tinted ? { background: `color-mix(in oklab, ${color} 24%, transparent)` } : undefined}
        >
          {Number(day.date.slice(8))}
          {tinted ? <span aria-hidden className="absolute bottom-1.5 size-1.5 rounded-full" style={{ background: color }} /> : null}
        </button>
      </TooltipTrigger>
      <TooltipContent className="text-xs">{label}</TooltipContent>
    </Tooltip>
  );
}

export function OnTimeTrend({ data }: { data: MyDashboard }) {
  const { apply } = useDrillParams();
  const rows = data.trend.map((m) => ({
    label: monthLabel(m.month),
    tip: monthLabel(m.month),
    month: m.month.slice(0, 7),
    me: m.presentDays ? m.onTimeRate : null,
    team: m.teamOnTimeRate || null,
  }));
  return (
    <ChartCard title={d.onTimeTrend} description={d.onTimeTrendQ}>
      <ComboChart
        data={rows}
        series={[
          { key: "me", label: d.me, type: "line", color: slotColor(0), format: pct1 },
          { key: "team", label: d.team, type: "line", color: MUTED_SERIES, format: pct1 },
        ]}
        yFormat={(v) => `${v}%`}
        yDomain="percent"
        onSelect={(r) => apply({ month: String(r.month) })}
        summary={`${d.onTimeTrend}: ${pct1(data.stats.onTimeRate)}`}
        height={220}
      />
    </ChartCard>
  );
}

export function WeeklyHours({ data }: { data: MyDashboard }) {
  const { open } = useDrill();
  const rows = data.weeks.map((w) => ({
    label: dayLabel(w.weekStart),
    tip: d.week(dayLabel(w.weekStart)),
    week: w.weekStart,
    worked: w.workHours,
    target: w.targetHours,
  }));
  return (
    <ChartCard title={d.weekly} description={d.weeklyQ}>
      <ComboChart
        data={rows}
        series={[
          { key: "worked", label: d.worked, color: slotColor(0), format: d.hours },
          { key: "target", label: d.target, color: MUTED_SERIES, format: d.hours },
        ]}
        yFormat={num0}
        onSelect={(r) => open("week", String(r.week))}
        summary={d.weekly}
        height={220}
      />
    </ChartCard>
  );
}

/** Annual leave as a ring (used vs remaining) with the other balances and recent requests. */
export function LeaveCard({ data, leaveTypes }: { data: MyDashboard; leaveTypes: Record<string, string> }) {
  const b = data.leaveBalance;
  const total = Math.max(b.annualLeaveTotal, 1);
  const used = Math.min(b.annualLeaveUsed, total);
  const r = 34;
  const c = 2 * Math.PI * r;
  return (
    <ChartCard
      title={d.leave}
      description={d.leaveQ}
      actions={
        <Button asChild variant="ghost" size="sm">
          <Link href="/erp/me/leave">{d.openList}</Link>
        </Button>
      }
    >
      <div className="flex items-center gap-5">
        <svg width={88} height={88} viewBox="0 0 88 88" role="img" aria-label={`${d.remaining} ${b.annualLeaveRemaining} dari ${b.annualLeaveTotal}`} className="shrink-0 -rotate-90">
          <circle cx={44} cy={44} r={r} fill="none" stroke="var(--muted)" strokeWidth={9} />
          <circle cx={44} cy={44} r={r} fill="none" stroke={slotColor(0)} strokeWidth={9} strokeLinecap="round" strokeDasharray={`${(used / total) * c} ${c}`} />
          <text x={44} y={44} transform="rotate(90 44 44)" textAnchor="middle" dominantBaseline="central" className="fill-foreground text-[20px] font-bold">
            {b.annualLeaveRemaining}
          </text>
        </svg>
        <dl className="grid flex-1 grid-cols-2 gap-x-4 gap-y-2 text-xs">
          <Stat label={d.remaining} value={d.days(b.annualLeaveRemaining)} />
          <Stat label={d.used} value={`${b.annualLeaveUsed} / ${b.annualLeaveTotal}`} />
          <Stat label={d.pendingDays} value={d.days(b.pendingDays)} />
          <Stat label={d.sick} value={d.days(b.sickLeaveUsed)} />
        </dl>
      </div>
      <h3 className="mt-5 mb-2 text-xs font-medium text-muted-foreground">{d.leaveHistory}</h3>
      {data.leaveHistory.length === 0 ? (
        <p className="text-sm text-muted-foreground">{d.noLeave}</p>
      ) : (
        <ul className="divide-y text-sm">
          {data.leaveHistory.slice(0, 5).map((l) => (
            <li key={l.id} className="flex items-center gap-3 py-2">
              <span className="min-w-0 flex-1">
                <span className="block truncate">{leaveTypes[l.leaveType] ?? l.leaveType}</span>
                <span className="block text-xs text-muted-foreground tabular">
                  {dayLabel(l.startDate)}
                  {l.endDate !== l.startDate ? ` – ${dayLabel(l.endDate)}` : ""} · {d.days(l.totalDays)}
                </span>
              </span>
              <ErpStatusBadge status={l.status} labels={th.leave.statuses} kinds={kinds.leave} />
            </li>
          ))}
        </ul>
      )}
    </ChartCard>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="text-sm font-semibold tabular">{value}</dd>
    </div>
  );
}

export function PayslipCard({ data }: { data: MyDashboard }) {
  const p = data.latestPayslip;
  const router = useRouter();
  const net = data.trend.map((m) => ({ label: monthLabel(m.month), tip: monthLabel(m.month), year: m.month.slice(0, 4), net: parseMoney(m.netPay) || null }));
  return (
    <ChartCard
      title={d.payslip}
      description={d.payslipQ}
      actions={
        <Button asChild variant="ghost" size="sm">
          <Link href="/erp/me/payslips">{d.allPayslips}</Link>
        </Button>
      }
    >
      {p ? (
        <>
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <p className="text-xs text-muted-foreground">
                {d.net} · {th.months[p.periodMonth - 1]} {p.periodYear}
              </p>
              <p className="text-[28px] leading-tight font-bold tabular" title={rp(p.netSalary)}>
                {rp(p.netSalary)}
              </p>
              <p className="text-xs text-muted-foreground">{d.attendanceDays(p.presentDays, p.workingDays)}</p>
            </div>
            <Button asChild variant="outline" size="sm">
              <a href={`/api/erp/v1/hr/me/payslips/${p.payslipId}/pdf`} target="_blank" rel="noreferrer">
                <Download className="size-3.5" /> {d.openPayslip}
              </a>
            </Button>
          </div>
          <dl className="mt-4 grid grid-cols-2 gap-x-4 gap-y-2 text-xs sm:grid-cols-3">
            <Stat label={d.base} value={rpCompact(p.baseSalary)} />
            <Stat label={d.allowances} value={rpCompact(p.allowances)} />
            <Stat label={d.gross} value={rpCompact(p.grossSalary)} />
            <Stat label={d.pph21} value={rpCompact(p.pph21)} />
            <Stat label={d.bpjs} value={rpCompact(p.bpjs)} />
            <Stat label={d.deductions} value={rpCompact(p.totalDeductions)} />
          </dl>
        </>
      ) : (
        <p className="text-sm text-muted-foreground">{d.noPayslip}</p>
      )}
      <h3 className="mt-5 mb-1 text-xs font-medium text-muted-foreground">{d.netTrend}</h3>
      <ComboChart
        data={net}
        series={[{ key: "net", label: d.net, type: "area", color: slotColor(0), format: rp }]}
        yFormat={(v) => rpCompact(v)}
        legend={false}
        onSelect={(r) => router.push(`/erp/me/payslips?year=${r.year}`)}
        summary={d.netTrend}
        height={150}
      />
    </ChartCard>
  );
}

export function PendingRequests({ data, leaveTypes }: { data: MyDashboard; leaveTypes: Record<string, string> }) {
  const href = { leave: "/erp/me/leave", overtime: "/erp/me/overtime", loan: "/erp/me/loans" } as const;
  return (
    <ChartCard title={d.pending}>
      {data.pending.length === 0 ? (
        <ChartEmpty height={96}>{d.noPending}</ChartEmpty>
      ) : (
        <ul className="divide-y text-sm">
          {data.pending.map((r) => (
            <li key={r.id}>
              <Link href={href[r.kind]} className="flex items-center gap-3 py-2.5 hover:bg-muted/30">
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-medium">
                    {d.kinds[r.kind]}
                    {r.kind === "leave" && r.type ? ` · ${leaveTypes[r.type] ?? r.type}` : ""}
                    {r.kind === "loan" && r.type ? ` · ${th.loans.types[r.type] ?? r.type}` : ""}
                  </span>
                  <span className="block text-xs text-muted-foreground tabular">
                    {r.kind === "loan" ? rp(r.amount) : `${r.date ? dayLabelLong(r.date) : ""} · ${r.kind === "leave" ? d.days(Number(r.amount)) : d.hours(Number(r.amount))}`}
                  </span>
                </span>
                <ErpStatusBadge
                  status={r.status}
                  labels={r.kind === "leave" ? th.leave.statuses : r.kind === "loan" ? th.loans.statuses : undefined}
                  kinds={r.kind === "leave" ? kinds.leave : r.kind === "loan" ? kinds.loan : kinds.overtime}
                />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </ChartCard>
  );
}

const TEAM_COLOR: Record<string, string> = {
  in: "var(--status-good)",
  remote: "var(--status-good)",
  late: "var(--status-warning)",
  absent: "var(--status-critical)",
  leave: "var(--chart-1)",
  not_in: MUTED_SERIES,
  off: MUTED_SERIES,
};

export function TeamToday({ data, leaveTypes }: { data: MyDashboard; leaveTypes: Record<string, string> }) {
  const counts = Object.keys(d.teamStatus)
    .map((k) => ({ k, n: data.teamToday.filter((m) => m.status === k).length }))
    .filter((x) => x.n > 0);
  return (
    <ChartCard title={d.teamToday} description={d.teamTodayQ}>
      <div className="mb-3 flex flex-wrap gap-2">
        {counts.map(({ k, n }) => (
          <span key={k} className="inline-flex items-center gap-1.5 rounded-full bg-muted/60 px-2.5 py-1 text-xs">
            <span aria-hidden className="size-2 rounded-full" style={{ background: TEAM_COLOR[k] }} />
            {d.teamStatus[k]} <span className="font-semibold tabular">{n}</span>
          </span>
        ))}
      </div>
      <ul className="divide-y text-sm">
        {data.teamToday.map((m) => (
          <li key={m.employeeId} className="flex items-center gap-3 py-2">
            <span aria-hidden className="size-2.5 shrink-0 rounded-full" style={{ background: TEAM_COLOR[m.status] }} />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-medium">{m.fullName}</span>
              <span className="block truncate text-xs text-muted-foreground">{m.positionName}</span>
            </span>
            <span className="shrink-0 text-right text-xs">
              <span className="block font-medium">{d.teamStatus[m.status]}</span>
              <span className="block text-muted-foreground tabular">
                {m.clockIn || (m.status === "not_in" && m.note ? d.shiftAt(m.note) : m.status === "leave" ? (leaveTypes[m.note] ?? m.note) : "")}
              </span>
            </span>
          </li>
        ))}
      </ul>
    </ChartCard>
  );
}

