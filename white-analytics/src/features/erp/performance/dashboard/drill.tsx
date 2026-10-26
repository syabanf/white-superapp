"use client";

import * as React from "react";
import { parseAsString, useQueryState } from "nuqs";
import { DrillSheet, useDrill, useDrillParams, type DrillAction, type DrillItem, type DrillResult, type DrillTarget } from "@/features/erp/shared/drill";
import { erpClientFetch } from "@/features/erp/shared/client";
import { formatDate } from "@/lib/format";
import { TaskDetailSheet } from "../components/task-detail-sheet";
import { jakartaToday } from "../lib";
import { sp } from "../strings";
import type { ListEntriesResponse, TaskItem } from "../types";
import { href } from "./lib";
import { ds } from "./strings";

/**
 * Drill-down of the performance dashboard (`?drill=<kind>:<filters>`).
 *
 * task: from to (occurrence days, inclusive) dept emp status indicator age turn
 *       status: pending | in_progress | done | approved | rejected | overdue | open
 * log:  from to dept emp status
 * `l` carries the mark's label. Task rows open the TaskDetailSheet (`?task=`).
 */
export type DrillFilter = Record<string, string | number | undefined | null>;

const t = ds.drill;

function encode(f: DrillFilter): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(f)) if (v != null && v !== "") p.set(k, String(v));
  return p.toString();
}

const read = (target: DrillTarget) => Object.fromEntries(new URLSearchParams(target.key));

/** Opens a performance drill; the dashboard's department and employee carry over. */
export function usePerfDrill() {
  const { open } = useDrill();
  const { get } = useDrillParams();
  const dept = get("dept");
  const emp = get("emp");
  return React.useCallback((kind: "task" | "log", f: DrillFilter) => open(kind, encode({ dept, emp, ...f })), [open, dept, emp]);
}

/** Months ("YYYY-MM") a day range touches, capped at 3 requests. */
function monthsOf(from: string, to: string): string[] {
  const out: string[] = [];
  const d = new Date(`${from.slice(0, 7)}-01T00:00:00Z`);
  while (out.length < 3 && d.toISOString().slice(0, 7) <= to.slice(0, 7)) {
    out.push(d.toISOString().slice(0, 7));
    d.setUTCMonth(d.getUTCMonth() + 1);
  }
  return out;
}

const AGE: Record<string, [number, number]> = { "1-3": [1, 3], "4-7": [4, 7], "8-14": [8, 14], "15-30": [15, 30], "31+": [31, Infinity] };

const daysLate = (day: string, today: string) => Math.round((Date.parse(today) - Date.parse(day)) / 86_400_000);

function matchStatus(it: TaskItem, status?: string): boolean {
  if (!status) return true;
  if (status === "overdue") return it.overdue;
  if (status === "open") return it.status !== "approved" && it.status !== "rejected";
  return it.status === status;
}

async function loadTasks(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const today = jakartaToday();
  const from = f.from ?? today.slice(0, 7) + "-01";
  const to = f.to ?? today;
  const lists = await Promise.all(
    monthsOf(from, to).map((month) =>
      erpClientFetch<{ items: TaskItem[] }>(
        `/v1/performance/tasks/items?${encode({ scope: "team", month, departmentId: f.dept, assigneeId: f.emp, indicatorId: f.indicator })}`,
        { signal },
      ),
    ),
  );
  const seen = new Set<string>();
  const age = f.age ? AGE[f.age] : null;
  const items = lists
    .flatMap((l) => l.items)
    .filter((it) => {
      if (seen.has(it.occurrenceId)) return false;
      seen.add(it.occurrenceId);
      if (!age && (it.occurrenceDate < from || it.occurrenceDate > to)) return false;
      if (age) {
        const late = daysLate(it.occurrenceDate, today);
        if (!it.overdue || late < age[0] || late > age[1]) return false;
      }
      return matchStatus(it, f.status);
    })
    .sort((a, b) => Number(b.overdue) - Number(a.overdue) || a.occurrenceDate.localeCompare(b.occurrenceDate));
  const done = items.filter((i) => i.status === "approved").length;
  return {
    summary: [
      { label: t.tasks, value: String(items.length) },
      { label: t.overdue, value: String(items.filter((i) => i.overdue).length) },
      { label: t.approved, value: String(done) },
      { label: t.completion, value: items.length ? `${Math.round((done / items.length) * 100)}%` : "–" },
    ],
    items: items.map(
      (it): DrillItem => ({
        id: it.occurrenceId,
        title: <TaskButton id={it.occurrenceId}>{it.title}</TaskButton>,
        subtitle: [it.assigneeName, it.departmentName].filter(Boolean).join(" · "),
        value: sp.taskStatus[it.status] ?? it.status,
        meta: it.overdue ? <span className="text-negative">{t.late(daysLate(it.occurrenceDate, today))}</span> : formatDate(it.occurrenceDate),
      }),
    ),
  };
}

