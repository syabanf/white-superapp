"use client";

import { Section } from "@/components/dashboard/page-header";
import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { CalendarPlus, MessageSquare, Plus, Send, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import type { StatusKind } from "@/components/dashboard/status-badge";
import type { ColMeta } from "@/components/dashboard/data-table";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDate, formatDateTime, formatNumber } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  createEntryAction,
  createTemplateAction,
  deleteEntryAction,
  deleteTemplateAction,
  reviewEntryAction,
  submitEntryAction,
  updateEntryItemAction,
} from "../actions";
import { sp } from "../strings";
import type { Department, Frequency, LogbookEntry, LogbookSummaryRow, LogbookTemplate } from "../types";
import { ConfirmAction, FormRow } from "./common";

const l = sp.logbook;

export const ENTRY_KINDS: Record<string, StatusKind> = {
  draft: "neutral",
  submitted: "info",
  reviewed: "good",
  rejected: "critical",
};

function EntryStatus({ status }: { status: string }) {
  return <ErpStatusBadge status={status} labels={l.entryStatus} kinds={ENTRY_KINDS} />;
}

/** Name of the entry's department; the service may leave it empty. */
const deptLabel = (e: { departmentName: string; departmentId: string }, names: Record<string, string>) =>
  e.departmentName || names[e.departmentId] || "";

// ── Checklist ─────────────────────────────────────────────────

