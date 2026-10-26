"use client";

import * as React from "react";
import { useSearchParams } from "next/navigation";
import { CalendarRange, Filter } from "lucide-react";
import { DrillSheet, useDrill, useDrillParams, type DrillAction, type DrillResult, type DrillTarget } from "@/features/erp/shared/drill";
import { erpClientFetch } from "@/features/erp/shared/client";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { parseMoney } from "@/features/erp/shared/money";
import { kinds, monthLabel as monthName, th } from "@/features/erp/hr/strings";
import { hrLinks } from "@/features/erp/hr/dashboard/links";
import { dayLabelLong, num0, num1, rangeLabel, rpCompact } from "@/features/erp/hr/dashboard/format";
import { monthEndOf } from "@/features/erp/hr/dashboard/range";
import { thd } from "@/features/erp/hr/dashboard/strings";
import type { HrDashboard } from "@/features/erp/hr/dashboard/types";
import type { Employee, ListAttendanceResponse, ListEmployeesResponse, ListLeavesResponse, Overtime, PayrollRun, PayrollRunDetail } from "@/features/erp/hr/types";
import { deptPalette } from "./shared";

/**
 * Drill sheets of the HR dashboard. Every chart mark opens one through
 * `useHrDrill()`: the key is a query string of the filters behind the mark,
 * merged with the dashboard's own department filter.
 */

type Params = Record<string, string | undefined>;
export type HrDrillKind = "att" | "leave" | "ot" | "emp" | "pay";

const PREVIEW = 50;
const t = thd.drill;

const encode = (p: Params) => {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(p)) if (v) sp.set(k, v);
  return sp.toString();
};
const decode = (key: string): Params => Object.fromEntries(new URLSearchParams(key));

/** The dashboard's dimension filters, applied to every drill and list link. */
export function useHrScope() {
  const sp = useSearchParams();
  return { dept: sp.get("dept") ?? undefined, status: sp.get("status") ?? undefined, position: sp.get("position") ?? undefined };
}

/** Opens a drill sheet; the dashboard department applies unless the mark names one. */
export function useHrDrill() {
  const { open } = useDrill();
  const scope = useHrScope();
  return React.useCallback(
    (kind: HrDrillKind, p: Params) => open(kind, encode({ dept: scope.dept, ...p })),
    [open, scope.dept],
  );
}

const isoWeekday = (day: string) => ((new Date(`${day}T00:00:00Z`).getUTCDay() + 6) % 7) + 1;

function tenureYears(join: string, at: string) {
  return (Date.parse(`${at}T00:00:00Z`) - Date.parse(`${join}T00:00:00Z`)) / (365.25 * 86_400_000);
}

const TENURE: Record<string, [number, number]> = { "<1": [0, 1], "1-2": [1, 2], "2-3": [2, 3], "3-5": [3, 5], "5+": [5, Infinity] };

function employeeItems(list: Employee[]) {
  return list.map((e) => ({
    id: e.id,
    title: e.fullName,
    subtitle: [e.positionName, e.departmentName].filter(Boolean).join(" · "),
    meta: e.endDate ? t.leftOn(dayLabelLong(e.endDate)) : t.joinedOn(dayLabelLong(e.joinDate)),
    href: hrLinks.employee(e.id),
  }));
}

async function loadAttendance(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const local = Boolean(p.wd || p.dom);
  const q = encode({
    startDate: p.from,
    endDate: p.to,
    departmentId: p.dept,
    employeeId: p.employee,
    status: p.status === "late" ? undefined : p.status,
    onlyLate: p.status === "late" ? "true" : undefined,
    pageSize: String(local ? 500 : PREVIEW),
  });
  const res = await erpClientFetch<ListAttendanceResponse>(`/v1/hr/attendance?${q}`, { signal });
  let rows = res.records;
  if (p.wd) rows = rows.filter((r) => isoWeekday(r.date) === Number(p.wd));
  if (p.dom) rows = rows.filter((r) => Number(r.date.slice(8, 10)) === Number(p.dom));
  const total = local ? rows.length : res.total;
  const late = rows.filter((r) => r.isLate);
  return {
    total,
    summary: [
      { label: t.records, value: num0(total) },
      late.length
        ? { label: t.avgLate, value: `${num1(late.reduce((a, r) => a + r.lateMinutes, 0) / late.length)} ${thd.units.minutes}` }
        : { label: t.hours, value: `${num1(rows.reduce((a, r) => a + r.workHours, 0))} ${thd.units.hours}` },
    ],
    items: rows.slice(0, PREVIEW).map((r) => ({
      id: r.id,
      title: r.employee.fullName,
      subtitle: `${dayLabelLong(r.date)} · ${r.employee.departmentName || thd.noDepartment}`,
      value: r.lateMinutes ? th.minutes(r.lateMinutes) : r.workHours ? th.hours(r.workHours) : undefined,
      badge: <ErpStatusBadge status={r.status} labels={th.attendance.statuses} kinds={kinds.attendance} />,
      href: hrLinks.attendance({ from: r.date, to: r.date, employee: r.employee.id }),
    })),
  };
}

