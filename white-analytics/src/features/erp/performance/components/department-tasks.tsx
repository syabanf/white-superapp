"use client";

import * as React from "react";
import { AlertTriangle, CheckCircle2, Clock, Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDate, formatNumber } from "@/lib/format";
import { cn } from "@/lib/utils";
import { occurrenceAction } from "../actions";
import { daysBetween, groupByStatus, quickAction, ratio, STATUS_ORDER } from "../lib";
import { sp } from "../strings";
import type { IndicatorTasks, Monitor, MonitorRow, TaskItem, TaskMeta } from "../types";
import { PriorityBadge, TaskStatusBadge } from "./common";
import { TaskDetailSheet } from "./task-detail-sheet";
import { TaskMetaLine, TaskRow } from "./task-row";

const s = sp.tasks;

/** Holds the open occurrence so every task view shares one detail sheet. */
function useDetail(meta: TaskMeta | null) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  const sheet = <TaskDetailSheet occurrenceId={openId} meta={meta} onClose={() => setOpenId(null)} />;
  return { open: setOpenId, sheet };
}

const COLUMN_TONE: Record<string, string> = {
  pending: "",
  in_progress: "text-[var(--chart-1)]",
  done: "text-[var(--status-warning)]",
  approved: "text-[var(--status-good)]",
  rejected: "text-destructive",
};

/** Department board: one column per status, quick start/finish/approve on each card. */
export function TaskBoard({ items, meta }: { items: TaskItem[]; meta: TaskMeta }) {
  const { open, sheet } = useDetail(meta);
  const columns = React.useMemo(() => groupByStatus(items), [items]);
  if (items.length === 0) return <ErpEmpty title={s.empty} />;
  return (
    <>
      <div className="flex snap-x gap-3 overflow-x-auto pb-3 scrollbar-thin">
        {STATUS_ORDER.map((status) => (
          <section key={status} className="flex w-72 shrink-0 snap-start flex-col rounded-2xl bg-muted/50">
            <h2 className={cn("px-4 pt-3 pb-1 text-sm font-semibold", COLUMN_TONE[status])}>
              {sp.taskStatus[status]} <span className="font-normal text-muted-foreground">· {columns[status].length}</span>
            </h2>
            <div className="flex min-h-24 flex-col gap-2 px-3 pb-3">
              {columns[status].map((item) => (
                <TaskCard key={item.occurrenceId} item={item} onOpen={open} />
              ))}
            </div>
          </section>
        ))}
      </div>
      {sheet}
    </>
  );
}

function TaskCard({ item, onOpen }: { item: TaskItem; onOpen: (id: string) => void }) {
  const { pending, run } = useErpAction();
  const next = quickAction(item);
  return (
    <div className="rounded-xl border bg-card p-3 shadow-xs transition hover:border-brand/40">
      <button type="button" onClick={() => onOpen(item.occurrenceId)} className="w-full text-left outline-none focus-visible:ring-2 focus-visible:ring-ring">
        <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
          {item.title}
          <PriorityBadge priority={item.priority} />
          {item.overdue ? <TaskStatusBadge status={item.status} overdue /> : null}
        </p>
        <div className="mt-1.5">
          <TaskMetaLine item={item} showAssignee />
        </div>
      </button>
      {next ? (
        <Button
          size="xs"
          variant="secondary"
          className="mt-2"
          disabled={pending}
          onClick={() => run(() => occurrenceAction(item.occurrenceId, { action: next }))}
        >
          {next === "start" ? <Play /> : <CheckCircle2 />}
          {next === "start" ? s.quickStart : next === "done" ? s.quickDone : s.quickApprove}
        </Button>
      ) : null}
    </div>
  );
}

const pct = (v?: number) => (v == null ? "–" : `${v}%`);

