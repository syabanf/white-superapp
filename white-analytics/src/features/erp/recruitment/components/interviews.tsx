"use client";

import * as React from "react";
import Link from "next/link";
import type { z } from "zod";
import { CalendarPlus, ClipboardCheck, MapPin, Pencil, Plus, Trash2, Video, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Fld, FormDialog, FormFooter, ReasonDialog, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { SearchableSelect, type SearchOption } from "@/features/erp/shared/searchable-select";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import { cancelInterviewAction, saveInterviewAction, submitScorecardAction } from "../actions";
import { averageScore, wibInstant, wibParts } from "../lib";
import type { interviewSchema } from "../schemas";
import { INTERVIEW_MODE, INTERVIEW_REC, INTERVIEW_STATUS, INTERVIEW_TYPE, kinds, rs } from "../strings";
import type { Interview, ScoreItem } from "../types";

const iv = rs.interviews;
const opts = (m: Record<string, string>) => Object.entries(m).map(([value, label]) => ({ value, label }));

export type InterviewLookups = { candidates: SearchOption[]; employees: SearchOption[] };

/** Interview cards with their actions. */
export function InterviewList({
  interviews,
  canWrite,
  lookups,
  showCandidate = true,
}: {
  interviews: Interview[];
  canWrite: boolean;
  lookups: InterviewLookups;
  showCandidate?: boolean;
}) {
  if (!interviews.length) return <ErpEmpty compact title={iv.empty} />;
  return (
    <div className="flex flex-col gap-3">
      {interviews.map((item) => (
        <InterviewCard key={item.id} item={item} canWrite={canWrite} lookups={lookups} showCandidate={showCandidate} />
      ))}
    </div>
  );
}

function InterviewCard({
  item,
  canWrite,
  lookups,
  showCandidate,
}: {
  item: Interview;
  canWrite: boolean;
  lookups: InterviewLookups;
  showCandidate: boolean;
}) {
  const scheduled = item.status === "scheduled";
  return (
    <Card className="flex flex-col gap-3 p-4 text-sm">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2 font-medium">
            <span className="tabular">{formatDateTime(item.interviewDate)}</span>
            <ErpStatusBadge status={item.status} labels={INTERVIEW_STATUS} kinds={kinds.interview} />
          </div>
          {showCandidate ? (
            <Link href={`/erp/hr/recruitment/candidates/${item.candidateId}?tab=interviews`} className="mt-0.5 block hover:underline">
              {item.candidateName}
              <span className="text-muted-foreground"> · {item.jobOpeningTitle || rs.none}</span>
            </Link>
          ) : null}
          <p className="mt-0.5 text-xs text-muted-foreground">
            {INTERVIEW_TYPE[item.type] ?? item.type} · {item.interviewerName || rs.none} · {rs.minutes(item.durationMinutes)}
          </p>
        </div>
        {canWrite ? (
          <div className="flex flex-wrap gap-1.5">
            {scheduled ? (
              <>
                <ScorecardButton item={item} />
                <FormDialog
                  title={iv.reschedule}
                  wide
                  trigger={
                    <Button size="xs" variant="outline">
                      <Pencil className="size-3.5" /> {iv.reschedule}
                    </Button>
                  }
                >
                  {(close) => <InterviewForm item={item} lookups={lookups} close={close} />}
                </FormDialog>
                <ReasonDialog
                  title={iv.cancelTitle}
                  label={iv.cancelReason}
                  submitLabel={iv.cancel}
                  success={iv.cancelled}
                  onSubmit={(reason) => cancelInterviewAction(item.id, reason)}
                  trigger={
                    <Button size="xs" variant="ghost" className="text-destructive">
                      <X className="size-3.5" /> {iv.cancel}
                    </Button>
                  }
                />
              </>
            ) : null}
            {item.status === "completed" ? <ScorecardButton item={item} /> : null}
          </div>
        ) : null}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
        {item.mode === "online" ? (
          <span className="flex items-center gap-1">
            <Video className="size-3.5" />
            {item.meetingLink ? (
              <a href={item.meetingLink} target="_blank" rel="noreferrer" className="text-foreground underline-offset-2 hover:underline">
                {item.meetingLink}
              </a>
            ) : (
              INTERVIEW_MODE.online
            )}
          </span>
        ) : (
          <span className="flex items-center gap-1">
            <MapPin className="size-3.5" /> {item.location || INTERVIEW_MODE.offline}
          </span>
        )}
      </div>
      {item.notes ? <p className="text-xs whitespace-pre-wrap">{item.notes}</p> : null}
      {item.cancelReason ? <p className="text-xs text-muted-foreground">{iv.cancelReason}: {item.cancelReason}</p> : null}
      {item.scorecard.length ? (
        <div className="rounded-lg bg-muted/40 p-3 text-xs">
          <div className="mb-2 flex flex-wrap items-center gap-2 font-medium">
            {iv.overall}: <span className="tabular">{item.overallScore ?? rs.none}</span>
            {item.recommendation ? (
              <ErpStatusBadge status={item.recommendation} labels={INTERVIEW_REC} kinds={kinds.rec} />
            ) : null}
          </div>
          <ul className="grid gap-1 sm:grid-cols-2">
            {item.scorecard.map((s) => (
              <li key={s.criterion}>
                {s.criterion}: <span className="font-medium tabular">{s.score}</span>
                {s.note ? <span className="text-muted-foreground"> · {s.note}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </Card>
  );
}

export function ScheduleButton({ lookups, candidateId }: { lookups: InterviewLookups; candidateId?: string }) {
  return (
    <FormDialog
      title={iv.schedule}
      wide
      trigger={
        <Button size="sm">
          <CalendarPlus className="size-4" /> {iv.schedule}
        </Button>
      }
    >
      {(close) => <InterviewForm lookups={lookups} close={close} candidateId={candidateId} />}
    </FormDialog>
  );
}

type FormValues = z.output<typeof interviewSchema>;

function InterviewForm({
  item,
  lookups,
  close,
  candidateId,
}: {
  item?: Interview;
  lookups: InterviewLookups;
  close: () => void;
  candidateId?: string;
}) {
  const parts = wibParts(item?.interviewDate ?? null);
  const [day, setDay] = React.useState(parts.day);
  const [time, setTime] = React.useState(parts.time || "10:00");
  const [v, setV] = React.useState<Omit<FormValues, "interviewDate">>({
    candidateId: item?.candidateId ?? candidateId ?? "",
    interviewerEmployeeId: item?.interviewerEmployeeId ?? "",
    interviewerName: item?.interviewerEmployeeId ? "" : (item?.interviewerName ?? ""),
    durationMinutes: item?.durationMinutes ?? 60,
    type: (item?.type as FormValues["type"]) ?? "hrd",
    mode: (item?.mode as FormValues["mode"]) ?? "online",
    meetingLink: item?.meetingLink ?? "",
    location: item?.location ?? "",
    notes: item?.notes ?? "",
  });
  const set = <K extends keyof typeof v>(k: K, val: (typeof v)[K]) => setV((p) => ({ ...p, [k]: val }));
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  const fixedCandidate = Boolean(item || candidateId);

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveInterviewAction(item?.id ?? null, { ...v, interviewDate: wibInstant(day, time) ?? "" }), {
          success: item ? iv.updated : iv.scheduled,
          onOk: close,
          onFail,
        });
      }}
    >
      {fixedCandidate ? null : (
        <Fld label={iv.candidate} htmlFor="iv-cand" error={errors?.candidateId} className="sm:col-span-2">
          <SearchableSelect id="iv-cand" options={lookups.candidates} value={v.candidateId} onChange={(x) => set("candidateId", x)} placeholder={iv.pickCandidate} />
        </Fld>
      )}
      <Fld label={iv.date} htmlFor="iv-date" error={errors?.interviewDate}>
        <DatePicker id="iv-date" value={day} onChange={setDay} />
      </Fld>
      <Fld label={iv.time} htmlFor="iv-time">
        <Input id="iv-time" type="time" value={time} onChange={(e) => setTime(e.target.value)} />
      </Fld>
      <Fld label={iv.type} htmlFor="iv-type">
        <SimpleSelect id="iv-type" value={v.type} onChange={(x) => set("type", x as FormValues["type"])} options={opts(INTERVIEW_TYPE)} />
      </Fld>
      <Fld label={iv.duration} htmlFor="iv-dur" error={errors?.durationMinutes}>
        <Input id="iv-dur" type="number" min={15} step={15} value={v.durationMinutes} onChange={(e) => set("durationMinutes", Number(e.target.value))} />
      </Fld>
      <Fld label={iv.interviewer} htmlFor="iv-emp" error={errors?.interviewerEmployeeId}>
        <SearchableSelect
          id="iv-emp"
          options={lookups.employees}
          value={v.interviewerEmployeeId}
          onChange={(x) => set("interviewerEmployeeId", x)}
          placeholder={iv.pickInterviewer}
          clearable
        />
      </Fld>
      <Fld label={iv.interviewerName} htmlFor="iv-name" error={errors?.interviewerName}>
        <Input id="iv-name" value={v.interviewerName} onChange={(e) => set("interviewerName", e.target.value)} disabled={Boolean(v.interviewerEmployeeId)} />
      </Fld>
      <Fld label={iv.mode} htmlFor="iv-mode">
        <SimpleSelect id="iv-mode" value={v.mode} onChange={(x) => set("mode", x as FormValues["mode"])} options={opts(INTERVIEW_MODE)} />
      </Fld>
      {v.mode === "online" ? (
        <Fld label={iv.meetingLink} htmlFor="iv-link" error={errors?.meetingLink}>
          <Input id="iv-link" type="url" value={v.meetingLink} onChange={(e) => set("meetingLink", e.target.value)} placeholder="https://meet.google.com/…" />
        </Fld>
      ) : (
        <Fld label={iv.location} htmlFor="iv-loc" error={errors?.location}>
          <Input id="iv-loc" value={v.location} onChange={(e) => set("location", e.target.value)} />
        </Fld>
      )}
      <Fld label={iv.notes} htmlFor="iv-notes" className="sm:col-span-2">
        <Textarea id="iv-notes" rows={2} value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function ScorecardButton({ item }: { item: Interview }) {
  return (
    <FormDialog
      title={iv.scorecardTitle}
      wide
      trigger={
        <Button size="xs" variant={item.status === "scheduled" ? "default" : "outline"}>
          <ClipboardCheck className="size-3.5" /> {iv.scorecard}
        </Button>
      }
    >
      {(close) => <ScorecardForm item={item} close={close} />}
    </FormDialog>
  );
}

function ScorecardForm({ item, close }: { item: Interview; close: () => void }) {
  const [rows, setRows] = React.useState<ScoreItem[]>(
    item.scorecard.length ? item.scorecard : iv.criteria.map((criterion) => ({ criterion, score: 3, note: "" })),
  );
  const [rec, setRec] = React.useState(item.recommendation || "proceed");
  const [notes, setNotes] = React.useState(item.notes);
  const { run, pending } = useErpAction();
  const patch = (i: number, p: Partial<ScoreItem>) => setRows((r) => r.map((x, j) => (j === i ? { ...x, ...p } : x)));
  const avg = averageScore(rows.map((r) => r.score));

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            submitScorecardAction(item.id, {
              scorecard: rows.filter((r) => r.criterion.trim()),
              recommendation: rec as "proceed" | "pool" | "reject",
              notes,
            }),
          { success: iv.scorecardSaved, onOk: close },
        );
      }}
    >
      <div className="flex flex-col gap-2">
        {rows.map((r, i) => (
          <div key={i} className="grid grid-cols-[1fr_auto_auto] items-start gap-2 sm:grid-cols-[1fr_auto_1fr_auto]">
            <Input value={r.criterion} onChange={(e) => patch(i, { criterion: e.target.value })} aria-label={iv.criterion} />
            <div className="flex gap-1" role="radiogroup" aria-label={`${iv.score} ${r.criterion}`}>
              {[1, 2, 3, 4, 5].map((n) => (
                <button
                  key={n}
                  type="button"
                  role="radio"
                  aria-checked={r.score === n}
                  onClick={() => patch(i, { score: n })}
                  className={cn(
                    "size-8 rounded-md border text-xs font-medium tabular transition-colors",
                    r.score === n ? "border-brand bg-brand text-white" : "hover:bg-accent",
                  )}
                >
                  {n}
                </button>
              ))}
            </div>
            <Input
              value={r.note}
              onChange={(e) => patch(i, { note: e.target.value })}
              placeholder={iv.notes}
              aria-label={`${iv.notes} ${r.criterion}`}
              className="col-span-2 row-start-2 sm:col-span-1 sm:row-start-auto"
            />
            <Button type="button" size="icon-sm" variant="ghost" onClick={() => setRows((x) => x.filter((_, j) => j !== i))} aria-label={rs.delete} disabled={rows.length <= 1}>
              <Trash2 className="size-4" />
            </Button>
          </div>
        ))}
        <Button type="button" variant="ghost" size="sm" className="self-start" onClick={() => setRows((x) => [...x, { criterion: "", score: 3, note: "" }])}>
          <Plus className="size-4" /> {iv.addCriterion}
        </Button>
      </div>
      <p className="text-sm">
        {iv.overall}: <span className="font-semibold tabular">{avg ?? rs.none}</span>
      </p>
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={iv.recommendation} htmlFor="sc-rec">
          <SimpleSelect id="sc-rec" value={rec} onChange={setRec} options={opts(INTERVIEW_REC)} />
        </Fld>
        <Fld label={iv.notes} htmlFor="sc-notes">
          <Textarea id="sc-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

const WEEKDAYS = ["Sen", "Sel", "Rab", "Kam", "Jum", "Sab", "Min"];

/** Month grid of interviews (WIB days); phones get an agenda list instead. */
export function InterviewCalendar({ interviews, month }: { interviews: Interview[]; month: string }) {
  const [y, m] = month.split("-").map(Number);
  const first = new Date(Date.UTC(y, m - 1, 1));
  const days = new Date(Date.UTC(y, m, 0)).getUTCDate();
  const lead = (first.getUTCDay() + 6) % 7;
  const byDay = new Map<string, Interview[]>();
  for (const item of interviews) {
    const { day } = wibParts(item.interviewDate);
    byDay.set(day, [...(byDay.get(day) ?? []), item]);
  }
  const key = (d: number) => `${month}-${String(d).padStart(2, "0")}`;
  const today = wibParts(new Date().toISOString()).day;

  return (
    <>
      <div className="hidden overflow-hidden rounded-xl border md:block">
        <div className="grid grid-cols-7 border-b bg-muted/40 text-center text-xs font-medium text-muted-foreground">
          {WEEKDAYS.map((w) => (
            <div key={w} className="py-2">
              {w}
            </div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: lead }, (_, i) => (
            <div key={`l${i}`} className="min-h-24 border-r border-b bg-muted/20" />
          ))}
          {Array.from({ length: days }, (_, i) => {
            const d = i + 1;
            const list = byDay.get(key(d)) ?? [];
            return (
              <div key={d} className="min-h-24 border-r border-b p-1.5">
                <div className={cn("mb-1 text-xs tabular", key(d) === today ? "font-semibold text-brand" : "text-muted-foreground")}>{d}</div>
                <ul className="flex flex-col gap-1">
                  {list.map((item) => (
                    <li key={item.id}>
                      <Link
                        href={`/erp/hr/recruitment/candidates/${item.candidateId}?tab=interviews`}
                        className={cn(
                          "block truncate rounded px-1.5 py-0.5 text-[11px] hover:opacity-80",
                          item.status === "cancelled" ? "bg-muted text-muted-foreground line-through" : "bg-brand/10 text-foreground",
                        )}
                        title={`${item.candidateName} · ${item.interviewerName}`}
                      >
                        {wibParts(item.interviewDate).time} {item.candidateName}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            );
          })}
        </div>
      </div>
      <div className="flex flex-col gap-3 md:hidden">
        {[...byDay.entries()]
          .sort(([a], [b]) => a.localeCompare(b))
          .map(([day, list]) => (
            <div key={day}>
              <p className="mb-1 text-xs font-medium text-muted-foreground">{day}</p>
              <ul className="divide-y rounded-xl border">
                {list.map((item) => (
                  <li key={item.id}>
                    <Link href={`/erp/hr/recruitment/candidates/${item.candidateId}?tab=interviews`} className="flex justify-between gap-2 px-3 py-2 text-sm">
                      <span className="truncate">{item.candidateName}</span>
                      <span className="text-xs text-muted-foreground tabular">{wibParts(item.interviewDate).time}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        {byDay.size === 0 ? <ErpEmpty compact title={iv.empty} /> : null}
      </div>
    </>
  );
}