export function ChecklistTab({
  templates,
  entries,
  date,
  canReview,
  deptNames,
}: {
  templates: LogbookTemplate[];
  entries: LogbookEntry[];
  date: string;
  canReview: boolean;
  deptNames: Record<string, string>;
}) {
  const { pending, run } = useErpAction();
  const [templateId, setTemplateId] = React.useState(templates[0]?.id ?? "");
  const [selected, setSelected] = React.useState<string | null>(null);
  const entry = entries.find((e) => e.id === selected) ?? entries[0] ?? null;

  return (
    <div className="flex flex-col gap-4">
      <Card className="gap-3 p-(--card-pad)">
        <p className="flex items-center gap-2 text-sm font-semibold">
          <CalendarPlus className="size-4" /> {l.createEntry}
        </p>
        <p className="text-xs text-muted-foreground">{l.createEntryHint}</p>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Select value={templateId} onValueChange={setTemplateId}>
            <SelectTrigger className="w-full sm:max-w-sm" aria-label={l.template}>
              <SelectValue placeholder={templates.length ? l.pickTemplate : l.noTemplates} />
            </SelectTrigger>
            <SelectContent>
              {templates.map((t) => (
                <SelectItem key={t.id} value={t.id}>
                  {t.name}
                  {deptLabel(t, deptNames) ? ` · ${deptLabel(t, deptNames)}` : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            disabled={pending || !templateId}
            onClick={() =>
              run(() => createEntryAction(templateId, date), { success: l.entryCreated, onOk: (e) => setSelected(e.id) })
            }
          >
            {l.createEntry}
          </Button>
        </div>
      </Card>

      <div className="grid gap-(--gap) xl:grid-cols-[340px_minmax(0,1fr)]">
        <Section title={l.entriesFor(formatDate(date))}>
          {entries.length === 0 ? (
            <ErpEmpty title={l.noEntries} compact />
          ) : (
            entries.map((e) => (
              <button
                key={e.id}
                type="button"
                onClick={() => setSelected(e.id)}
                className={cn(
                  "rounded-2xl border bg-card p-3 text-left transition outline-none hover:bg-muted/40 focus-visible:ring-2 focus-visible:ring-ring",
                  entry?.id === e.id && "border-brand",
                )}
              >
                <div className="flex items-start justify-between gap-2">
                  <p className="text-sm font-medium">{e.title}</p>
                  <EntryStatus status={e.status} />
                </div>
                <p className="mt-1 text-xs text-muted-foreground">
                  {[e.employeeName, deptLabel(e, deptNames)].filter(Boolean).join(" · ")}
                </p>
                <Progress value={e.completionPercentage} className="mt-2 h-1.5" aria-label={l.completion} />
                <p className="mt-1 text-xs text-muted-foreground tabular">
                  {l.pctDone(Math.round(e.completionPercentage), Math.round(e.kpiScore))}
                </p>
              </button>
            ))
          )}
        </Section>
        <Card className="gap-3 p-(--card-pad)">
          {entry ? (
            <EntryView key={entry.id} entry={entry} canReview={canReview} editable />
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">{l.pickEntry}</p>
          )}
        </Card>
      </div>
    </div>
  );
}

/** Checklist of one entry with item notes, submit, delete draft and review. */
function EntryView({
  entry,
  canReview,
  editable,
  onDone,
}: {
  entry: LogbookEntry;
  canReview: boolean;
  editable?: boolean;
  onDone?: () => void;
}) {
  const { pending, run } = useErpAction();
  const [noteFor, setNoteFor] = React.useState<string | null>(null);
  const [note, setNote] = React.useState("");
  const [reviewNotes, setReviewNotes] = React.useState("");
  const canEdit = editable && entry.status === "draft";
  const items = [...entry.items].sort((a, b) => a.sortOrder - b.sortOrder);

  return (
    <div className="flex flex-col gap-(--gap-inner)">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="font-semibold">{entry.title}</p>
        <EntryStatus status={entry.status} />
      </div>
      {entry.reviewNotes ? <p className="rounded-xl bg-muted/60 px-3 py-2 text-xs">{l.rejectedNote(entry.reviewNotes)}</p> : null}
      <ul className="flex flex-col divide-y rounded-xl border">
        {items.map((it) => (
          <li key={it.id} className="flex flex-col gap-1.5 px-3 py-2.5">
            <div className="flex items-start gap-3">
              <Checkbox
                className="mt-0.5 size-5"
                checked={it.isChecked}
                disabled={!canEdit || pending}
                aria-label={it.title}
                onCheckedChange={(v) => run(() => updateEntryItemAction(it.id, { isChecked: v === true }))}
              />
              <div className="min-w-0 flex-1">
                <p className={cn("text-sm", it.isChecked && "text-muted-foreground line-through")}>
                  {it.title}
                  {it.isRequired ? <span className="ml-1.5 text-xs text-destructive">{l.required}</span> : null}
                </p>
                {it.notes ? <p className="text-xs text-muted-foreground">{it.notes}</p> : null}
                {it.checkedAt ? <p className="text-[11px] text-muted-foreground">{formatDateTime(it.checkedAt)}</p> : null}
              </div>
              <span className="text-xs text-muted-foreground tabular">{formatNumber(it.weight, 1)}</span>
              {canEdit ? (
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={l.itemNotes}
                  onClick={() => {
                    setNoteFor(noteFor === it.id ? null : it.id);
                    setNote(it.notes ?? "");
                  }}
                >
                  <MessageSquare />
                </Button>
              ) : null}
            </div>
            {noteFor === it.id ? (
              <form
                className="flex gap-2 pl-8"
                onSubmit={(e) => {
                  e.preventDefault();
                  run(() => updateEntryItemAction(it.id, { notes: note }), { success: l.noteSaved, onOk: () => setNoteFor(null) });
                }}
              >
                <Input value={note} onChange={(e) => setNote(e.target.value)} aria-label={l.itemNotes} placeholder={l.itemNotes} />
                <Button type="submit" size="sm" disabled={pending}>
                  {sp.common.save}
                </Button>
              </form>
            ) : null}
          </li>
        ))}
      </ul>
      <p className="text-xs text-muted-foreground tabular">
        {l.pctDone(Math.round(entry.completionPercentage), Math.round(entry.kpiScore))}
      </p>
      {canEdit ? (
        <div className="flex flex-wrap gap-2 border-t pt-3">
          <Button size="sm" disabled={pending} onClick={() => run(() => submitEntryAction(entry.id), { success: l.submitted, onOk: onDone })}>
            <Send /> {l.submit}
          </Button>
          <ConfirmAction
            trigger={
              <Button size="sm" variant="destructive">
                <Trash2 /> {l.deleteDraft}
              </Button>
            }
            title={sp.common.confirmDelete}
            description={sp.common.confirmDeleteBody}
            confirmLabel={sp.common.delete}
            destructive
            onConfirm={() => run(() => deleteEntryAction(entry.id), { success: l.deleted, onOk: onDone })}
          />
        </div>
      ) : null}
      {canReview && entry.status === "submitted" ? (
        <div className="flex flex-col gap-2 border-t pt-3">
          <Textarea value={reviewNotes} onChange={(e) => setReviewNotes(e.target.value)} placeholder={l.reviewNotes} aria-label={l.reviewNotes} />
          <div className="flex gap-2">
            <Button
              size="sm"
              disabled={pending}
              onClick={() => run(() => reviewEntryAction(entry.id, "reviewed", reviewNotes || undefined), { success: l.reviewedOk, onOk: onDone })}
            >
              {l.markReviewed}
            </Button>
            <Button
              size="sm"
              variant="destructive"
              disabled={pending}
              onClick={() => run(() => reviewEntryAction(entry.id, "rejected", reviewNotes), { success: l.rejectedOk, onOk: onDone })}
            >
              {l.reject}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

// ── History ───────────────────────────────────────────────────

export function HistoryTab({
  entries,
  pagination,
  canReview,
  deptNames,
  toolbar,
}: {
  entries: LogbookEntry[];
  pagination: ServerPage;
  canReview: boolean;
  deptNames: Record<string, string>;
  toolbar: React.ReactNode;
}) {
  const [open, setOpen] = React.useState<LogbookEntry | null>(null);
  const columns: ColumnDef<LogbookEntry, unknown>[] = [
    { accessorKey: "entryDate", header: sp.common.date, cell: ({ row }) => <span className="tabular">{formatDate(row.original.entryDate)}</span> },
    {
      id: "title",
      accessorFn: (e) => e.title,
      header: l.template,
      cell: ({ row }) => (
        <span className="min-w-0">
          <span className="block truncate font-medium">{row.original.templateName || row.original.title}</span>
          <span className="block truncate text-xs text-muted-foreground">
            {[row.original.employeeName, deptLabel(row.original, deptNames)].filter(Boolean).join(" · ")}
          </span>
        </span>
      ),
    },
    {
      id: "completion",
      accessorFn: (e) => e.completionPercentage,
      header: l.completion,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ getValue }) => <span className="tabular">{Math.round(getValue() as number)}%</span>,
    },
    {
      id: "kpi",
      accessorFn: (e) => e.kpiScore,
      header: l.kpiScore,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ getValue }) => <span className="tabular">{Math.round(getValue() as number)}%</span>,
    },
    { accessorKey: "status", header: sp.common.status, cell: ({ row }) => <EntryStatus status={row.original.status} /> },
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={entries}
        pagination={pagination}
        toolbar={toolbar}
        emptyMessage={l.noHistory}
        getRowId={(e) => e.id}
        onRowClick={(row) => setOpen(row.original)}
      />
      <Dialog open={!!open} onOpenChange={(o) => !o && setOpen(null)}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{l.reviewTitle}</DialogTitle>
            <DialogDescription>{open ? formatDate(open.entryDate) : null}</DialogDescription>
          </DialogHeader>
          {open ? <EntryView entry={open} canReview={canReview} onDone={() => setOpen(null)} /> : null}
        </DialogContent>
      </Dialog>
    </>
  );
}

// ── Templates ─────────────────────────────────────────────────

export function TemplatesTab({
  templates,
  departments,
  defaultDept,
  canManage,
  deptNames,
}: {
  templates: LogbookTemplate[];
  departments: Department[];
  defaultDept: string;
  canManage: boolean;
  deptNames: Record<string, string>;
}) {
  const { run } = useErpAction();
  const [creating, setCreating] = React.useState(false);
  return (
    <div className="flex flex-col gap-(--gap-inner)">
      {canManage ? (
        <Button className="self-start" onClick={() => setCreating(true)}>
          <Plus className="size-4" /> {l.newTemplate}
        </Button>
      ) : null}
      {templates.length === 0 ? (
        <ErpEmpty title={l.noTemplates} />
      ) : (
        <div className="grid grid-cols-1 gap-(--gap) md:grid-cols-2 xl:grid-cols-3">
          {templates.map((t) => (
            <Card key={t.id} className={cn("gap-2 p-(--card-pad)", !t.isActive && "opacity-60")}>
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="font-semibold">{t.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {deptLabel(t, deptNames)} · {l.frequency[t.frequency]} · {l.itemsCount(t.items.length)}
                  </p>
                </div>
                {canManage ? (
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
                    onConfirm={() =>
                      run(() => deleteTemplateAction(t.id), {
                        onOk: (r) => toast.success(r.archived ? l.templateArchived : l.templateDeleted),
                      })
                    }
                  />
                ) : null}
              </div>
              {t.description ? <p className="text-xs text-muted-foreground">{t.description}</p> : null}
              <ul className="list-disc pl-4 text-sm">
                {[...t.items]
                  .sort((a, b) => a.sortOrder - b.sortOrder)
                  .map((it) => (
                    <li key={it.id}>
                      {it.title} <span className="text-xs text-muted-foreground tabular">({formatNumber(it.weight, 1)})</span>
                    </li>
                  ))}
              </ul>
            </Card>
          ))}
        </div>
      )}
      {creating ? <TemplateDialog departments={departments} defaultDept={defaultDept} onClose={() => setCreating(false)} /> : null}
    </div>
  );
}

function TemplateDialog({ departments, defaultDept, onClose }: { departments: Department[]; defaultDept: string; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const [form, setForm] = React.useState({
    departmentId: defaultDept,
    name: "",
    description: "",
    frequency: "daily" as Frequency,
    items: [{ title: "", weight: "1", isRequired: true }],
  });
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((p) => ({ ...p, [k]: v }));
  const setItem = (i: number, p: Partial<(typeof form.items)[number]>) =>
    set(
      "items",
      form.items.map((it, j) => (j === i ? { ...it, ...p } : it)),
    );
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            run(
              () =>
                createTemplateAction({
                  departmentId: form.departmentId,
                  name: form.name,
                  description: form.description || undefined,
                  frequency: form.frequency,
                  items: form.items
                    .filter((it) => it.title.trim())
                    .map((it) => ({ title: it.title, weight: Number(it.weight) || 1, isRequired: it.isRequired })),
                }),
              { success: l.templateSaved, onOk: onClose, onFail: (r) => setErrors(r.fieldErrors ?? {}) },
            );
          }}
        >
          <DialogHeader>
            <DialogTitle>{l.newTemplate}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormRow label={sp.common.department} error={errors.departmentId?.[0]}>
              <Select value={form.departmentId} onValueChange={(v) => set("departmentId", v)}>
                <SelectTrigger className="w-full" aria-label={sp.common.department}>
                  <SelectValue placeholder={l.pickDepartment} />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((d) => (
                    <SelectItem key={d.id} value={d.id}>
                      {d.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={sp.common.period}>
              <Select value={form.frequency} onValueChange={(v) => set("frequency", v as Frequency)}>
                <SelectTrigger className="w-full" aria-label={sp.common.period}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(l.frequency).map(([v, label]) => (
                    <SelectItem key={v} value={v}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={l.templateName} htmlFor="tp-name" className="sm:col-span-2" error={errors.name?.[0]}>
              <Input id="tp-name" value={form.name} onChange={(e) => set("name", e.target.value)} required />
            </FormRow>
            <FormRow label={sp.kpi.description} htmlFor="tp-desc" className="sm:col-span-2">
              <Input id="tp-desc" value={form.description} onChange={(e) => set("description", e.target.value)} />
            </FormRow>
          </div>
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium">{l.items}</p>
            {errors.items?.[0] ? <p className="text-xs text-destructive">{errors.items[0]}</p> : null}
            {form.items.map((it, i) => (
              <div key={i} className="flex items-center gap-2">
                <Input value={it.title} placeholder={l.itemTitle} aria-label={l.itemTitle} onChange={(e) => setItem(i, { title: e.target.value })} />
                <Input
                  type="number"
                  min={0.1}
                  step="any"
                  className="w-20"
                  value={it.weight}
                  aria-label={l.weight}
                  onChange={(e) => setItem(i, { weight: e.target.value })}
                />
                <Switch checked={it.isRequired} onCheckedChange={(v) => setItem(i, { isRequired: v })} aria-label={l.required} />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label={sp.common.delete}
                  disabled={form.items.length === 1}
                  onClick={() => set("items", form.items.filter((_, j) => j !== i))}
                >
                  <X />
                </Button>
              </div>
            ))}
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="self-start"
              onClick={() => set("items", [...form.items, { title: "", weight: "1", isRequired: true }])}
            >
              <Plus className="size-4" /> {l.addItem}
            </Button>
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

// ── Summary ───────────────────────────────────────────────────

export function SummaryTab({ rows }: { rows: LogbookSummaryRow[] }) {
  if (rows.length === 0) return <ErpEmpty />;
  return (
    <Card className="py-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{sp.common.department}</TableHead>
            <TableHead className="text-right">{l.total}</TableHead>
            <TableHead className="hidden text-right sm:table-cell">{l.submittedCol}</TableHead>
            <TableHead className="hidden text-right sm:table-cell">{l.reviewedCol}</TableHead>
            <TableHead className="hidden text-right sm:table-cell">{l.rejectedCol}</TableHead>
            <TableHead className="text-right">{l.avgCompletion}</TableHead>
            <TableHead className="text-right">{l.avgKpi}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.departmentId}>
              <TableCell className="font-medium">{r.departmentName}</TableCell>
              <TableCell className="text-right tabular">{r.totalEntries}</TableCell>
              <TableCell className="hidden text-right tabular sm:table-cell">{r.submittedEntries}</TableCell>
              <TableCell className="hidden text-right tabular sm:table-cell">{r.reviewedEntries}</TableCell>
              <TableCell className="hidden text-right tabular sm:table-cell">{r.rejectedEntries}</TableCell>
              <TableCell className="text-right tabular">{formatNumber(r.avgCompletion, 1)}%</TableCell>
              <TableCell className="text-right tabular">{formatNumber(r.avgKpiScore, 1)}%</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  );
}