function MonitorRows({ rows, today, late, onOpen }: { rows: MonitorRow[]; today: string; late: boolean; onOpen: (id: string) => void }) {
  return (
    <ul className="divide-y">
      {rows.map((o) => (
        <li key={o.occurrenceId}>
          <button
            type="button"
            onClick={() => onOpen(o.occurrenceId)}
            className="flex w-full flex-wrap items-center gap-x-3 gap-y-1 py-2 text-left text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <span className="w-20 text-xs text-muted-foreground tabular">{formatDate(o.occurrenceDate)}</span>
            <span className="min-w-0 flex-1 basis-40">
              <span className="block font-medium">{o.title}</span>
              <span className="block text-xs text-muted-foreground">
                {o.departmentName} · {s.pic(o.assigneeName ?? "")}
                {o.lastNote ? ` · “${o.lastNote}”` : ""}
              </span>
            </span>
            <span className="text-xs text-muted-foreground tabular">{o.progress}%</span>
            {late ? (
              <span className="text-xs font-medium text-destructive">{s.lateDays(daysBetween(o.occurrenceDate, today))}</span>
            ) : null}
            <TaskStatusBadge status={o.status} />
          </button>
        </li>
      ))}
    </ul>
  );
}

/** Completion and progress per department and assignee, overdue rows, rows waiting for review. */
export function MonitorPanel({ data, meta }: { data: Monitor; meta: TaskMeta | null }) {
  const { open, sheet } = useDetail(meta);
  const m = data.summary;
  return (
    <div className="flex flex-col gap-4">
      <KpiGrid cols={5}>
        <KpiTile size="sm" label={sp.overview.due} value={formatNumber(m.due)} caption={s.scheduled(m.scheduled)} />
        <KpiTile size="sm" label={sp.overview.completion} value={ratio(m.completion)} caption={s.approvedCount(m.approved)} />
        <KpiTile size="sm" label={sp.overview.avgProgress} value={pct(m.avgProgress)} caption={s.inProgressCount(m.inProgress)} />
        <KpiTile size="sm" label={sp.overview.overdue} value={formatNumber(m.overdue)} accentVar={m.overdue > 0 ? "var(--status-critical)" : undefined} />
        <KpiTile size="sm" label={sp.overview.waitingReview} value={formatNumber(m.waitingReview)} />
      </KpiGrid>

      {data.departments.length === 0 ? <ErpEmpty title={s.monitorEmpty} compact /> : null}

      {data.departments.map((d) => (
        <Card key={d.departmentId} className="gap-3 p-(--card-pad)">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="font-semibold">{d.departmentName}</p>
              <p className="text-xs text-muted-foreground">
                {d.summary.due} {sp.overview.due.toLowerCase()} · {s.approvedCount(d.summary.approved)} · {d.summary.waitingReview}{" "}
                {sp.overview.waitingReview.toLowerCase()}
              </p>
            </div>
            {d.summary.overdue > 0 ? (
              <TaskStatusBadge status="pending" overdue />
            ) : (
              <span className="text-xs font-medium text-[var(--status-good)]">{s.onTime}</span>
            )}
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground">
                <span>{sp.overview.completion}</span>
                <span className="font-semibold tabular">{ratio(d.summary.completion)}</span>
              </div>
              <Progress value={(d.summary.completion ?? 0) * 100} className="h-2" aria-label={sp.overview.completion} />
            </div>
            <div>
              <div className="mb-1 flex justify-between text-xs text-muted-foreground">
                <span>{s.progress}</span>
                <span className="font-semibold tabular">{pct(d.summary.avgProgress)}</span>
              </div>
              <Progress value={d.summary.avgProgress ?? 0} className="h-2" aria-label={s.progress} />
            </div>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{s.assignee}</TableHead>
                <TableHead className="text-right">{sp.overview.due}</TableHead>
                <TableHead className="text-right">{sp.overview.completion}</TableHead>
                <TableHead className="hidden text-right sm:table-cell">{s.progress}</TableHead>
                <TableHead className="text-right">{sp.overview.overdue}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {d.assignees.map((a) => (
                <TableRow key={a.assigneeId ?? "dept"}>
                  <TableCell>{a.assigneeName || s.wholeDepartment}</TableCell>
                  <TableCell className="text-right tabular">{a.summary.due}</TableCell>
                  <TableCell className="text-right tabular">{ratio(a.summary.completion)}</TableCell>
                  <TableCell className="hidden text-right tabular sm:table-cell">{pct(a.summary.avgProgress)}</TableCell>
                  <TableCell className={cn("text-right font-semibold tabular", a.summary.overdue > 0 ? "text-destructive" : "text-muted-foreground")}>
                    {a.summary.overdue}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      ))}

      <Card className="gap-2 p-(--card-pad)">
        <p className="flex items-center gap-1.5 text-sm font-semibold">
          <AlertTriangle className="size-4 text-destructive" /> {sp.overview.overdueTasks}
        </p>
        {data.overdue.length === 0 ? (
          <p className="py-3 text-center text-sm text-muted-foreground">{sp.overview.noOverdue}</p>
        ) : (
          <MonitorRows rows={data.overdue} today={data.today} late onOpen={open} />
        )}
      </Card>

      {data.waitingReview.length > 0 ? (
        <Card className="gap-2 p-(--card-pad)">
          <p className="flex items-center gap-1.5 text-sm font-semibold">
            <Clock className="size-4 text-[var(--status-warning)]" /> {sp.overview.waitingReview}
          </p>
          <MonitorRows rows={data.waitingReview} today={data.today} late={false} onOpen={open} />
        </Card>
      ) : null}
      {sheet}
    </div>
  );
}

/** Per department indicator: linked occurrences this month and their completion rate. */
export function KpiTasksTable({ rows, meta }: { rows: IndicatorTasks[]; meta: TaskMeta }) {
  const { open, sheet } = useDetail(meta);
  const [expanded, setExpanded] = React.useState<string | null>(null);
  if (rows.length === 0) return <ErpEmpty title={s.kpiEmpty} />;
  return (
    <Card className="py-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{s.indicator}</TableHead>
            <TableHead className="hidden text-right sm:table-cell">{s.weight}</TableHead>
            <TableHead className="text-right">{s.approvedOf}</TableHead>
            <TableHead className="w-44">{s.rate}</TableHead>
            <TableHead className="hidden text-right md:table-cell">{s.contributionTotal}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => {
            const isOpen = expanded === row.indicatorId;
            return (
              <TableRow key={row.indicatorId} className="align-top">
                <TableCell className="whitespace-normal">
                  <button
                    type="button"
                    aria-expanded={isOpen}
                    onClick={() => setExpanded(isOpen ? null : row.indicatorId)}
                    className="text-left font-medium underline-offset-4 hover:underline"
                  >
                    {row.name}
                  </button>
                  <p className="text-xs text-muted-foreground">
                    {row.code}
                    {row.unit ? ` · ${row.unit}` : ""}
                    {row.waitingReview > 0 ? ` · ${row.waitingReview} ${sp.overview.waitingReview.toLowerCase()}` : ""}
                    {row.overdue > 0 ? <span className="text-destructive"> · {row.overdue} {sp.overview.overdue.toLowerCase()}</span> : null}
                  </p>
                  {isOpen ? (
                    <ul className="mt-2 divide-y rounded-xl bg-muted/50">
                      {row.tasks.length === 0 ? (
                        <li className="px-(--card-pad) py-3 text-xs text-muted-foreground">{s.noLinkedTasks}</li>
                      ) : (
                        row.tasks.map((t) => <TaskRow key={t.occurrenceId} item={t} onOpen={open} showAssignee />)
                      )}
                    </ul>
                  ) : null}
                </TableCell>
                <TableCell className="hidden text-right tabular sm:table-cell">
                  {row.weight == null ? <span className="text-xs text-muted-foreground">{s.noWeight}</span> : formatNumber(row.weight)}
                </TableCell>
                <TableCell className="text-right tabular">
                  {row.approved}/{row.total}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Progress value={(row.rate ?? 0) * 100} className="h-2 flex-1" aria-label={`${s.rate} ${row.name}`} />
                    <span className="w-10 text-right text-xs tabular">{ratio(row.rate)}</span>
                  </div>
                </TableCell>
                <TableCell className="hidden text-right tabular md:table-cell">
                  {formatNumber(row.contributionTotal, 2)}
                  {row.expectedTotal > 0 ? <span className="text-xs text-muted-foreground"> / {formatNumber(row.expectedTotal, 2)}</span> : null}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      {sheet}
    </Card>
  );
}
