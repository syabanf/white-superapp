"use client";

import * as React from "react";
import { Check, Loader2, Lock, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { formatDate, formatDecimal } from "@/lib/format";
import { cn } from "@/lib/utils";
import { closeReviewCycleAction, openReviewCycleAction, reviewAction } from "../actions";
import { scoreTone } from "../lib";
import { sp } from "../strings";
import type { RealtimeKpi, Review, ReviewAction, ReviewCycle, ReviewDetail, Scorecard } from "../types";
import { ConfirmAction, DRAFT_FINAL_KINDS, DRAFT_FINAL_LABELS, FormRow, Score } from "./common";
import { useResource } from "./use-resource";

const r = sp.reviews;

function CategoryBadge({ category }: { category?: string }) {
  if (!category) return null;
  return <span className="rounded-full bg-muted px-2 py-0.5 text-xs font-semibold">{category}</span>;
}

function MonthChip({ month, score }: { month: number; score?: number }) {
  return (
    <span className="flex-1 rounded-lg bg-muted/60 px-2 py-1 text-center text-xs">
      <span className="block text-[10px] text-muted-foreground">{sp.monthShort(month)}</span>
      <Score value={score} digits={0} />
    </span>
  );
}

function PersonCard({
  name,
  department,
  score,
  children,
  onClick,
}: {
  name: string;
  department: string;
  score?: number;
  children: React.ReactNode;
  onClick?: () => void;
}) {
  const body = (
    <>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold">{name}</p>
          <p className="truncate text-xs text-muted-foreground">{department || "–"}</p>
        </div>
        <Score value={score} className="text-xl" />
      </div>
      <div className="mt-2">{children}</div>
    </>
  );
  return onClick ? (
    <button
      type="button"
      onClick={onClick}
      className="lift rounded-2xl bg-card p-4 text-left shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {body}
    </button>
  ) : (
    <Card className="gap-0 p-(--card-pad)">{body}</Card>
  );
}

/** Quarter-to-date KPI average per employee. */
export function RealtimeGrid({ data }: { data: RealtimeKpi }) {
  if (data.employees.length === 0) return <ErpEmpty title={r.realtimeEmpty} />;
  return (
    <div className="grid grid-cols-1 gap-(--gap) sm:grid-cols-2 lg:grid-cols-3">
      {data.employees.map((e) => (
        <PersonCard key={e.employeeId} name={e.employeeName} department={e.departmentName} score={e.avgScore}>
          <div className="flex gap-2">
            {data.months.map((m) => (
              <MonthChip key={m} month={m} score={e.months.find((s) => s.periodMonth === m)?.score} />
            ))}
          </div>
        </PersonCard>
      ))}
    </div>
  );
}

/** Cycle picker (URL `cycle`), open and close actions for HR. */
export function CycleBar({ cycles, active, canManage }: { cycles: ReviewCycle[]; active: string; canManage: boolean }) {
  const { set } = useUrlFilters(["cycle"]);
  const { pending, run } = useErpAction();
  const current = cycles.find((c) => c.id === active);
  return (
    <div className="flex flex-wrap items-center gap-2">
      {cycles.map((c) => (
        <Button key={c.id} size="sm" variant={c.id === active ? "default" : "outline"} onClick={() => set({ cycle: c.id })}>
          {c.name}
          <span className="text-xs opacity-70">{r.finalOf(c.finalReviews, c.totalReviews)}</span>
          {c.status === "closed" ? <Lock className="size-3" /> : null}
        </Button>
      ))}
      {canManage ? <OpenCycleDialog /> : null}
      {canManage && current?.status === "open" ? (
        <ConfirmAction
          trigger={
            <Button size="sm" variant="ghost" disabled={pending}>
              <Lock className="size-4" /> {r.closeCycle}
            </Button>
          }
          title={r.closeCycleTitle}
          description={r.closeCycleBody}
          confirmLabel={r.closeCycle}
          destructive
          onConfirm={() => run(() => closeReviewCycleAction(current.id), { success: r.cycleClosed })}
        />
      ) : null}
    </div>
  );
}

function OpenCycleDialog() {
  const now = new Date();
  const [open, setOpen] = React.useState(false);
  const [year, setYear] = React.useState(String(now.getFullYear()));
  const [quarter, setQuarter] = React.useState(Math.ceil((now.getMonth() + 1) / 3));
  const [due, setDue] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          <Plus className="size-4" /> {r.openCycle}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{r.openCycleTitle}</DialogTitle>
          <DialogDescription>{r.openCycleBody}</DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-[6rem_minmax(0,1fr)] gap-3">
          <FormRow label={sp.common.year} htmlFor="cy-year">
            <Input id="cy-year" type="number" value={year} onChange={(e) => setYear(e.target.value)} />
          </FormRow>
          <FormRow label={r.quarter}>
            <div className="flex gap-1">
              {[1, 2, 3, 4].map((q) => (
                <Button key={q} type="button" size="sm" className="flex-1" variant={q === quarter ? "default" : "outline"} onClick={() => setQuarter(q)}>
                  Q{q}
                </Button>
              ))}
            </div>
          </FormRow>
          <FormRow label={r.dueDate} hint={r.dueDateHint} className="col-span-2">
            <DatePicker value={due} onChange={setDue} clearable />
          </FormRow>
        </div>
        <DialogFooter>
          <Button
            disabled={pending}
            onClick={() =>
              run(() => openReviewCycleAction({ periodYear: Number(year), periodQuarter: quarter, dueDate: due || undefined }), {
                success: r.cycleOpened,
                onOk: () => setOpen(false),
              })
            }
          >
            {r.openCycle}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ReviewsGrid({ reviews }: { reviews: Review[] }) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  if (reviews.length === 0) return <ErpEmpty title={r.noReviews} />;
  return (
    <>
      <div className="grid grid-cols-1 gap-(--gap) sm:grid-cols-2 lg:grid-cols-3">
        {reviews.map((v) => (
          <PersonCard key={v.id} name={v.employeeName} department={v.departmentName} score={v.grandTotalScore} onClick={() => setOpenId(v.id)}>
            <div className="flex flex-wrap items-center gap-1.5">
              <CategoryBadge category={v.category} />
              <ErpStatusBadge status={v.status} labels={DRAFT_FINAL_LABELS} kinds={DRAFT_FINAL_KINDS} />
              <span className="text-xs text-muted-foreground">{r.behaviorOf(v.ratedItems, v.totalItems)}</span>
              {v.employeeSignDate ? (
                <span className="inline-flex items-center gap-0.5 text-xs text-[var(--status-good)]">
                  <Check className="size-3" /> {r.signedEmployee}
                </span>
              ) : null}
              {v.reviewerSignDate ? (
                <span className="inline-flex items-center gap-0.5 text-xs text-[var(--status-good)]">
                  <Check className="size-3" /> {r.signedReviewer}
                </span>
              ) : null}
            </div>
          </PersonCard>
        ))}
      </div>
      <ReviewDialog id={openId} onClose={() => setOpenId(null)} />
    </>
  );
}

function ReviewDialog({ id, onClose }: { id: string | null; onClose: () => void }) {
  const { data: current, error, reload } = useResource<ReviewDetail>(
    id ? `/v1/performance/reviews/${encodeURIComponent(id)}` : null,
  );
  return (
    <Dialog open={!!id} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        {error ? (
          <DialogHeader>
            <DialogTitle>{sp.common.loadFailed}</DialogTitle>
            <DialogDescription>{error}</DialogDescription>
          </DialogHeader>
        ) : !current ? (
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Loader2 className="size-4 animate-spin" /> {sp.common.loading}
            </DialogTitle>
          </DialogHeader>
        ) : (
          <ReviewBody key={current.review.id} detail={current} reload={reload} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function ReviewBody({ detail, reload }: { detail: ReviewDetail; reload: () => Promise<void> }) {
  const { review } = detail;
  const { pending, run } = useErpAction();
  const [selfDraft, setSelfDraft] = React.useState(review.selfAssessment ?? "");
  const [notesDraft, setNotesDraft] = React.useState(review.reviewerNotes ?? "");
  const [projectDraft, setProjectDraft] = React.useState(review.projectScore != null ? String(review.projectScore) : "");
  const [managerDraft, setManagerDraft] = React.useState(review.managerNotes ?? "");
  const editable = review.status !== "final";
  const act = (input: { action: ReviewAction; text?: string; itemId?: string; score?: number; projectScore?: number }, success?: string) =>
    run(() => reviewAction(review.id, input), { success: success ?? sp.common.saved, onOk: () => void reload() });

  const summary = [
    { label: r.workResult, value: review.workResultScore },
    { label: r.behavior, value: review.behavioralScore },
    { label: r.project, value: review.projectScore },
    { label: r.grandTotal, value: review.grandTotalScore },
  ];

  return (
    <div className="flex flex-col gap-5">
      <DialogHeader>
        <DialogTitle className="flex flex-wrap items-center gap-2">
          {review.employeeName}
          <CategoryBadge category={review.category} />
          <ErpStatusBadge status={review.status} labels={DRAFT_FINAL_LABELS} kinds={DRAFT_FINAL_KINDS} />
        </DialogTitle>
        <DialogDescription>
          {review.cycleName} · {review.departmentName || "–"}
        </DialogDescription>
      </DialogHeader>

      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
        {summary.map((c) => (
          <div key={c.label} className="rounded-xl bg-muted/50 p-3 text-center">
            <p className={cn("text-lg font-bold tabular", scoreTone(c.value))}>{c.value == null ? "–" : formatDecimal(c.value, 1)}</p>
            <p className="text-[11px] text-muted-foreground">{c.label}</p>
          </div>
        ))}
      </div>

      <section className="flex flex-col gap-1.5">
        <h3 className="text-sm font-semibold">{r.kpiMonths}</h3>
        {detail.kpiMonths.length === 0 ? (
          <p className="text-xs text-muted-foreground">{r.noKpiMonths}</p>
        ) : (
          <div className="flex gap-2">
            {detail.kpiMonths.map((m: Scorecard) => (
              <MonthChip key={m.id} month={m.periodMonth} score={m.score} />
            ))}
          </div>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">
          {r.behaviorTitle}
          {detail.canRate && editable ? <span className="ml-1 font-normal text-muted-foreground">· {r.tapToRate}</span> : null}
        </h3>
        {detail.items.map((item) => (
          <div key={item.id} className="rounded-xl border p-3">
            <p className="text-sm font-medium">{item.valueName}</p>
            {item.behavioralStandard ? <p className="text-xs text-muted-foreground">{item.behavioralStandard}</p> : null}
            <div className="mt-2 flex gap-2">
              {[1, 2, 3, 4, 5].map((n) => {
                const active = item.score === n;
                const canTap = detail.canRate && editable && !pending;
                return (
                  <button
                    key={n}
                    type="button"
                    disabled={!canTap}
                    aria-pressed={active}
                    title={item.scoreDescriptions[n - 1]}
                    onClick={() => act({ action: "rate_item", itemId: item.id, score: n })}
                    className={cn(
                      "flex h-11 flex-1 items-center justify-center rounded-lg border text-sm font-semibold transition-colors",
                      active ? "border-brand bg-brand text-white" : "bg-card hover:bg-muted",
                      !canTap && "opacity-60",
                    )}
                  >
                    {n}
                  </button>
                );
              })}
            </div>
            {item.score && item.scoreDescriptions[item.score - 1] ? (
              <p className="mt-1.5 text-xs text-muted-foreground">{item.scoreDescriptions[item.score - 1]}</p>
            ) : null}
          </div>
        ))}
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{r.selfAssessment}</h3>
        {detail.isOwner && editable ? (
          <>
            <Textarea value={selfDraft} onChange={(e) => setSelfDraft(e.target.value)} placeholder={r.selfPlaceholder} rows={3} aria-label={r.selfAssessment} />
            <Button size="sm" className="self-start" disabled={pending} onClick={() => act({ action: "self_assessment", text: selfDraft })}>
              {r.saveSelf}
            </Button>
          </>
        ) : (
          <p className="rounded-xl bg-muted/50 p-3 text-sm whitespace-pre-wrap">{review.selfAssessment || r.notFilled}</p>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">
          {r.reviewerNotes}
          {review.reviewerName ? <span className="font-normal text-muted-foreground"> · {review.reviewerName}</span> : null}
        </h3>
        {detail.canRate && editable ? (
          <>
            <Textarea value={notesDraft} onChange={(e) => setNotesDraft(e.target.value)} placeholder={r.reviewerPlaceholder} rows={3} aria-label={r.reviewerNotes} />
            <div className="flex flex-wrap items-end gap-2">
              <FormRow label={r.projectScore} htmlFor="rv-project" className="w-44">
                <Input id="rv-project" type="number" min={0} max={100} value={projectDraft} onChange={(e) => setProjectDraft(e.target.value)} />
              </FormRow>
              <Button
                size="sm"
                disabled={pending}
                onClick={() =>
                  act({ action: "reviewer_notes", text: notesDraft, projectScore: projectDraft === "" ? undefined : Number(projectDraft) })
                }
              >
                {r.saveNotes}
              </Button>
            </div>
          </>
        ) : (
          <p className="rounded-xl bg-muted/50 p-3 text-sm whitespace-pre-wrap">{review.reviewerNotes || r.notFilled}</p>
        )}
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{r.managerNotes}</h3>
        {detail.canFinalize && editable ? (
          <>
            <Textarea value={managerDraft} onChange={(e) => setManagerDraft(e.target.value)} rows={2} aria-label={r.managerNotes} />
            <Button size="sm" variant="secondary" className="self-start" disabled={pending} onClick={() => act({ action: "manager_notes", text: managerDraft })}>
              {r.saveNotes}
            </Button>
          </>
        ) : (
          <p className="rounded-xl bg-muted/50 p-3 text-sm whitespace-pre-wrap">{review.managerNotes || r.notFilled}</p>
        )}
      </section>

      <div className="flex flex-wrap items-center gap-2 border-t pt-4">
        {detail.isOwner && !review.employeeSignDate ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => act({ action: "sign" })}>
            {r.signEmployee}
          </Button>
        ) : null}
        {detail.canRate && !detail.isOwner && !review.reviewerSignDate ? (
          <Button size="sm" variant="outline" disabled={pending} onClick={() => act({ action: "sign" })}>
            {r.signReviewer}
          </Button>
        ) : null}
        {review.employeeSignDate ? (
          <span className="inline-flex items-center gap-1 text-xs text-[var(--status-good)]">
            <Check className="size-3" /> {r.signedOn(r.employee, formatDate(review.employeeSignDate))}
          </span>
        ) : null}
        {review.reviewerSignDate ? (
          <span className="inline-flex items-center gap-1 text-xs text-[var(--status-good)]">
            <Check className="size-3" /> {r.signedOn(r.reviewer, formatDate(review.reviewerSignDate))}
          </span>
        ) : null}
        {detail.canFinalize && editable ? (
          <div className="ml-auto">
            <ConfirmAction
              trigger={<Button size="sm">{r.finalize}</Button>}
              title={r.finalizeTitle}
              description={r.finalizeBody}
              confirmLabel={r.finalize}
              onConfirm={() => act({ action: "finalize" }, r.finalized)}
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}
