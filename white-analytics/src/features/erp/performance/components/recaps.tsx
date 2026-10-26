"use client";

import * as React from "react";
import { CheckCircle2, Circle, History, Loader2, RefreshCw, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDate, formatDateLong, formatDateTime, formatDecimal, formatNumber } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { regenerateMonthlyRecapAction } from "../actions";
import { ratio } from "../lib";
import { sp } from "../strings";
import type { DailyRecap, Department, MonthlyRecap, Person } from "../types";
import { FormRow, PriorityBadge, Score, TaskStatusBadge } from "./common";
import { ENTRY_KINDS } from "./logbook";
import { useResource } from "./use-resource";

const c = sp.recaps;

// ── Daily ─────────────────────────────────────────────────────

/** Summary, tasks with subtasks and notes, logbook and personal notes of one day. */
export function DailyRecapView({ recap, hideNotes }: { recap: DailyRecap; hideNotes?: boolean }) {
  return (
    <div className="flex flex-col gap-4">
      {!recap.saved ? <p className="rounded-xl bg-muted/60 px-3 py-2 text-xs text-muted-foreground">{c.previewHint}</p> : null}
      <section className="flex flex-col gap-1.5">
        <h3 className="flex items-center gap-1.5 text-sm font-semibold">
          <Sparkles className="size-4 text-brand" /> {c.summary}
        </h3>
        <p className="text-sm whitespace-pre-wrap">{recap.generatedSummary || <span className="text-muted-foreground">{c.noSummary}</span>}</p>
        {recap.generatedAt ? <p className="text-xs text-muted-foreground">{c.generatedAt(formatDateTime(recap.generatedAt))}</p> : null}
      </section>
      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{c.tasks}</h3>
        {recap.tasks.length === 0 ? (
          <p className="text-sm text-muted-foreground">{c.noTasks}</p>
        ) : (
          <ul className="flex flex-col divide-y rounded-xl border">
            {recap.tasks.map((t) => (
              <li key={t.occurrenceId} className="flex flex-col gap-1 px-3 py-2.5">
                <div className="flex flex-wrap items-center gap-2">
                  {t.completed ? (
                    <CheckCircle2 className="size-4 text-[var(--status-good)]" />
                  ) : (
                    <Circle className="size-4 text-muted-foreground" />
                  )}
                  <span className="text-sm font-medium">{t.title}</span>
                  <PriorityBadge priority={t.priority} />
                  <TaskStatusBadge status={t.status} />
                  <span className="ml-auto text-xs text-muted-foreground tabular">{t.progress}%</span>
                </div>
                {t.subtasks.length > 0 ? (
                  <ul className="flex flex-wrap gap-x-3 gap-y-0.5 pl-6 text-xs text-muted-foreground">
                    {t.subtasks.map((s, i) => (
                      <li key={i} className={s.checked ? "line-through" : undefined}>
                        {s.title}
                      </li>
                    ))}
                  </ul>
                ) : null}
                {t.notes.map((n, i) => (
                  <p key={i} className="pl-6 text-xs italic text-muted-foreground">
                    “{n}”
                  </p>
                ))}
              </li>
            ))}
          </ul>
        )}
      </section>
      {recap.logbook.length > 0 ? (
        <section className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold">{c.logbook}</h3>
          <ul className="flex flex-col gap-1.5">
            {recap.logbook.map((lb) => (
              <li key={lb.entryId} className="flex flex-wrap items-center gap-2 text-sm">
                <span className="min-w-0 flex-1 truncate">{lb.title}</span>
                <ErpStatusBadge status={lb.status} labels={sp.logbook.entryStatus} kinds={ENTRY_KINDS} />
                <span className="text-xs text-muted-foreground tabular">
                  {sp.logbook.pctDone(Math.round(lb.completion), Math.round(lb.kpiScore))}
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {!hideNotes && recap.notes ? (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-sm font-semibold">{c.notes}</h3>
          <p className="rounded-xl bg-muted/50 p-3 text-sm whitespace-pre-wrap">{recap.notes}</p>
        </section>
      ) : null}
    </div>
  );
}

const doneCount = (r: DailyRecap) => r.tasks.filter((t) => t.completed).length;

export function TeamDailyRecaps({ recaps }: { recaps: DailyRecap[] }) {
  const [open, setOpen] = React.useState<DailyRecap | null>(null);
  const [historyOf, setHistoryOf] = React.useState<DailyRecap | null>(null);
  if (recaps.length === 0) return <ErpEmpty title={c.noTeamRecaps} />;
  return (
    <>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        {recaps.map((r) => (
          <Card key={r.employeeId} className="gap-2 p-4">
            <button type="button" onClick={() => setOpen(r)} className="text-left outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate font-semibold">{r.employeeName}</p>
                  <p className="truncate text-xs text-muted-foreground">{r.departmentName}</p>
                </div>
                {!r.saved ? <span className="text-[11px] text-muted-foreground">{c.preview}</span> : null}
              </div>
              <p className="mt-2 text-xs text-muted-foreground">
                {c.tasksDone(doneCount(r), r.tasks.length)}
                {r.logbook.length ? ` · ${c.logbookCount(r.logbook.length)}` : ""}
              </p>
              <p className="mt-1.5 line-clamp-3 text-sm">{r.generatedSummary || <span className="text-muted-foreground">{c.noSummary}</span>}</p>
            </button>
            <Button variant="ghost" size="xs" className="self-start" onClick={() => setHistoryOf(r)}>
              <History /> {c.history}
            </Button>
          </Card>
        ))}
      </div>
      <Dialog open={!!open} onOpenChange={(o) => !o && setOpen(null)}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{open?.employeeName}</DialogTitle>
            <DialogDescription>{open ? formatDateLong(open.recapDate) : null}</DialogDescription>
          </DialogHeader>
          {open ? <DailyRecapView recap={open} /> : null}
        </DialogContent>
      </Dialog>
      <RecapHistoryDialog employeeId={historyOf?.employeeId ?? null} name={historyOf?.employeeName ?? ""} onClose={() => setHistoryOf(null)} />
    </>
  );
}

/** Past daily recaps of one employee, newest first. */
export function RecapHistoryList({ recaps }: { recaps: DailyRecap[] }) {
  const [open, setOpen] = React.useState<string | null>(null);
  if (recaps.length === 0) return <p className="text-sm text-muted-foreground">{c.noHistory}</p>;
  return (
    <ul className="flex flex-col divide-y rounded-xl border">
      {recaps.map((r) => {
        const key = r.id || r.recapDate;
        const isOpen = open === key;
        return (
          <li key={key} className="px-3 py-2.5">
            <button
              type="button"
              aria-expanded={isOpen}
              onClick={() => setOpen(isOpen ? null : key)}
              className="flex w-full items-center justify-between gap-2 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <span className="text-sm font-medium">{formatDate(r.recapDate)}</span>
              <span className="text-xs text-muted-foreground">{c.tasksDone(doneCount(r), r.tasks.length)}</span>
            </button>
            {isOpen ? (
              <div className="mt-3">
                <DailyRecapView recap={r} />
              </div>
            ) : (
              <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">{r.generatedSummary}</p>
            )}
          </li>
        );
      })}
    </ul>
  );
}

function RecapHistoryDialog({ employeeId, name, onClose }: { employeeId: string | null; name: string; onClose: () => void }) {
  const { data, error } = useResource<{ recaps: DailyRecap[] }>(
    employeeId ? `/v1/performance/recaps/daily/history?employeeId=${encodeURIComponent(employeeId)}&limit=30` : null,
  );
  const recaps = data?.recaps ?? null;
  return (
    <Dialog open={!!employeeId} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{c.historyOf(name)}</DialogTitle>
          {error ? <DialogDescription className="text-destructive">{error}</DialogDescription> : null}
        </DialogHeader>
        {recaps ? <RecapHistoryList recaps={recaps} /> : !error ? <Loader2 className="size-4 animate-spin" /> : null}
      </DialogContent>
    </Dialog>
  );
}

// ── Monthly ───────────────────────────────────────────────────

function MetricsGrid({ recap }: { recap: MonthlyRecap }) {
  const m = recap.metrics;
  const mt = c.metrics;
  return (
    <KpiGrid cols={4}>
      <KpiTile size="sm" label={mt.tasks} value={`${m.tasksDone}/${m.tasksPlanned}`} caption={`${mt.completion} ${ratio(m.completionRate)}`} />
      <KpiTile size="sm" label={mt.onTime} value={ratio(m.onTimeRate)} caption={`${mt.overdue} ${m.tasksOverdue}`} />
      <KpiTile size="sm" label={mt.kpi} value={m.kpiScore == null ? "–" : formatDecimal(m.kpiScore, 1)} />
      <KpiTile size="sm" label={mt.logbook} value={ratio(m.logbookCompliance)} caption={sp.recaps.logbookCount(m.logbookEntries)} />
      {m.hasAttendance ? (
        <>
          <KpiTile size="sm" label={mt.attendance} value={c.presentOf(m.presentDays, m.workingDays)} caption={`${mt.late} ${m.lateDays}`} />
          <KpiTile size="sm" label={mt.absent} value={c.days(m.absentDays)} caption={`${mt.leave} ${m.leaveDays}`} />
          <KpiTile size="sm" label={mt.overtime} value={c.hours(formatNumber(m.overtimeHours, 1))} />
        </>
      ) : null}
      {recap.scope === "department" ? <KpiTile size="sm" label={mt.members} value={formatNumber(m.members)} /> : null}
      <KpiTile size="sm" label={mt.improvements} value={formatNumber(m.improvementsOpen)} />
    </KpiGrid>
  );
}

export function MonthlyRecaps({
  recaps,
  canGenerate,
  people,
  departments,
  scope,
  period,
}: {
  recaps: MonthlyRecap[];
  canGenerate: boolean;
  people: Person[];
  departments: Department[];
  scope: "employee" | "department";
  period: { periodYear: number; periodMonth: number };
}) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  const [generating, setGenerating] = React.useState(false);
  const open = recaps.find((r) => r.id === openId) ?? null;
  return (
    <div className="flex flex-col gap-3">
      {canGenerate ? (
        <Button size="sm" variant="secondary" className="self-start" onClick={() => setGenerating(true)}>
          <Sparkles className="size-4" /> {c.generate}
        </Button>
      ) : null}
      {recaps.length === 0 ? (
        <ErpEmpty title={c.noMonthly} />
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {recaps.map((r) => (
            <button
              key={r.id}
              type="button"
              onClick={() => setOpenId(r.id)}
              className="lift flex flex-col gap-2 rounded-2xl bg-card p-4 text-left shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate font-semibold">{r.subjectName}</p>
                  <p className="truncate text-xs text-muted-foreground">{r.scope === "employee" ? r.departmentName : c.scopeDepartment}</p>
                </div>
                <Score value={r.metrics.kpiScore} className="text-xl" />
              </div>
              <p className="text-xs text-muted-foreground">
                {c.metrics.tasks} {r.metrics.tasksDone}/{r.metrics.tasksPlanned} · {c.metrics.onTime} {ratio(r.metrics.onTimeRate)} ·{" "}
                {c.metrics.logbook} {ratio(r.metrics.logbookCompliance)}
              </p>
              <p className="line-clamp-3 text-sm">{r.narrative}</p>
            </button>
          ))}
        </div>
      )}
      <Dialog open={!!open} onOpenChange={(o) => !o && setOpenId(null)}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-[calc(100%-2rem)] lg:max-w-3xl">
          {open ? <MonthlyRecapView recap={open} canRegenerate={canGenerate} /> : null}
        </DialogContent>
      </Dialog>
      {generating ? (
        <GenerateDialog scope={scope} people={people} departments={departments} period={period} onClose={() => setGenerating(false)} />
      ) : null}
    </div>
  );
}

function MonthlyRecapView({ recap, canRegenerate }: { recap: MonthlyRecap; canRegenerate: boolean }) {
  const { pending, run } = useErpAction();
  return (
    <div className="flex flex-col gap-5">
      <DialogHeader>
        <DialogTitle>{recap.subjectName}</DialogTitle>
        <DialogDescription>
          {sp.monthLabel(recap.periodYear, recap.periodMonth)}
          {recap.scope === "employee" && recap.departmentName ? ` · ${recap.departmentName}` : ""}
          {recap.generatedAt ? ` · ${c.generatedAt(formatDateTime(recap.generatedAt))}` : ""}
        </DialogDescription>
      </DialogHeader>
      <MetricsGrid recap={recap} />
      <section className="flex flex-col gap-1.5">
        <h3 className="text-sm font-semibold">{c.narrative}</h3>
        <p className="text-sm leading-relaxed whitespace-pre-wrap">{recap.narrative}</p>
      </section>
      {recap.accomplishments.length > 0 ? (
        <section className="flex flex-col gap-1.5">
          <h3 className="text-sm font-semibold">{c.accomplishments}</h3>
          <ul className="list-disc pl-5 text-sm">
            {recap.accomplishments.map((a) => (
              <li key={a}>{a}</li>
            ))}
          </ul>
        </section>
      ) : null}
      <div className="flex flex-wrap justify-end gap-2 border-t pt-4">
        {canRegenerate ? (
          <Button
            variant="outline"
            size="sm"
            disabled={pending}
            onClick={() =>
              run(
                () =>
                  regenerateMonthlyRecapAction({
                    scope: recap.scope,
                    subjectId: recap.subjectId,
                    periodYear: recap.periodYear,
                    periodMonth: recap.periodMonth,
                  }),
                { onOk: (r) => toast.success(c.regenerated(r.suggestionsFiled)) },
              )
            }
          >
            <RefreshCw className={cn("size-3.5", pending && "animate-spin")} /> {pending ? c.regenerating : c.regenerate}
          </Button>
        ) : null}
        <DownloadButton
          href={`/api/erp/v1/performance/recaps/monthly/${encodeURIComponent(recap.id)}/pdf`}
          fileName={`rekap-${recap.subjectName}-${recap.periodYear}-${recap.periodMonth}.pdf`}
        >
          {c.pdf}
        </DownloadButton>
      </div>
    </div>
  );
}

function GenerateDialog({
  scope,
  people,
  departments,
  period,
  onClose,
}: {
  scope: "employee" | "department";
  people: Person[];
  departments: Department[];
  period: { periodYear: number; periodMonth: number };
  onClose: () => void;
}) {
  const { pending, run } = useErpAction();
  const [subjectId, setSubjectId] = React.useState("");
  const options = scope === "employee" ? people : departments;
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{c.generateTitle}</DialogTitle>
          <DialogDescription>
            {sp.monthLabel(period.periodYear, period.periodMonth)} · {scope === "employee" ? c.scopeEmployee : c.scopeDepartment}
          </DialogDescription>
        </DialogHeader>
        <FormRow label={c.subject}>
          <Select value={subjectId} onValueChange={setSubjectId}>
            <SelectTrigger className="w-full" aria-label={c.subject}>
              <SelectValue placeholder={c.pickSubject} />
            </SelectTrigger>
            <SelectContent>
              {options.map((o) => (
                <SelectItem key={o.id} value={o.id}>
                  {o.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </FormRow>
        <DialogFooter>
          <Button
            disabled={pending || !subjectId}
            onClick={() =>
              run(() => regenerateMonthlyRecapAction({ scope, subjectId, ...period }), {
                onOk: (r) => {
                  toast.success(c.regenerated(r.suggestionsFiled));
                  onClose();
                },
              })
            }
          >
            {pending ? c.regenerating : c.generate}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
