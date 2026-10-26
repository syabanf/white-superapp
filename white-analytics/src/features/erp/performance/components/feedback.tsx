"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Calculator, Loader2, Pencil, Plus, Trash2, Users, Wand2 } from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { StatusKind } from "@/components/dashboard/status-badge";
import type { ColMeta } from "@/components/dashboard/data-table";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { formatDate, formatDecimal } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  autoAssignAction,
  createAssignmentAction,
  decideAssignmentAction,
  deleteAssignmentAction,
  deleteFeedbackCycleAction,
  generateSummariesAction,
  saveFeedbackCycleAction,
  submitResponsesAction,
  updateSummaryAction,
} from "../actions";
import { lines } from "../lib";
import { sp } from "../strings";
import type { Assignment, AssignmentDetail, FeedbackCycle, FeedbackSummary, Person, Relationship } from "../types";
import { ConfirmAction, FormRow, Score } from "./common";
import { useResource } from "./use-resource";

const f = sp.feedback;

const CYCLE_KINDS: Record<string, StatusKind> = { draft: "neutral", active: "info", closed: "good" };
const ASG_KINDS: Record<string, StatusKind> = {
  pending: "neutral",
  in_progress: "warning",
  submitted: "info",
  approved: "good",
  rejected: "critical",
};

// ── Cycles ────────────────────────────────────────────────────

export function FeedbackCycles({
  cycles,
  active,
  canWrite,
}: {
  cycles: FeedbackCycle[];
  active?: string;
  canWrite: boolean;
}) {
  const { set } = useUrlFilters(["cycle"]);
  const [editing, setEditing] = React.useState<FeedbackCycle | "new" | null>(null);
  return (
    <div className="flex flex-col gap-3">
      {canWrite ? (
        <Button size="sm" className="self-start" onClick={() => setEditing("new")}>
          <Plus className="size-4" /> {f.newCycle}
        </Button>
      ) : null}
      {cycles.length === 0 ? (
        <ErpEmpty title={f.noCycles} />
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {cycles.map((c) => (
            <button
              key={c.id}
              type="button"
              onClick={() => set({ cycle: c.id })}
              className={cn(
                "lift rounded-2xl bg-card p-4 text-left shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring",
                c.id === active && "ring-2 ring-brand",
              )}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate font-semibold">{c.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {c.periodLabel || (c.startDate ? `${formatDate(c.startDate)} – ${formatDate(c.endDate)}` : "–")}
                  </p>
                </div>
                <ErpStatusBadge status={c.status} labels={f.cycleStatus} kinds={CYCLE_KINDS} />
              </div>
              <p className="mt-2 text-xs text-muted-foreground">{f.counts(c.assignmentsCount, c.submittedCount, c.summariesCount)}</p>
            </button>
          ))}
        </div>
      )}
      {editing ? <CycleDialog cycle={editing === "new" ? null : editing} onClose={() => setEditing(null)} /> : null}
    </div>
  );
}

