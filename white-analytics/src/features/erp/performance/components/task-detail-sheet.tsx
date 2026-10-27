"use client";

import * as React from "react";
import { CheckCircle2, Download, Loader2, Paperclip, Play, RotateCcw, Trash2, X, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { formatDate, formatDateTime } from "@/lib/format";
import { deactivateTaskAction, deleteAttachmentAction, occurrenceAction, updateTaskAction } from "../actions";
import { formatBytes } from "../lib";
import { sp } from "../strings";
import type { OccurrenceAction, TaskDetail, TaskMeta, TaskUpdate } from "../types";
import { ConfirmAction, FormRow, PriorityBadge, TaskStatusBadge } from "./common";
import { useResource } from "./use-resource";

const s = sp.tasks;
const NONE = "__none__";

function updateLine(u: TaskUpdate): string {
  const who = u.authorName || s.system;
  let detail: Record<string, unknown> = {};
  try {
    detail = JSON.parse(u.detail || "{}");
  } catch {
    detail = {};
  }
  const status = typeof detail.status === "string" ? (sp.taskStatus[detail.status] ?? detail.status) : "";
  switch (u.kind) {
    case "progress":
      return s.updateLine.progress(who, u.progressPercent);
    case "comment":
      return s.updateLine.comment(who);
    case "status":
      return s.updateLine.status(who, status);
    case "review":
      return s.updateLine.review(who, status);
    case "attachment":
      return s.updateLine.attachment(who, typeof detail.name === "string" ? detail.name : "file");
    default:
      return s.updateLine.created(who);
  }
}

/** Occurrence panel: subtasks, progress, review, attachments, comments, history and task settings. */
export function TaskDetailSheet({
  occurrenceId,
  meta,
  onClose,
}: {
  occurrenceId: string | null;
  meta?: TaskMeta | null;
  onClose: () => void;
}) {
  const { data: current, error, reload } = useResource<TaskDetail>(
    occurrenceId ? `/v1/performance/tasks/occurrences/${encodeURIComponent(occurrenceId)}` : null,
  );

  return (
    <Sheet open={!!occurrenceId} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full overflow-y-auto sm:max-w-lg">
        {error ? (
          <SheetHeader>
            <SheetTitle>{s.notFound}</SheetTitle>
            <SheetDescription>{error}</SheetDescription>
          </SheetHeader>
        ) : !current ? (
          <SheetHeader>
            <SheetTitle className="flex items-center gap-2">
              <Loader2 className="size-4 animate-spin" /> {s.loadingTask}
            </SheetTitle>
            <SheetDescription className="sr-only">{s.loadingTask}</SheetDescription>
          </SheetHeader>
        ) : (
          <TaskDetailBody key={current.item.occurrenceId} detail={current} meta={meta} reload={reload} onClose={onClose} />
        )}
      </SheetContent>
    </Sheet>
  );
}

function TaskDetailBody({
  detail,
  meta,
  reload,
  onClose,
}: {
  detail: TaskDetail;
  meta?: TaskMeta | null;
  reload: () => Promise<void>;
  onClose: () => void;
}) {
  const { item, subtasks, updates, attachments } = detail;
  const { pending, run } = useErpAction();
  const [note, setNote] = React.useState("");
  const [percent, setPercent] = React.useState(item.progressPercent != null ? String(item.progressPercent) : "");
  const [comment, setComment] = React.useState("");
  const [rejectNote, setRejectNote] = React.useState("");
  const [rejecting, setRejecting] = React.useState(false);
  const [editing, setEditing] = React.useState(false);

  const act = (input: { action: OccurrenceAction; note?: string; notes?: string; progressPercent?: number; subtaskId?: string; checked?: boolean }, after?: () => void) =>
    run(() => occurrenceAction(item.occurrenceId, input), {
      onOk: () => {
        after?.();
        void reload();
      },
    });

  const open = item.status === "pending" || item.status === "in_progress" || item.status === "rejected";
  const canApprove = item.canReview && item.status === "done";
  const canReopen =
    (item.status === "done" && (item.canReview || item.canWork)) ||
    (item.status === "approved" && (item.canReview || (item.canWork && !item.reviewRequired)));
  const schedule =
    item.recurrence === "once"
      ? ""
      : `${sp.recurrence[item.recurrence]}${item.weeklyDay ? ` (${sp.days[item.weeklyDay]})` : item.monthlyDay ? ` (tgl ${item.monthlyDay})` : ""}`;

  return (
    <div className="flex flex-col gap-5 px-(--card-pad) pb-8">
      <SheetHeader className="px-0">
        <SheetTitle className="flex flex-wrap items-center gap-2 pr-8">
          {item.title}
          <PriorityBadge priority={item.priority} />
        </SheetTitle>
        <SheetDescription className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <TaskStatusBadge status={item.status} overdue={item.overdue} />
          <span>{formatDate(item.occurrenceDate)}</span>
          <span>· {item.departmentName}</span>
          <span>· {s.pic(item.assigneeName)}</span>
          {schedule ? <span>· {schedule}</span> : null}
          {!item.reviewRequired ? <Badge variant="outline">{s.personal}</Badge> : null}
        </SheetDescription>
      </SheetHeader>

      {item.description ? <p className="text-sm whitespace-pre-wrap">{item.description}</p> : null}
      {item.kpiIndicatorName ? (
        <p className="text-xs text-muted-foreground">
          {s.kpiLine(item.kpiIndicatorName)}
          {item.kpiContribution != null ? ` · ${s.contributionLine(item.kpiContribution)}` : ""}
        </p>
      ) : null}
      {item.status === "rejected" && item.reviewNotes ? (
        <p className="rounded-xl bg-destructive/10 px-3 py-2 text-sm text-destructive">{s.rejectedBecause(item.reviewNotes)}</p>
      ) : null}

      <div className="flex items-center gap-2">
        <Progress value={item.progress} aria-label={s.progress} className="h-2 flex-1" />
        <span className="w-10 text-right text-xs tabular">{item.progress}%</span>
      </div>

      {subtasks.length > 0 ? (
        <section className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold">{s.subtasks}</h3>
          <ul className="flex flex-col gap-1.5">
            {subtasks.map((st) => (
              <li key={st.id} className="flex items-center gap-2 text-sm">
                <Checkbox
                  aria-label={st.title}
                  checked={st.isChecked}
                  disabled={!item.canWork || item.status === "approved" || pending}
                  onCheckedChange={(v) => act({ action: "check_subtask", subtaskId: st.id, checked: v === true })}
                />
                <span className={st.isChecked ? "flex-1 text-muted-foreground line-through" : "flex-1"}>{st.title}</span>
                <span className="text-xs text-muted-foreground tabular">{Math.round(st.weight)}%</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {item.canWork && open ? (
        <section className="flex flex-col gap-3 rounded-2xl bg-muted/50 p-3">
          <h3 className="text-sm font-semibold">{s.updateProgress}</h3>
          <div className="grid gap-2 sm:grid-cols-[7rem_minmax(0,1fr)]">
            {subtasks.length === 0 ? (
              <FormRow label={s.progressPercent} htmlFor="task-pct">
                <Input
                  id="task-pct"
                  type="number"
                  min={0}
                  max={99}
                  inputMode="numeric"
                  value={percent}
                  onChange={(e) => setPercent(e.target.value)}
                />
              </FormRow>
            ) : (
              <p className="self-end text-xs text-muted-foreground">{s.progressFromSubtasks}</p>
            )}
            <FormRow label={s.progressNote} htmlFor="task-note">
              <Input
                id="task-note"
                value={note}
                maxLength={2000}
                placeholder={s.progressNotePlaceholder}
                onChange={(e) => setNote(e.target.value)}
              />
            </FormRow>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={pending || (!note.trim() && percent === "")}
              onClick={() =>
                act(
                  {
                    action: "progress",
                    note: note.trim() || undefined,
                    progressPercent: subtasks.length === 0 && percent !== "" ? Number(percent) : undefined,
                  },
                  () => setNote(""),
                )
              }
            >
              {s.saveProgress}
            </Button>
            {item.status !== "in_progress" ? (
              <Button variant="outline" size="sm" disabled={pending} onClick={() => act({ action: "start" })}>
                <Play /> {s.start}
              </Button>
            ) : null}
            {subtasks.length === 0 ? (
              <Button size="sm" disabled={pending} onClick={() => act({ action: "done" })}>
                <CheckCircle2 /> {item.reviewRequired ? s.markDone : s.finish}
              </Button>
            ) : null}
          </div>
        </section>
      ) : null}

      {canApprove || canReopen ? (
        <section className="flex flex-wrap items-center gap-2">
          {canApprove ? (
            <>
              <Button size="sm" disabled={pending} onClick={() => act({ action: "approve" })}>
                <CheckCircle2 /> {s.approve}
              </Button>
              <Button variant="destructive" size="sm" disabled={pending} onClick={() => setRejecting((v) => !v)}>
                <XCircle /> {s.reject}
              </Button>
            </>
          ) : null}
          {canReopen ? (
            <Button variant="ghost" size="sm" disabled={pending} onClick={() => act({ action: "reopen" })}>
              <RotateCcw /> {s.reopen}
            </Button>
          ) : null}
          {rejecting ? (
            <form
              className="flex w-full gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                act({ action: "reject", notes: rejectNote.trim() || undefined }, () => {
                  setRejecting(false);
                  setRejectNote("");
                });
              }}
            >
              <Input
                aria-label={s.rejectReason}
                placeholder={s.rejectReason}
                value={rejectNote}
                onChange={(e) => setRejectNote(e.target.value)}
              />
              <Button type="submit" variant="destructive" size="sm" disabled={pending}>
                {s.sendBack}
              </Button>
            </form>
          ) : null}
        </section>
      ) : null}

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{s.attachments}</h3>
        {attachments.length > 0 ? (
          <ul className="flex flex-col gap-1.5">
            {attachments.map((a) => {
              const href = `/api/erp/v1/performance/tasks/occurrences/${encodeURIComponent(item.occurrenceId)}/attachments/${encodeURIComponent(a.id)}`;
              return (
                <li key={a.id} className="flex items-center gap-2 text-sm">
                  <Paperclip className="size-4 shrink-0 text-muted-foreground" />
                  <a href={href} target="_blank" rel="noreferrer" className="min-w-0 flex-1 truncate underline-offset-4 hover:underline">
                    {a.name}
                  </a>
                  <span className="text-xs text-muted-foreground tabular">{formatBytes(Number(a.sizeBytes))}</span>
                  <a href={href} download aria-label={`${sp.common.detail} ${a.name}`} className="text-muted-foreground hover:text-foreground">
                    <Download className="size-4" />
                  </a>
                  {item.canManage || item.canWork ? (
                    <Button
                      variant="ghost"
                      size="icon-xs"
                      aria-label={s.removeAttachment(a.name)}
                      disabled={pending}
                      onClick={() => run(() => deleteAttachmentAction(item.occurrenceId, a.id), { onOk: () => void reload() })}
                    >
                      <X />
                    </Button>
                  ) : null}
                </li>
              );
            })}
          </ul>
        ) : null}
        {item.canWork || item.canManage ? (
          <FileUpload
            url={`/api/erp/api/performance/tasks/occurrences/${encodeURIComponent(item.occurrenceId)}/attachments`}
            accept="image/*,application/pdf,.doc,.docx,.xls,.xlsx,.csv,.txt,.zip"
            acceptLabel="Gambar, PDF, dokumen"
            multiple={false}
            onUploaded={() => void reload()}
          />
        ) : null}
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{s.comments}</h3>
        <ul className="flex flex-col gap-2">
          {[...updates]
            .reverse()
            .filter((u) => u.note)
            .map((u) => (
              <li key={u.id} className="rounded-xl bg-muted/50 px-3 py-2 text-sm">
                <p className="text-xs text-muted-foreground">
                  {u.authorName || s.system} · {formatDateTime(u.createdAt)}
                  {u.kind === "progress" && u.progressPercent != null ? ` · ${u.progressPercent}%` : ""}
                </p>
                <p className="whitespace-pre-wrap">{u.note}</p>
              </li>
            ))}
        </ul>
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            if (comment.trim()) act({ action: "comment", note: comment.trim() }, () => setComment(""));
          }}
        >
          <Input aria-label={s.commentPlaceholder} placeholder={s.commentPlaceholder} value={comment} onChange={(e) => setComment(e.target.value)} />
          <Button type="submit" variant="secondary" disabled={!comment.trim() || pending}>
            {s.send}
          </Button>
        </form>
      </section>

      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-semibold">{s.history}</h3>
        <ol className="flex flex-col gap-1 text-xs text-muted-foreground">
          {updates.map((u) => (
            <li key={u.id}>
              {formatDateTime(u.createdAt)} · {updateLine(u)}
            </li>
          ))}
          {updates.length === 0 ? <li>{s.noActivity}</li> : null}
        </ol>
      </section>

      {item.canManage ? (
        <section className="flex flex-col gap-3 border-t pt-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">{s.manage}</h3>
            <Button variant="ghost" size="sm" onClick={() => setEditing((v) => !v)}>
              {editing ? sp.common.close : sp.common.edit}
            </Button>
          </div>
          {editing ? <TaskEditForm detail={detail} meta={meta} onSaved={() => { setEditing(false); void reload(); }} /> : null}
          <div className="flex justify-end">
            <ConfirmAction
              trigger={
                <Button variant="ghost" size="sm" className="text-destructive">
                  <Trash2 /> {s.deactivate}
                </Button>
              }
              title={s.deactivateTitle}
              description={s.deactivateBody(item.title)}
              confirmLabel={s.deactivate}
              destructive
              onConfirm={() => run(() => deactivateTaskAction(item.taskId), { success: s.deactivated, onOk: onClose })}
            />
          </div>
        </section>
      ) : null}
    </div>
  );
}