async function loadLeaves(p: Params, signal: AbortSignal, types: Record<string, string>): Promise<DrillResult> {
  const q = encode({
    startDate: p.from,
    endDate: p.to,
    leaveType: p.type,
    status: p.status,
    departmentId: p.dept,
    employeeId: p.employee,
    pageSize: String(PREVIEW),
  });
  const res = await erpClientFetch<ListLeavesResponse>(`/v1/hr/leaves?${q}`, { signal });
  return {
    total: res.total,
    summary: [
      { label: t.requests, value: num0(res.total) },
      { label: t.days, value: `${num0(res.leaves.reduce((a, l) => a + l.totalDays, 0))} ${thd.units.days}` },
    ],
    items: res.leaves.map((l) => ({
      id: l.id,
      title: l.employee.fullName,
      subtitle: `${types[l.leaveType] ?? l.leaveType} · ${rangeLabel(l.startDate, l.endDate)}`,
      value: th.days(l.totalDays),
      badge: <ErpStatusBadge status={l.status} labels={th.leave.statuses} kinds={kinds.leave} />,
      href: hrLinks.employee(l.employee.id),
    })),
  };
}

async function loadOvertime(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const q = encode({ month: p.month, status: p.status ?? "approved", departmentId: p.dept, employeeId: p.employee });
  const res = await erpClientFetch<{ overtime: Overtime[] }>(`/v1/hr/overtime?${q}`, { signal });
  const rows = res.overtime;
  return {
    total: rows.length,
    summary: [
      { label: t.requests, value: num0(rows.length) },
      { label: t.hours, value: `${num1(rows.reduce((a, o) => a + o.hours, 0))} ${thd.units.hours}` },
    ],
    items: rows.slice(0, PREVIEW).map((o) => ({
      id: o.id,
      title: o.employee.fullName,
      subtitle: `${dayLabelLong(o.date)} · ${o.startTime}–${o.endTime} · ${o.employee.departmentName || thd.noDepartment}`,
      value: th.hours(o.hours),
      href: hrLinks.employee(o.employee.id),
    })),
  };
}

async function loadEmployees(p: Params, signal: AbortSignal, at: string): Promise<DrillResult> {
  const moved = Boolean(p.joined || p.left || p.tenure);
  const q = encode({
    departmentId: p.dept,
    positionId: p.position,
    level: p.level,
    gender: p.gender,
    employmentStatus: p.status,
    includeInactive: moved ? "true" : undefined,
    pageSize: "500",
  });
  const res = await erpClientFetch<ListEmployeesResponse>(`/v1/hr/employees?${q}`, { signal });
  let rows = res.employees;
  if (p.joined) rows = rows.filter((e) => e.joinDate.startsWith(p.joined!));
  if (p.left) rows = rows.filter((e) => e.endDate?.startsWith(p.left!));
  if (p.tenure && TENURE[p.tenure]) {
    const [lo, hi] = TENURE[p.tenure]!;
    rows = rows.filter((e) => !e.endDate && tenureYears(e.joinDate, at) >= lo && tenureYears(e.joinDate, at) < hi);
  }
  return { total: rows.length, summary: [{ label: t.people, value: num0(rows.length) }], items: employeeItems(rows.slice(0, PREVIEW)) };
}

async function loadPayroll(p: Params, signal: AbortSignal, deptName?: string): Promise<DrillResult> {
  const [year, month] = (p.month ?? "").split("-").map(Number);
  const res = await erpClientFetch<{ runs: PayrollRun[] }>(`/v1/hr/payroll/runs?year=${year}`, { signal });
  const runs = res.runs.filter((r) => r.periodMonth === month && r.status !== "draft");
  const run = runs.find((r) => r.status === "paid") ?? runs[0];
  if (!p.dept || !run) {
    return {
      total: runs.length,
      summary: [
        { label: t.runs, value: num0(runs.length) },
        { label: thd.kpi.gross, value: rpCompact(runs.reduce((a, r) => a + parseMoney(r.totalGross), 0)) },
      ],
      items: runs.map((r) => ({
        id: r.id,
        title: r.runName,
        subtitle: `${monthName(r.periodMonth, r.periodYear)} · ${num0(r.totalEmployees)} ${thd.units.people}`,
        value: rpCompact(r.totalNet),
        badge: <ErpStatusBadge status={r.status} labels={th.payroll.statuses} kinds={kinds.payroll} />,
        href: hrLinks.payrollRun(r.id),
      })),
    };
  }
  const detail = await erpClientFetch<PayrollRunDetail>(`/v1/hr/payroll/runs/${run.id}`, { signal });
  const slips = detail.payslips.filter((s) => s.employee.departmentName === deptName);
  return {
    total: slips.length,
    summary: [
      { label: t.slips, value: num0(slips.length) },
      { label: thd.kpi.gross, value: rpCompact(slips.reduce((a, s) => a + parseMoney(s.grossSalary), 0)) },
    ],
    items: slips.map((s) => ({
      id: s.id,
      title: s.employee.fullName,
      subtitle: `${run.runName} · ${s.employee.employeeNo}`,
      value: rpCompact(s.netSalary),
      meta: t.net,
      href: hrLinks.payrollRun(run.id, { dept: p.dept, employee: s.employee.id }),
    })),
  };
}