function CycleDialog({ cycle, onClose }: { cycle: FeedbackCycle | null; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const [form, setForm] = React.useState({
    name: cycle?.name ?? "",
    periodLabel: cycle?.periodLabel ?? "",
    startDate: cycle?.startDate ?? "",
    endDate: cycle?.endDate ?? "",
    status: cycle?.status ?? "draft",
    kpiWeight: String(cycle?.kpiWeight ?? 70),
    feedbackWeight: String(cycle?.feedbackWeight ?? 30),
    isAnonymous: cycle?.isAnonymous ?? true,
    allowSelfAssessment: cycle?.allowSelfAssessment ?? true,
    requireManagerReview: cycle?.requireManagerReview ?? false,
  });
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((p) => ({ ...p, [k]: v }));
  const submit = () =>
    run(
      () =>
        saveFeedbackCycleAction({
          id: cycle?.id ?? "",
          name: form.name,
          periodLabel: form.periodLabel,
          startDate: form.startDate || undefined,
          endDate: form.endDate || undefined,
          status: form.status,
          kpiWeight: Number(form.kpiWeight),
          feedbackWeight: Number(form.feedbackWeight),
          isAnonymous: form.isAnonymous,
          allowSelfAssessment: form.allowSelfAssessment,
          requireManagerReview: form.requireManagerReview,
        }),
      { success: f.cycleSaved, onOk: onClose, onFail: (r) => setErrors(r.fieldErrors ?? {}) },
    );
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            submit();
          }}
        >
          <DialogHeader>
            <DialogTitle>{cycle ? f.editCycle : f.newCycle}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3 sm:grid-cols-2">
            <FormRow label={f.name} htmlFor="fc-name" className="sm:col-span-2" error={errors.name?.[0]}>
              <Input id="fc-name" value={form.name} onChange={(e) => set("name", e.target.value)} required />
            </FormRow>
            <FormRow label={f.periodLabel} htmlFor="fc-period">
              <Input id="fc-period" value={form.periodLabel} onChange={(e) => set("periodLabel", e.target.value)} placeholder="Q4 2026" />
            </FormRow>
            <FormRow label={sp.common.status}>
              <Select value={form.status} onValueChange={(v) => set("status", v as typeof form.status)}>
                <SelectTrigger className="w-full" aria-label={sp.common.status}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(f.cycleStatus).map(([v, l]) => (
                    <SelectItem key={v} value={v}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={f.start}>
              <DatePicker value={form.startDate} onChange={(v) => set("startDate", v)} clearable />
            </FormRow>
            <FormRow label={f.end}>
              <DatePicker value={form.endDate} onChange={(v) => set("endDate", v)} clearable />
            </FormRow>
            <FormRow label={f.kpiWeight} htmlFor="fc-kw">
              <Input id="fc-kw" type="number" min={0} max={100} value={form.kpiWeight} onChange={(e) => set("kpiWeight", e.target.value)} />
            </FormRow>
            <FormRow label={f.feedbackWeight} htmlFor="fc-fw" error={errors.feedbackWeight?.[0]}>
              <Input id="fc-fw" type="number" min={0} max={100} value={form.feedbackWeight} onChange={(e) => set("feedbackWeight", e.target.value)} />
            </FormRow>
            {(
              [
                ["isAnonymous", f.anonymous],
                ["allowSelfAssessment", f.allowSelf],
                ["requireManagerReview", f.requireManager],
              ] as const
            ).map(([key, label]) => (
              <label key={key} className="flex items-center gap-2 text-sm">
                <Switch checked={form[key]} onCheckedChange={(v) => set(key, v)} /> {label}
              </label>
            ))}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {sp.common.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? sp.common.saving : sp.common.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** One cycle: actions, assignments and generated summaries. */
export function FeedbackCycleDetail({
  cycle,
  assignments,
  summaries,
  people,
  canWrite,
}: {
  cycle: FeedbackCycle;
  assignments: Assignment[];
  summaries: FeedbackSummary[];
  people: Person[];
  canWrite: boolean;
}) {
  const { pending, run } = useErpAction();
  const { set } = useUrlFilters(["cycle"]);
  const [editing, setEditing] = React.useState(false);
  const [adding, setAdding] = React.useState(false);
  const [peers, setPeers] = React.useState("2");
  const [openId, setOpenId] = React.useState<string | null>(null);

  const columns: ColumnDef<Assignment, unknown>[] = [
    { id: "employee", accessorFn: (a) => a.employeeName, header: f.reviewee },
    { id: "reviewer", accessorFn: (a) => a.reviewerName, header: f.reviewerLabel },
    { id: "rel", accessorFn: (a) => f.relationships[a.relationshipType] ?? a.relationshipType, header: f.relationship },
    {
      accessorKey: "status",
      header: sp.common.status,
      cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={f.assignmentStatus} kinds={ASG_KINDS} />,
    },
    {
      id: "due",
      accessorFn: (a) => a.dueDate ?? "",
      header: sp.tasks.dueDate,
      cell: ({ row }) => <span className="text-muted-foreground">{formatDate(row.original.dueDate)}</span>,
    },
    {
      id: "actions",
      header: sp.common.actions,
      enableSorting: false,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => (
        <span className="flex justify-end gap-1">
          <Button variant="ghost" size="xs" onClick={() => setOpenId(row.original.id)}>
            {f.view}
          </Button>
          {canWrite ? (
            <ConfirmAction
              trigger={
                <Button variant="ghost" size="icon-xs" aria-label={sp.common.delete}>
                  <Trash2 />
                </Button>
              }
              title={sp.common.confirmDelete}
              description={sp.common.confirmDeleteBody}
              confirmLabel={sp.common.delete}
              destructive
              onConfirm={() => run(() => deleteAssignmentAction(row.original.id), { success: sp.common.saved })}
            />
          ) : null}
        </span>
      ),
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <Card className="gap-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <p className="text-lg font-semibold">{cycle.name}</p>
            <p className="text-xs text-muted-foreground">
              {f.kpiWeight} {cycle.kpiWeight ?? 70} · {f.feedbackWeight} {cycle.feedbackWeight ?? 30}
              {cycle.isAnonymous ? ` · ${f.anonymous}` : ""}
            </p>
          </div>
          {canWrite ? (
            <div className="flex flex-wrap gap-2">
              <Button size="sm" variant="outline" onClick={() => setEditing(true)}>
                <Pencil className="size-4" /> {sp.common.edit}
              </Button>
              <ConfirmAction
                trigger={
                  <Button size="sm" variant="ghost" className="text-destructive">
                    <Trash2 className="size-4" /> {sp.common.delete}
                  </Button>
                }
                title={sp.common.confirmDelete}
                description={sp.common.confirmDeleteBody}
                confirmLabel={sp.common.delete}
                destructive
                onConfirm={() =>
                  run(() => deleteFeedbackCycleAction(cycle.id), { success: f.cycleDeleted, onOk: () => set({ cycle: null }) })
                }
              />
            </div>
          ) : null}
        </div>
        {canWrite ? (
          <div className="flex flex-wrap items-end gap-2 border-t pt-3">
            <FormRow label={f.peers} htmlFor="fb-peers" className="w-44">
              <Input id="fb-peers" type="number" min={0} max={10} value={peers} onChange={(e) => setPeers(e.target.value)} />
            </FormRow>
            <Button
              size="sm"
              variant="secondary"
              disabled={pending}
              title={f.autoAssignBody}
              onClick={() => run(() => autoAssignAction(cycle.id, Number(peers)), { onOk: (r) => toast.success(r.message) })}
            >
              <Wand2 className="size-4" /> {f.autoAssign}
            </Button>
            <Button size="sm" variant="outline" onClick={() => setAdding(true)}>
              <Users className="size-4" /> {f.addAssignment}
            </Button>
            <Button
              size="sm"
              disabled={pending}
              onClick={() =>
                run(() => generateSummariesAction(cycle.id), { onOk: (r) => toast.success(f.generated(r.generated, r.improvementsFiled)) })
              }
            >
              <Calculator className="size-4" /> {f.generate}
            </Button>
          </div>
        ) : null}
      </Card>

      <section className="flex flex-col gap-2">
        <h2 className="text-base font-semibold">{f.assignments}</h2>
        <ErpDataTable columns={columns} data={assignments} emptyMessage={f.noAssignments} getRowId={(a) => a.id} />
      </section>

      <section className="flex flex-col gap-2">
        <h2 className="text-base font-semibold">{f.summaries}</h2>
        {summaries.length === 0 ? (
          <ErpEmpty title={f.noSummaries} compact />
        ) : (
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
            {summaries.map((s) => (
              <SummaryCard key={s.id} summary={s} canWrite={canWrite} />
            ))}
          </div>
        )}
      </section>

      {editing ? <CycleDialog cycle={cycle} onClose={() => setEditing(false)} /> : null}
      {adding ? <AssignmentDialog cycleId={cycle.id} people={people} onClose={() => setAdding(false)} /> : null}
      <AnswerDialog id={openId} onClose={() => setOpenId(null)} />
    </div>
  );
}

function AssignmentDialog({ cycleId, people, onClose }: { cycleId: string; people: Person[]; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [form, setForm] = React.useState({ employeeId: "", reviewerId: "", relationshipType: "peer" as Relationship, dueDate: "" });
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((p) => ({ ...p, [k]: v }));
  const personSelect = (key: "employeeId" | "reviewerId", label: string) => (
    <FormRow label={label}>
      <Select value={form[key]} onValueChange={(v) => set(key, v)}>
        <SelectTrigger className="w-full" aria-label={label}>
          <SelectValue placeholder={label} />
        </SelectTrigger>
        <SelectContent>
          {people.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              {p.name}
              {p.departmentName ? ` · ${p.departmentName}` : ""}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </FormRow>
  );
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{f.addAssignment}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-3">
          {personSelect("employeeId", f.reviewee)}
          {personSelect("reviewerId", f.reviewerLabel)}
          <FormRow label={f.relationship}>
            <Select value={form.relationshipType} onValueChange={(v) => set("relationshipType", v as Relationship)}>
              <SelectTrigger className="w-full" aria-label={f.relationship}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.entries(f.relationships).map(([v, l]) => (
                  <SelectItem key={v} value={v}>
                    {l}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormRow>
          <FormRow label={sp.tasks.dueDate}>
            <DatePicker value={form.dueDate} onChange={(v) => set("dueDate", v)} clearable />
          </FormRow>
        </div>
        <DialogFooter>
          <Button
            disabled={pending}
            onClick={() =>
              run(() => createAssignmentAction(cycleId, { ...form, dueDate: form.dueDate || undefined }), {
                success: sp.common.saved,
                onOk: onClose,
              })
            }
          >
            {sp.common.save}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SummaryCard({ summary, canWrite }: { summary: FeedbackSummary; canWrite: boolean }) {
  const [editing, setEditing] = React.useState(false);
  return (
    <Card className="gap-3 p-4">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate font-semibold">{summary.employeeName}</p>
          <p className="truncate text-xs text-muted-foreground">{summary.departmentName}</p>
        </div>
        <div className="text-right">
          <Score value={summary.finalScore} className="text-xl" />
          {summary.finalGrade ? <p className="text-xs text-muted-foreground">{f.grade} {summary.finalGrade}</p> : null}
        </div>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs sm:grid-cols-3">
        <span>
          {f.overall}: <Score value={summary.overall360Score} />
        </span>
        <span>
          {f.kpi}: <Score value={summary.kpiScore} />
        </span>
        {Object.entries(summary.scores).map(([code, v]) => (
          <span key={code}>
            {f.categories[code] ?? code}: <span className="tabular">{formatDecimal(v, 0)}</span>
          </span>
        ))}
      </div>
      {summary.strengths.length > 0 ? (
        <div className="text-xs">
          <p className="font-medium">{f.strengths}</p>
          <ul className="list-disc pl-4 text-muted-foreground">
            {summary.strengths.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {summary.weaknesses.length > 0 ? (
        <div className="text-xs">
          <p className="font-medium">{f.weaknesses}</p>
          <ul className="list-disc pl-4 text-muted-foreground">
            {summary.weaknesses.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {summary.managerComments ? <p className="text-xs italic">{summary.managerComments}</p> : null}
      {canWrite && !summary.isLocked ? (
        <Button size="sm" variant="outline" className="self-start" onClick={() => setEditing(true)}>
          <Pencil className="size-4" /> {sp.common.edit}
        </Button>
      ) : summary.isLocked ? (
        <span className="text-xs text-muted-foreground">{f.locked}</span>
      ) : null}
      {editing ? <SummaryDialog summary={summary} onClose={() => setEditing(false)} /> : null}
    </Card>
  );
}

function SummaryDialog({ summary, onClose }: { summary: FeedbackSummary; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [strengths, setStrengths] = React.useState(summary.strengths.join("\n"));
  const [weaknesses, setWeaknesses] = React.useState(summary.weaknesses.join("\n"));
  const [comments, setComments] = React.useState(summary.managerComments ?? "");
  const [locked, setLocked] = React.useState(summary.isLocked);
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{summary.employeeName}</DialogTitle>
          <DialogDescription>{f.onePerLine}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <FormRow label={f.strengths} htmlFor="sm-str">
            <Textarea id="sm-str" value={strengths} onChange={(e) => setStrengths(e.target.value)} />
          </FormRow>
          <FormRow label={f.weaknesses} htmlFor="sm-weak">
            <Textarea id="sm-weak" value={weaknesses} onChange={(e) => setWeaknesses(e.target.value)} />
          </FormRow>
          <FormRow label={f.managerComments} htmlFor="sm-com">
            <Textarea id="sm-com" value={comments} onChange={(e) => setComments(e.target.value)} />
          </FormRow>
          <label className="flex items-center gap-2 text-sm">
            <Switch checked={locked} onCheckedChange={setLocked} /> {f.lock}
          </label>
        </div>
        <DialogFooter>
          <Button
            disabled={pending}
            onClick={() =>
              run(
                () =>
                  updateSummaryAction(summary.id, {
                    strengths: lines(strengths),
                    weaknesses: lines(weaknesses),
                    managerComments: comments || undefined,
                    isLocked: locked,
                  }),
                { success: f.summarySaved, onOk: onClose },
              )
            }
          >
            {sp.common.save}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── My assignments and answering ──────────────────────────────

export function MyAssignments({ assignments }: { assignments: Assignment[] }) {
  const [openId, setOpenId] = React.useState<string | null>(null);
  if (assignments.length === 0) return <ErpEmpty title={f.noMine} />;
  return (
    <>
      <Card className="gap-0 divide-y py-0">
        {assignments.map((a) => (
          <div key={a.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
            <div className="min-w-0 flex-1">
              <p className="font-medium">{a.employeeName}</p>
              <p className="text-xs text-muted-foreground">
                {a.cycleName} · {f.relationships[a.relationshipType]}
                {a.dueDate ? ` · ${f.due(formatDate(a.dueDate))}` : ""}
              </p>
            </div>
            <ErpStatusBadge status={a.status} labels={f.assignmentStatus} kinds={ASG_KINDS} />
            <Button size="sm" variant={a.status === "pending" || a.status === "in_progress" || a.status === "rejected" ? "default" : "outline"} onClick={() => setOpenId(a.id)}>
              {a.status === "pending" || a.status === "in_progress" || a.status === "rejected" ? f.answer : f.view}
            </Button>
          </div>
        ))}
      </Card>
      <AnswerDialog id={openId} onClose={() => setOpenId(null)} />
    </>
  );
}

function AnswerDialog({ id, onClose }: { id: string | null; onClose: () => void }) {
  const { data: detail, error } = useResource<AssignmentDetail>(
    id ? `/v1/performance/feedback/assignments/${encodeURIComponent(id)}` : null,
  );
  return (
    <Dialog open={!!id} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        {error ? (
          <DialogHeader>
            <DialogTitle>{sp.common.loadFailed}</DialogTitle>
            <DialogDescription>{error}</DialogDescription>
          </DialogHeader>
        ) : !detail ? (
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Loader2 className="size-4 animate-spin" /> {sp.common.loading}
            </DialogTitle>
          </DialogHeader>
        ) : (
          <AnswerBody key={detail.assignment.id} detail={detail} onDone={onClose} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function AnswerBody({ detail, onDone }: { detail: AssignmentDetail; onDone: () => void }) {
  const { assignment: a } = detail;
  const { pending, run } = useErpAction();
  const initial = Object.fromEntries(detail.responses.map((r) => [r.criteriaId, { rating: r.rating, comments: r.comments ?? "" }]));
  const [answers, setAnswers] = React.useState<Record<string, { rating: number; comments: string }>>(initial);
  const [reason, setReason] = React.useState("");
  const [managerComments, setManagerComments] = React.useState(a.managerComments ?? "");
  const patch = (cid: string, p: Partial<{ rating: number; comments: string }>) =>
    setAnswers((prev) => ({ ...prev, [cid]: { ...(prev[cid] ?? { rating: 0, comments: "" }), ...p } }));
  const send = (submit: boolean) =>
    run(
      () =>
        submitResponsesAction(a.id, {
          submit,
          answers: Object.entries(answers)
            .filter(([, v]) => v.rating > 0)
            .map(([criteriaId, v]) => ({ criteriaId, rating: v.rating, comments: v.comments || undefined })),
        }),
      { success: submit ? f.submitted : f.draftSaved, onOk: submit ? onDone : undefined },
    );

  return (
    <div className="flex flex-col gap-5">
      <DialogHeader>
        <DialogTitle>{f.answerTitle(a.employeeName)}</DialogTitle>
        <DialogDescription>
          {a.cycleName} · {f.relationships[a.relationshipType]} · {a.reviewerName}
        </DialogDescription>
      </DialogHeader>
      {a.rejectionReason ? (
        <p className="rounded-xl bg-destructive/10 px-3 py-2 text-sm text-destructive">{a.rejectionReason}</p>
      ) : null}
      {detail.categories.map((c) => (
        <section key={c.id} className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold">
            {c.name} <span className="font-normal text-muted-foreground">· {c.description}</span>
          </h3>
          {c.criteria.map((cr) => {
            const v = answers[cr.id];
            return (
              <div key={cr.id} className="rounded-xl border p-3">
                <p className="text-sm font-medium">{cr.name}</p>
                <p className="text-xs text-muted-foreground">{cr.description}</p>
                <div className="mt-2 flex gap-2">
                  {[1, 2, 3, 4, 5].map((n) => (
                    <button
                      key={n}
                      type="button"
                      disabled={!detail.canAnswer || pending}
                      aria-pressed={v?.rating === n}
                      onClick={() => patch(cr.id, { rating: n })}
                      className={cn(
                        "flex h-11 flex-1 items-center justify-center rounded-lg border text-sm font-semibold transition-colors",
                        v?.rating === n ? "border-brand bg-brand text-white" : "bg-card hover:bg-muted",
                        !detail.canAnswer && "opacity-60",
                      )}
                    >
                      {n}
                    </button>
                  ))}
                </div>
                {detail.canAnswer ? (
                  <Input
                    className="mt-2"
                    placeholder={f.comment}
                    aria-label={f.comment}
                    value={v?.comments ?? ""}
                    onChange={(e) => patch(cr.id, { comments: e.target.value })}
                  />
                ) : v?.comments ? (
                  <p className="mt-2 text-xs italic">{v.comments}</p>
                ) : null}
              </div>
            );
          })}
        </section>
      ))}
      {detail.canAnswer ? (
        <DialogFooter>
          <Button variant="outline" disabled={pending} onClick={() => send(false)}>
            {f.saveDraft}
          </Button>
          <Button disabled={pending} onClick={() => send(true)}>
            {f.submit}
          </Button>
        </DialogFooter>
      ) : null}
      {detail.canDecide && a.status === "submitted" ? (
        <section className="flex flex-col gap-2 border-t pt-4">
          <FormRow label={f.managerComments} htmlFor="dc-com">
            <Textarea id="dc-com" value={managerComments} onChange={(e) => setManagerComments(e.target.value)} />
          </FormRow>
          <FormRow label={f.rejectionReason} htmlFor="dc-reason">
            <Input id="dc-reason" value={reason} onChange={(e) => setReason(e.target.value)} />
          </FormRow>
          <div className="flex gap-2">
            <Button
              disabled={pending}
              onClick={() =>
                run(() => decideAssignmentAction(a.id, { approve: true, managerComments: managerComments || undefined }), {
                  success: f.decided,
                  onOk: onDone,
                })
              }
            >
              {f.approve}
            </Button>
            <Button
              variant="destructive"
              disabled={pending}
              onClick={() =>
                run(() => decideAssignmentAction(a.id, { approve: false, rejectionReason: reason, managerComments: managerComments || undefined }), {
                  success: f.decided,
                  onOk: onDone,
                })
              }
            >
              {f.reject}
            </Button>
          </div>
        </section>
      ) : null}
    </div>
  );
}