function TaskEditForm({ detail, meta, onSaved }: { detail: TaskDetail; meta?: TaskMeta | null; onSaved: () => void }) {
  const { item } = detail;
  const { pending, run } = useErpAction();
  const members = (meta?.members ?? []).filter((m) => m.departmentId === item.departmentId);
  const indicators = (meta?.indicators ?? []).filter(
    (i) => i.departmentIds.length === 0 || i.departmentIds.includes(item.departmentId),
  );
  const [form, setForm] = React.useState({
    title: item.title,
    description: item.description,
    priority: item.priority,
    assignee: item.assigneeEmployeeId ?? NONE,
    indicator: item.kpiIndicatorId ?? NONE,
    contribution: item.kpiContribution != null ? String(item.kpiContribution) : "",
  });
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [k]: v }));

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        run(
          () =>
            updateTaskAction(item.taskId, {
              title: form.title,
              description: form.description,
              priority: form.priority,
              assigneeEmployeeId: form.assignee === NONE ? "" : form.assignee,
              kpiIndicatorId: form.indicator === NONE ? "" : form.indicator,
              kpiContribution: form.contribution === "" || form.indicator === NONE ? null : Number(form.contribution),
            }),
          { success: sp.common.saved, onOk: onSaved },
        );
      }}
    >
      <FormRow label={s.titleLabel} htmlFor="edit-title" className="sm:col-span-2">
        <Input id="edit-title" value={form.title} onChange={(e) => set("title", e.target.value)} required minLength={3} maxLength={200} />
      </FormRow>
      <FormRow label={s.priorityLabel}>
        <Select value={form.priority} onValueChange={(v) => set("priority", v as typeof form.priority)}>
          <SelectTrigger className="w-full" aria-label={s.priorityLabel}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(sp.priority).map(([v, l]) => (
              <SelectItem key={v} value={v}>
                {l}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormRow>
      <FormRow label={s.assignee}>
        <Select value={form.assignee} onValueChange={(v) => set("assignee", v)}>
          <SelectTrigger className="w-full" aria-label={s.assignee}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NONE}>{s.wholeDepartment}</SelectItem>
            {members.map((m) => (
              <SelectItem key={m.id} value={m.id}>
                {m.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormRow>
      <FormRow label={s.indicator}>
        <Select value={form.indicator} onValueChange={(v) => set("indicator", v)}>
          <SelectTrigger className="w-full" aria-label={s.indicator}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={NONE}>{s.noIndicator}</SelectItem>
            {indicators.map((i) => (
              <SelectItem key={i.id} value={i.id}>
                {i.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </FormRow>
      <FormRow label={s.contribution} htmlFor="edit-contrib">
        <Input
          id="edit-contrib"
          type="number"
          step="any"
          inputMode="decimal"
          value={form.contribution}
          disabled={form.indicator === NONE}
          onChange={(e) => set("contribution", e.target.value)}
        />
      </FormRow>
      <FormRow label={s.description} htmlFor="edit-desc" className="sm:col-span-2">
        <Textarea id="edit-desc" value={form.description} onChange={(e) => set("description", e.target.value)} />
      </FormRow>
      <div className="flex justify-end sm:col-span-2">
        <Button type="submit" size="sm" disabled={pending || form.title.trim().length < 3}>
          {pending ? sp.common.saving : sp.common.save}
        </Button>
      </div>
    </form>
  );
}
