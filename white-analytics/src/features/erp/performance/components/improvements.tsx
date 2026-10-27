"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Check, Lock, Pencil, Plus, Sparkles, Trash2, X } from "lucide-react";
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
import { Textarea } from "@/components/ui/textarea";
import type { StatusKind } from "@/components/dashboard/status-badge";
import type { ColMeta } from "@/components/dashboard/data-table";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { formatDate } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  closeRetroAction,
  createImprovementAction,
  decideSuggestionAction,
  deleteImprovementAction,
  saveRetroAction,
  updateImprovementAction,
} from "../actions";
import { lines } from "../lib";
import { sp } from "../strings";
import type { Department, Improvement, ImprovementStatus, Person, Retro, RetroAction } from "../types";
import { ConfirmAction, FormRow } from "./common";

const im = sp.improvements;
const NONE = "__none__";

const STATUS_KINDS: Record<string, StatusKind> = {
  suggested: "info",
  open: "warning",
  in_progress: "info",
  done: "good",
  dropped: "neutral",
  declined: "neutral",
};
const SOURCE_KINDS: Record<string, StatusKind> = { ai: "info", manual: "neutral", retro: "good" };

function PersonSelect({
  value,
  onChange,
  people,
  label,
  emptyLabel,
}: {
  value: string;
  onChange: (v: string) => void;
  people: { id: string; name: string }[];
  label: string;
  emptyLabel: string;
}) {
  return (
    <Select value={value || NONE} onValueChange={(v) => onChange(v === NONE ? "" : v)}>
      <SelectTrigger className="w-full" aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={NONE}>{emptyLabel}</SelectItem>
        {people.map((p) => (
          <SelectItem key={p.id} value={p.id}>
            {p.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

// ── Items ─────────────────────────────────────────────────────

export function ImprovementsTable({
  items,
  people,
  departments,
  toolbar,
}: {
  items: Improvement[];
  people: Person[];
  departments: Department[];
  toolbar: React.ReactNode;
}) {
  const { pending, run } = useErpAction();
  const { set } = useUrlFilters(["status"]);
  const [editing, setEditing] = React.useState<Improvement | "new" | null>(null);
  const [accepting, setAccepting] = React.useState<Improvement | null>(null);
  const suggested = items.filter((i) => i.status === "suggested").length;

  const columns: ColumnDef<Improvement, unknown>[] = [
    {
      id: "title",
      accessorFn: (r) => `${r.title} ${r.description}`,
      header: im.titleLabel,
      cell: ({ row }) => (
        <span className="block max-w-md min-w-48 whitespace-normal">
          <span className="block font-medium">{row.original.title}</span>
          {row.original.description ? (
            <span className="line-clamp-2 text-xs text-muted-foreground">{row.original.description}</span>
          ) : null}
        </span>
      ),
    },
    {
      id: "subject",
      accessorFn: (r) => (r.scope === "employee" ? r.employeeName : r.departmentName),
      header: im.subject,
      cell: ({ row }) => (
        <span className="min-w-0">
          <span className="block truncate">{row.original.scope === "employee" ? row.original.employeeName : row.original.departmentName}</span>
          <span className="block text-xs text-muted-foreground">{im.scopes[row.original.scope]}</span>
        </span>
      ),
    },
    {
      accessorKey: "source",
      header: im.source,
      cell: ({ row }) => <ErpStatusBadge status={row.original.source} labels={im.sources} kinds={SOURCE_KINDS} />,
    },
    { id: "owner", accessorFn: (r) => r.ownerName || im.noOwner, header: im.owner },
    {
      id: "due",
      accessorFn: (r) => r.dueDate ?? "",
      header: im.dueDate,
      cell: ({ row }) => (
        <span className={cn("tabular", row.original.overdue ? "font-medium text-destructive" : "text-muted-foreground")}>
          {formatDate(row.original.dueDate)}
        </span>
      ),
    },
    {
      accessorKey: "status",
      header: sp.common.status,
      cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={im.statuses} kinds={STATUS_KINDS} />,
    },
    {
      id: "actions",
      header: sp.common.actions,
      enableSorting: false,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => {
        const it = row.original;
        return (
          <span className="flex justify-end gap-1">
            {it.status === "suggested" ? (
              <>
                <Button size="xs" onClick={() => setAccepting(it)}>
                  <Check /> {im.accept}
                </Button>
                <Button
                  size="xs"
                  variant="ghost"
                  disabled={pending}
                  onClick={() => run(() => decideSuggestionAction(it.id, false), { success: im.declined })}
                >
                  <X /> {im.decline}
                </Button>
              </>
            ) : (
              <Button variant="ghost" size="icon-xs" aria-label={im.editItem} onClick={() => setEditing(it)}>
                <Pencil />
              </Button>
            )}
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
              onConfirm={() => run(() => deleteImprovementAction(it.id), { success: im.deleted })}
            />
          </span>
        );
      },
    },
  ];

  return (
    <div className="flex flex-col gap-(--gap-inner)">
      {suggested > 0 ? (
        <div className="flex flex-wrap items-center gap-3 rounded-2xl bg-[color-mix(in_oklab,var(--chart-1)_10%,transparent)] px-4 py-3 text-sm">
          <Sparkles className="size-4 text-[var(--chart-1)]" />
          <span className="flex-1">{im.suggestedBanner(suggested)}</span>
          <Button size="xs" variant="secondary" onClick={() => set({ status: "suggested" })}>
            {im.showSuggested}
          </Button>
        </div>
      ) : null}
      <ErpDataTable
        columns={columns}
        data={items}
        emptyMessage={im.empty}
        exportName="perbaikan"
        getRowId={(r) => r.id}
        toolbar={
          <div className="flex flex-wrap items-center gap-2">
            {toolbar}
            <Button onClick={() => setEditing("new")}>
              <Plus className="size-4" /> {im.newItem}
            </Button>
          </div>
        }
      />
      {editing ? (
        <ImprovementDialog item={editing === "new" ? null : editing} people={people} departments={departments} onClose={() => setEditing(null)} />
      ) : null}
      {accepting ? <AcceptDialog item={accepting} people={people} onClose={() => setAccepting(null)} /> : null}
    </div>
  );
}

function ImprovementDialog({
  item,
  people,
  departments,
  onClose,
}: {
  item: Improvement | null;
  people: Person[];
  departments: Department[];
  onClose: () => void;
}) {
  const { pending, run } = useErpAction();
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const [form, setForm] = React.useState({
    scope: item?.scope ?? ("employee" as "employee" | "department"),
    employeeId: item?.employeeId ?? "",
    departmentId: item?.departmentId ?? "",
    title: item?.title ?? "",
    description: item?.description ?? "",
    ownerEmployeeId: item?.ownerEmployeeId ?? "",
    dueDate: item?.dueDate ?? "",
    status: (item?.status ?? "open") as ImprovementStatus,
  });
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((p) => ({ ...p, [k]: v }));
  const submit = () => {
    const common = {
      title: form.title,
      description: form.description,
      ownerEmployeeId: form.ownerEmployeeId || undefined,
      dueDate: form.dueDate || undefined,
    };
    run(
      () =>
        item
          ? updateImprovementAction(item.id, {
              ...common,
              status: form.status as "open" | "in_progress" | "done" | "dropped",
            })
          : createImprovementAction({
              ...common,
              scope: form.scope,
              employeeId: form.scope === "employee" ? form.employeeId || undefined : undefined,
              departmentId: form.scope === "department" ? form.departmentId || undefined : undefined,
            }),
      { success: item ? im.updated : im.created, onOk: onClose, onFail: (r) => setErrors(r.fieldErrors ?? {}) },
    );
  };
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
            <DialogTitle>{item ? im.editItem : im.newItem}</DialogTitle>
            {item ? <DialogDescription>{im.sources[item.source]}</DialogDescription> : null}
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            {!item ? (
              <>
                <FormRow label={im.scope}>
                  <Select value={form.scope} onValueChange={(v) => set("scope", v as typeof form.scope)}>
                    <SelectTrigger className="w-full" aria-label={im.scope}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(im.scopes).map(([v, l]) => (
                        <SelectItem key={v} value={v}>
                          {l}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormRow>
                <FormRow label={im.subject} error={errors.subject?.[0]}>
                  <PersonSelect
                    value={form.scope === "employee" ? form.employeeId : form.departmentId}
                    onChange={(v) => set(form.scope === "employee" ? "employeeId" : "departmentId", v)}
                    people={form.scope === "employee" ? people : departments}
                    label={im.subject}
                    emptyLabel={im.subjectRequired}
                  />
                </FormRow>
              </>
            ) : null}
            <FormRow label={im.titleLabel} htmlFor="im-title" className="sm:col-span-2" error={errors.title?.[0]}>
              <Input id="im-title" value={form.title} onChange={(e) => set("title", e.target.value)} required />
            </FormRow>
            <FormRow label={im.description} htmlFor="im-desc" className="sm:col-span-2">
              <Textarea id="im-desc" value={form.description} onChange={(e) => set("description", e.target.value)} />
            </FormRow>
            <FormRow label={im.owner}>
              <PersonSelect value={form.ownerEmployeeId} onChange={(v) => set("ownerEmployeeId", v)} people={people} label={im.owner} emptyLabel={im.noOwner} />
            </FormRow>
            <FormRow label={im.dueDate}>
              <DatePicker value={form.dueDate} onChange={(v) => set("dueDate", v)} clearable />
            </FormRow>
            {item ? (
              <FormRow label={sp.common.status}>
                <Select value={form.status} onValueChange={(v) => set("status", v as ImprovementStatus)}>
                  <SelectTrigger className="w-full" aria-label={sp.common.status}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(["open", "in_progress", "done", "dropped"] as const).map((v) => (
                      <SelectItem key={v} value={v}>
                        {im.statuses[v]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormRow>
            ) : null}
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

function AcceptDialog({ item, people, onClose }: { item: Improvement; people: Person[]; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [owner, setOwner] = React.useState(item.ownerEmployeeId ?? item.employeeId ?? "");
  const [due, setDue] = React.useState(item.dueDate ?? "");
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{im.acceptTitle}</DialogTitle>
          <DialogDescription>{item.title}</DialogDescription>
        </DialogHeader>
        {item.description ? <p className="text-sm text-muted-foreground">{item.description}</p> : null}
        <div className="grid gap-3">
          <FormRow label={im.owner}>
            <PersonSelect value={owner} onChange={setOwner} people={people} label={im.owner} emptyLabel={im.noOwner} />
          </FormRow>
          <FormRow label={im.dueDate}>
            <DatePicker value={due} onChange={setDue} clearable />
          </FormRow>
        </div>
        <DialogFooter>
          <Button
            disabled={pending}
            onClick={() =>
              run(() => decideSuggestionAction(item.id, true, { ownerEmployeeId: owner || undefined, dueDate: due || undefined }), {
                success: im.accepted,
                onOk: onClose,
              })
            }
          >
            <Check className="size-4" /> {im.accept}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Retros ────────────────────────────────────────────────────

export function RetrosPanel({
  retros,
  departments,
  people,
  defaultDept,
}: {
  retros: Retro[];
  departments: Department[];
  people: Person[];
  defaultDept: string;
}) {
  const [editing, setEditing] = React.useState<Retro | "new" | null>(null);
  const now = new Date();
  const sorted = [...retros].sort((a, b) => b.periodYear * 12 + b.periodMonth - (a.periodYear * 12 + a.periodMonth));
  return (
    <div className="flex flex-col gap-(--gap-inner)">
      <Button className="self-start" onClick={() => setEditing("new")} disabled={departments.length === 0}>
        <Plus className="size-4" /> {im.newRetro}
      </Button>
      {sorted.length === 0 ? (
        <ErpEmpty title={im.noRetros} />
      ) : (
        <div className="grid grid-cols-1 gap-(--gap) lg:grid-cols-2">
          {sorted.map((r) => (
            <RetroCard key={r.id} retro={r} onEdit={() => setEditing(r)} />
          ))}
        </div>
      )}
      {editing ? (
        <RetroDialog
          retro={
            editing === "new"
              ? {
                  id: "",
                  departmentId: defaultDept,
                  departmentName: "",
                  periodYear: now.getFullYear(),
                  periodMonth: now.getMonth() + 1,
                  wentWell: [],
                  toImprove: [],
                  actions: [],
                  status: "draft",
                }
              : editing
          }
          departments={departments}
          people={people}
          onClose={() => setEditing(null)}
        />
      ) : null}
    </div>
  );
}

function RetroCard({ retro, onEdit }: { retro: Retro; onEdit: () => void }) {
  const { pending, run } = useErpAction();
  const list = (title: string, items: string[]) =>
    items.length > 0 ? (
      <div className="text-sm">
        <p className="text-xs font-medium text-muted-foreground">{title}</p>
        <ul className="list-disc pl-4">
          {items.map((s, i) => (
            <li key={i}>{s}</li>
          ))}
        </ul>
      </div>
    ) : null;
  return (
    <Card className="gap-3 p-(--card-pad)">
      <div className="flex items-start justify-between gap-2">
        <p className="font-semibold">{im.retroFor(retro.departmentName, sp.monthLabel(retro.periodYear, retro.periodMonth))}</p>
        <ErpStatusBadge status={retro.status} labels={im.retroStatus} kinds={{ draft: "neutral", closed: "good" }} />
      </div>
      {list(im.wentWell, retro.wentWell)}
      {list(im.toImprove, retro.toImprove)}
      {retro.actions.length > 0 ? (
        <div className="text-sm">
          <p className="text-xs font-medium text-muted-foreground">{im.retroActions}</p>
          <ul className="flex flex-col gap-1">
            {retro.actions.map((a, i) => (
              <li key={i} className="flex flex-wrap items-center gap-x-2">
                <span>{a.description}</span>
                {a.dueDate ? <span className="text-xs text-muted-foreground">{formatDate(a.dueDate)}</span> : null}
                {a.improvementId ? <span className="text-xs text-[var(--status-good)]">{im.linked}</span> : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {retro.status === "draft" ? (
        <div className="flex flex-wrap gap-2 border-t pt-3">
          <Button size="sm" variant="outline" onClick={onEdit}>
            <Pencil className="size-4" /> {sp.common.edit}
          </Button>
          <ConfirmAction
            trigger={
              <Button size="sm" disabled={pending}>
                <Lock className="size-4" /> {im.closeRetro}
              </Button>
            }
            title={im.closeRetroTitle}
            description={im.closeRetroBody}
            confirmLabel={im.closeRetro}
            onConfirm={() => run(() => closeRetroAction(retro.id), { onOk: (r) => toast.success(im.retroClosed(r.improvementsFiled)) })}
          />
        </div>
      ) : null}
    </Card>
  );
}

function RetroDialog({
  retro,
  departments,
  people,
  onClose,
}: {
  retro: Retro;
  departments: Department[];
  people: Person[];
  onClose: () => void;
}) {
  const { pending, run } = useErpAction();
  const isNew = !retro.id;
  const [dept, setDept] = React.useState(retro.departmentId);
  const [month, setMonth] = React.useState(`${retro.periodYear}-${String(retro.periodMonth).padStart(2, "0")}`);
  const [wentWell, setWentWell] = React.useState(retro.wentWell.join("\n"));
  const [toImprove, setToImprove] = React.useState(retro.toImprove.join("\n"));
  const [actions, setActions] = React.useState<RetroAction[]>(retro.actions);
  const members = people.filter((p) => !dept || p.departmentId === dept);
  const patch = (i: number, p: Partial<RetroAction>) => setActions((list) => list.map((a, j) => (j === i ? { ...a, ...p } : a)));
  const [y, m] = month.split("-").map(Number);
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isNew ? im.newRetro : im.retroFor(retro.departmentName, sp.monthLabel(retro.periodYear, retro.periodMonth))}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-4 sm:grid-cols-2">
          <FormRow label={sp.common.department}>
            <Select value={dept} onValueChange={setDept} disabled={!isNew}>
              <SelectTrigger className="w-full" aria-label={sp.common.department}>
                <SelectValue placeholder={sp.logbook.pickDepartment} />
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
          <FormRow label={sp.common.month} htmlFor="rt-month">
            <Input id="rt-month" type="month" value={month} disabled={!isNew} onChange={(e) => setMonth(e.target.value)} />
          </FormRow>
          <FormRow label={im.wentWell} htmlFor="rt-well" hint={sp.feedback.onePerLine}>
            <Textarea id="rt-well" rows={4} value={wentWell} onChange={(e) => setWentWell(e.target.value)} />
          </FormRow>
          <FormRow label={im.toImprove} htmlFor="rt-imp" hint={sp.feedback.onePerLine}>
            <Textarea id="rt-imp" rows={4} value={toImprove} onChange={(e) => setToImprove(e.target.value)} />
          </FormRow>
        </div>
        <div className="flex flex-col gap-2">
          <p className="text-sm font-medium">{im.retroActions}</p>
          {actions.map((a, i) => (
            <div key={i} className="grid gap-2 rounded-xl border p-2 sm:grid-cols-[minmax(0,1fr)_10rem_10rem_auto]">
              <Input value={a.description} placeholder={im.actionDesc} aria-label={im.actionDesc} onChange={(e) => patch(i, { description: e.target.value })} />
              <PersonSelect
                value={a.ownerEmployeeId ?? ""}
                onChange={(v) => patch(i, { ownerEmployeeId: v || undefined })}
                people={members}
                label={im.owner}
                emptyLabel={im.noOwner}
              />
              <DatePicker value={a.dueDate ?? ""} onChange={(v) => patch(i, { dueDate: v || undefined })} clearable />
              <Button variant="ghost" size="icon-sm" aria-label={sp.common.delete} onClick={() => setActions((l) => l.filter((_, j) => j !== i))}>
                <X />
              </Button>
            </div>
          ))}
          <Button
            variant="outline"
            size="sm"
            className="self-start"
            onClick={() => setActions((l) => [...l, { description: "" }])}
          >
            <Plus className="size-4" /> {im.addAction}
          </Button>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {sp.common.cancel}
          </Button>
          <Button
            disabled={pending || !dept}
            onClick={() =>
              run(
                () =>
                  saveRetroAction({
                    departmentId: dept,
                    periodYear: y,
                    periodMonth: m,
                    wentWell: lines(wentWell),
                    toImprove: lines(toImprove),
                    actions: actions
                      .filter((a) => a.description.trim())
                      .map((a) => ({ description: a.description, ownerEmployeeId: a.ownerEmployeeId, dueDate: a.dueDate })),
                  }),
                { success: im.retroSaved, onOk: onClose },
              )
            }
          >
            {im.saveRetro}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