/** Mount once per dashboard tab set. */
export function HrDrill({ data, leaveTypes }: { data: HrDashboard; leaveTypes: Record<string, string> }) {
  const depts = deptPalette(data);
  const { hrefWith } = useDrillParams();
  const scope = useHrScope();
  const deptName = (id?: string) => (id ? depts.name(id) : undefined);

  const chips = ({ key }: DrillTarget) => {
    const p = decode(key);
    const period = p.month ? monthName(Number(p.month.slice(5, 7)), Number(p.month.slice(0, 4))) : p.from && p.to ? rangeLabel(p.from, p.to) : undefined;
    return [
      period,
      deptName(p.dept),
      p.wd ? thd.weekdays[Number(p.wd) - 1] : undefined,
      p.dom ? t.dayOfMonth(p.dom) : undefined,
      p.type ? (leaveTypes[p.type] ?? p.type) : undefined,
      p.status ? (p.label ?? t.statusLabel(p.status)) : p.label,
      p.level ? (th.masters.levels[p.level] ?? p.level) : undefined,
      p.gender ? (th.genders[p.gender] ?? p.gender) : undefined,
      p.tenure ? t.tenure(thd.tenureBands[p.tenure] ?? p.tenure) : undefined,
    ].filter((c): c is string => Boolean(c));
  };

  // Time marks zoom the dashboard into their range; department marks filter it.
  const actions = ({ key }: DrillTarget): DrillAction[] => {
    const p = decode(key);
    const out: DrillAction[] = [];
    const from = p.from ?? (p.month ? `${p.month}-01` : undefined);
    const to = p.to ?? (p.month ? monthEndOf(`${p.month}-01`) : undefined);
    if (from && to && (from !== data.from || to !== data.to)) {
      out.push({
        label: t.zoom,
        icon: <CalendarRange aria-hidden />,
        href: hrefWith({ from, to, preset: null, back: `${data.from}_${data.to}` }),
      });
    }
    if (p.dept && p.dept !== scope.dept) {
      out.push({ label: t.filterDept, icon: <Filter aria-hidden />, href: hrefWith({ dept: p.dept }) });
    }
    return out;
  };

  return (
    <>
      <DrillSheet
        kind="att"
        title={({ key }) => t.attendanceTitle(decode(key).status)}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadAttendance(decode(key), signal)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return hrLinks.attendance({ from: p.from ?? data.from, to: p.to ?? data.to, dept: p.dept, employee: p.employee, late: p.status === "late", status: p.status === "late" ? undefined : p.status });
        }}
      />
      <DrillSheet
        kind="leave"
        title={() => t.leaveTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadLeaves(decode(key), signal, leaveTypes)}
        fullListHref={({ key }) => hrLinks.leave(decode(key))}
      />
      <DrillSheet
        kind="ot"
        title={() => t.overtimeTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadOvertime(decode(key), signal)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return hrLinks.overtime(p.month ?? data.to.slice(0, 7), { dept: p.dept, employee: p.employee, status: p.status });
        }}
      />
      <DrillSheet
        kind="emp"
        title={({ key }) => {
          const p = decode(key);
          return p.joined ? t.joinersTitle : p.left ? t.leaversTitle : t.employeesTitle;
        }}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadEmployees(decode(key), signal, data.to)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return hrLinks.employees({ dept: p.dept, position: p.position, level: p.level, gender: p.gender, status: p.status, inactive: Boolean(p.left) });
        }}
      />
      <DrillSheet
        kind="pay"
        title={() => t.payrollTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => {
          const p = decode(key);
          return loadPayroll(p, signal, deptName(p.dept));
        }}
        fullListHref={({ key }) => {
          const p = decode(key);
          return p.dept ? hrLinks.salaries({ dept: p.dept }) : hrLinks.payroll(p.month?.slice(0, 4));
        }}
      />
    </>
  );
}