async function loadLogs(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const res = await erpClientFetch<ListEntriesResponse>(
    `/v1/performance/logbook/entries?${encode({ departmentId: f.dept, employeeId: f.emp, status: f.status, from: f.from, to: f.to, limit: 100 })}`,
    { signal },
  );
  const scored = res.entries.filter((e) => e.kpiScore > 0);
  return {
    total: res.count,
    summary: [
      { label: t.entries, value: String(res.count) },
      { label: t.avgScore, value: scored.length ? (scored.reduce((a, e) => a + e.kpiScore, 0) / scored.length).toFixed(1) : "–" },
    ],
    items: res.entries.map((e) => ({
      id: e.id,
      title: e.title || e.templateName,
      subtitle: [e.employeeName, e.departmentName].filter(Boolean).join(" · "),
      value: sp.logbook.entryStatus[e.status] ?? e.status,
      meta: formatDate(e.entryDate),
      href: href("/erp/performance/logbook", { tab: "history", dept: e.departmentId, from: e.entryDate, to: e.entryDate }),
    })),
  };
}

function fullList(target: DrillTarget): string {
  const f = read(target);
  if (target.kind === "log") return href("/erp/performance/logbook", { tab: "history", dept: f.dept, status: f.status, from: f.from, to: f.to });
  return href("/erp/performance/tasks", { tab: "monitor", dept: f.dept, month: (f.to ?? f.from ?? jakartaToday()).slice(0, 7) });
}

/** The task detail sheet keyed by `?task=<occurrenceId>` (shallow). */
function useTaskParam() {
  return useQueryState("task", parseAsString.withOptions({ shallow: true, history: "push", scroll: false }));
}

function TaskButton({ id, children }: { id: string; children: React.ReactNode }) {
  const [, setTask] = useTaskParam();
  return (
    <button type="button" onClick={() => void setTask(id)} className="max-w-full cursor-pointer truncate text-left underline-offset-4 outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring">
      {children}
    </button>
  );
}

/** Opens one task's detail sheet from a table row. */
export function useOpenTask() {
  const [, setTask] = useTaskParam();
  return React.useCallback((id: string) => void setTask(id), [setTask]);
}

/** Drill sheet and task detail sheet of the performance dashboard; mount once per tab. */
export function PerfDrillSheet({ period }: { period: { from: string; to: string } }) {
  const [task, setTask] = useTaskParam();
  const { hrefWith } = useDrillParams();
  const actions = (target: DrillTarget): DrillAction[] => {
    const f = read(target);
    const out: DrillAction[] = [];
    const whole = f.from === period.from && f.to === period.to;
    if (f.from && f.to && !whole && f.dim !== "none") out.push({ label: t.focusPeriod, href: hrefWith({ preset: "custom", from: f.from, to: f.to }) });
    if (f.dim === "dept" && f.dept) out.push({ label: t.focusDim(f.l ?? ""), href: hrefWith({ dept: f.dept, emp: null }) });
    if (f.dim === "emp" && f.emp) out.push({ label: t.focusDim(f.l ?? ""), href: hrefWith({ emp: f.emp }) });
    return out;
  };
  return (
    <>
      <DrillSheet
        kind={["task", "log"]}
        title={(target) => read(target).l || (target.kind === "log" ? t.entries : t.tasks)}
        description={(target) => (target.kind === "log" ? t.logHint : t.taskHint)}
        chips={(target) => {
          const f = read(target);
          return [f.from && f.to ? (f.from === f.to ? formatDate(f.from) : `${formatDate(f.from)} – ${formatDate(f.to)}`) : null, f.status ? (t.status[f.status] ?? f.status) : null].filter(
            (c): c is string => c != null,
          );
        }}
        actions={actions}
        load={(target, signal) => (target.kind === "log" ? loadLogs(read(target), signal) : loadTasks(read(target), signal))}
        fullListHref={fullList}
      />
      <TaskDetailSheet occurrenceId={task} onClose={() => void setTask(null, { history: "replace" })} />
    </>
  );
}
