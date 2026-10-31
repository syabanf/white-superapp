"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { ColMeta } from "@/components/dashboard/data-table";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpEmpty } from "@/features/erp/shared/states";
import { useErpAction } from "@/features/erp/shared/client";
import { formatNumber } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  createTargetAction,
  deleteIndicatorAction,
  deleteTargetAction,
  saveDepartmentConfigAction,
  saveIndicatorAction,
  updateTargetAction,
  type IndicatorInput,
  type TargetInput,
} from "../actions";
import { sp } from "../strings";
import type { Department, Indicator, KpiConfig, KpiTarget, Person } from "../types";
import { ConfirmAction, FormRow } from "./common";

const k = sp.kpi;

// ── Indicators ────────────────────────────────────────────────

export function IndicatorsTable({ indicators, canWrite }: { indicators: Indicator[]; canWrite: boolean }) {
  const [editing, setEditing] = React.useState<Indicator | "new" | null>(null);
  const { run } = useErpAction();
  const columns: ColumnDef<Indicator, unknown>[] = [
    {
      id: "name",
      accessorFn: (r) => `${r.name} ${r.code}`,
      header: k.name,
      cell: ({ row }) => (
        <span className="min-w-0">
          <span className="block font-medium">{row.original.name}</span>
          <span className="block font-mono text-xs text-muted-foreground">{row.original.code}</span>
        </span>
      ),
    },
    { accessorKey: "unit", header: k.unit },
    { id: "direction", accessorFn: (r) => k.directions[r.direction], header: k.direction },
    { id: "source", accessorFn: (r) => k.sources[r.sourceKind], header: k.source },
    {
      id: "target",
      accessorFn: (r) => r.defaultTarget ?? null,
      header: k.defaultTarget,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => <span className="tabular">{formatNumber(row.original.defaultTarget, 2)}</span>,
    },
    {
      id: "active",
      accessorFn: (r) => (r.isActive ? k.active : k.inactive),
      header: sp.common.status,
      cell: ({ row }) => (
        <span className={cn("text-xs", row.original.isActive ? "text-[var(--status-good)]" : "text-muted-foreground")}>
          {row.original.isActive ? k.active : k.inactive}
        </span>
      ),
    },
    ...(canWrite
      ? [
          {
            id: "actions",
            header: sp.common.actions,
            enableSorting: false,
            meta: { align: "right" } satisfies ColMeta,
            cell: ({ row }) => (
              <span className="flex justify-end gap-1">
                <Button variant="ghost" size="icon-xs" aria-label={k.editIndicator} onClick={() => setEditing(row.original)}>
                  <Pencil />
                </Button>
                <ConfirmAction
                  trigger={
                    <Button variant="ghost" size="icon-xs" aria-label={k.deleteIndicator}>
                      <Trash2 />
                    </Button>
                  }
                  title={k.deleteIndicatorTitle(row.original.name)}
                  description={k.deleteIndicatorBody}
                  confirmLabel={sp.common.delete}
                  destructive
                  onConfirm={() => run(() => deleteIndicatorAction(row.original.id), { onOk: (r) => toast.success(r.message) })}
                />
              </span>
            ),
          } satisfies ColumnDef<Indicator, unknown>,
        ]
      : []),
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={indicators}
        getRowId={(r) => r.id}
        toolbar={
          canWrite ? (
            <Button onClick={() => setEditing("new")}>
              <Plus className="size-4" /> {k.newIndicator}
            </Button>
          ) : null
        }
      />
      {editing ? (
        <IndicatorDialog indicator={editing === "new" ? null : editing} onClose={() => setEditing(null)} />
      ) : null}
    </>
  );
}

function IndicatorDialog({ indicator, onClose }: { indicator: Indicator | null; onClose: () => void }) {
  const { pending, run } = useErpAction();
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const [form, setForm] = React.useState({
    code: indicator?.code ?? "",
    name: indicator?.name ?? "",
    description: indicator?.description ?? "",
    unit: indicator?.unit ?? "",
    direction: indicator?.direction ?? "higher_better",
    sourceKind: indicator?.sourceKind ?? "manual",
    defaultTarget: indicator?.defaultTarget != null ? String(indicator.defaultTarget) : "",
    isActive: indicator?.isActive ?? true,
  });
  const set = <K extends keyof typeof form>(key: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [key]: v }));
  const submit = () => {
    const input: IndicatorInput = {
      id: indicator?.id ?? "",
      code: form.code,
      name: form.name,
      description: form.description,
      unit: form.unit,
      direction: form.direction,
      sourceKind: form.sourceKind,
      defaultTarget: form.defaultTarget === "" ? undefined : Number(form.defaultTarget),
      isActive: form.isActive,
    };
    run(() => saveIndicatorAction(input), { success: k.indicatorSaved, onOk: onClose, onFail: (r) => setErrors(r.fieldErrors ?? {}) });
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
            <DialogTitle>{indicator ? k.editIndicator : k.newIndicator}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormRow label={k.code} htmlFor="ind-code" error={errors.code?.[0]}>
              <Input id="ind-code" value={form.code} disabled={!!indicator} onChange={(e) => set("code", e.target.value)} required />
            </FormRow>
            <FormRow label={k.unit} htmlFor="ind-unit">
              <Input id="ind-unit" value={form.unit} onChange={(e) => set("unit", e.target.value)} placeholder="%" />
            </FormRow>
            <FormRow label={k.name} htmlFor="ind-name" className="sm:col-span-2" error={errors.name?.[0]}>
              <Input id="ind-name" value={form.name} onChange={(e) => set("name", e.target.value)} required />
            </FormRow>
            <FormRow label={k.description} htmlFor="ind-desc" className="sm:col-span-2">
              <Input id="ind-desc" value={form.description} onChange={(e) => set("description", e.target.value)} />
            </FormRow>
            <FormRow label={k.direction}>
              <Select value={form.direction} onValueChange={(v) => set("direction", v as Indicator["direction"])}>
                <SelectTrigger className="w-full" aria-label={k.direction}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(k.directions).map(([v, l]) => (
                    <SelectItem key={v} value={v}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={k.source}>
              <Select value={form.sourceKind} onValueChange={(v) => set("sourceKind", v as Indicator["sourceKind"])}>
                <SelectTrigger className="w-full" aria-label={k.source}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(k.sources).map(([v, l]) => (
                    <SelectItem key={v} value={v}>
                      {l}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={k.defaultTarget} htmlFor="ind-target">
              <Input id="ind-target" type="number" step="any" value={form.defaultTarget} onChange={(e) => set("defaultTarget", e.target.value)} />
            </FormRow>
            <label className="flex items-center gap-2 self-end pb-2 text-sm">
              <Switch checked={form.isActive} onCheckedChange={(v) => set("isActive", v)} /> {k.active}
            </label>
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

// ── Department weights ───────────────────────────────────────

type Draft = Record<string, { enabled: boolean; weight: string }>;

function baseDraft(config: KpiConfig, departmentId: string): Draft {
  const out: Draft = {};
  for (const ind of config.indicators) out[ind.id] = { enabled: false, weight: "10" };
  for (const w of config.weights)
    if (w.departmentId === departmentId) out[w.indicatorId] = { enabled: true, weight: String(w.weight) };
  return out;
}

/** HR ticks the indicators that count for each department and sets their weights. */
export function DepartmentWeights({ config, canWrite }: { config: KpiConfig; canWrite: boolean }) {
  const [dept, setDept] = React.useState(config.departments[0]?.id ?? "");
  const [edits, setEdits] = React.useState<Record<string, Draft>>({});
  const { pending, run } = useErpAction();
  const draft = edits[dept] ?? baseDraft(config, dept);
  const configured = config.weights.some((w) => w.departmentId === dept);
  const indicators = config.indicators.filter((i) => i.isActive);
  const total = indicators.reduce((sum, i) => sum + (draft[i.id]?.enabled ? Number(draft[i.id].weight) || 0 : 0), 0);
  const deptName = config.departments.find((d) => d.id === dept)?.name ?? "";

  const patch = (indicatorId: string, p: Partial<Draft[string]>) =>
    setEdits((prev) => ({ ...prev, [dept]: { ...draft, [indicatorId]: { ...draft[indicatorId], ...p } } }));

  if (config.departments.length === 0) return <ErpEmpty />;
  return (
    <div className="flex flex-col gap-4">
      <p className="max-w-3xl text-sm text-muted-foreground">{k.configHint}</p>
      <div className="flex flex-wrap gap-2">
        {config.departments.map((d) => (
          <Button key={d.id} size="sm" variant={d.id === dept ? "default" : "outline"} onClick={() => setDept(d.id)}>
            {d.name}
          </Button>
        ))}
      </div>
      {!configured ? (
        <p className="rounded-xl bg-[color-mix(in_oklab,var(--status-warning)_12%,transparent)] px-3 py-2 text-xs">{k.notConfigured}</p>
      ) : null}
      <Card className="gap-0 divide-y py-0">
        {indicators.map((ind) => {
          const d = draft[ind.id] ?? { enabled: false, weight: "10" };
          return (
            <div key={ind.id} className="flex items-center gap-3 px-(--card-pad) py-3">
              <Checkbox
                checked={d.enabled}
                disabled={!canWrite}
                onCheckedChange={(v) => patch(ind.id, { enabled: v === true })}
                aria-label={k.enable(ind.name)}
              />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">
                  {ind.name}
                  <span className="ml-2 text-xs font-normal text-muted-foreground">
                    {ind.unit ? `${ind.unit} · ` : ""}
                    {k.directions[ind.direction]}
                  </span>
                </p>
                {ind.description ? <p className="truncate text-xs text-muted-foreground">{ind.description}</p> : null}
              </div>
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min={1}
                  max={100}
                  value={d.weight}
                  disabled={!d.enabled || !canWrite}
                  onChange={(e) => patch(ind.id, { weight: e.target.value })}
                  aria-label={k.weightOf(ind.name)}
                  className="h-8 w-20 text-right"
                />
                <span className="text-xs text-muted-foreground">%</span>
              </div>
            </div>
          );
        })}
      </Card>
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-card px-(--card-pad) py-3 shadow-(--card-shadow)">
        <p className="text-sm">
          {k.totalWeight}:{" "}
          <span className={cn("font-semibold tabular", total === 100 ? "text-[var(--status-good)]" : "text-[var(--status-warning)]")}>
            {total}%
          </span>
          {total !== 100 ? <span className="ml-2 text-xs text-muted-foreground">{k.weightNote}</span> : null}
        </p>
        {canWrite ? (
          <Button
            disabled={pending || !dept}
            onClick={() =>
              run(
                () =>
                  saveDepartmentConfigAction(
                    dept,
                    indicators.filter((i) => draft[i.id]?.enabled).map((i) => ({ indicatorId: i.id, weight: Number(draft[i.id].weight) })),
                  ),
                { success: sp.common.saved, onOk: () =>
                  setEdits((prev) => {
                    const next = { ...prev };
                    delete next[dept];
                    return next;
                  }),
              },
              )
            }
          >
            {pending ? sp.common.saving : k.saveFor(deptName)}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

// ── Targets ───────────────────────────────────────────────────

const ALL = "__all__";

type TargetForm = {
  indicatorId: string;
  scope: "global" | "department" | "employee";
  scopeId: string;
  period: "all" | "year" | "month";
  year: string;
  month: string;
  target: string;
};

function blankTarget(): TargetForm {
  const now = new Date();
  return {
    indicatorId: "",
    scope: "global",
    scopeId: "",
    period: "all",
    year: String(now.getFullYear()),
    month: String(now.getMonth() + 1),
    target: "",
  };
}

export function TargetsPanel({
  targets,
  indicators,
  departments,
  people,
  canWrite,
}: {
  targets: KpiTarget[];
  indicators: Indicator[];
  departments: Department[];
  people: Person[];
  canWrite: boolean;
}) {
  const { pending, run } = useErpAction();
  const [form, setForm] = React.useState(blankTarget);
  const [editingId, setEditingId] = React.useState<string | null>(null);
  const valueRef = React.useRef<HTMLInputElement>(null);
  const set = <K extends keyof TargetForm>(key: K, v: TargetForm[K]) => setForm((f) => ({ ...f, [key]: v }));
  const indName = new Map(indicators.map((i) => [i.id, i]));
  const deptName = new Map(departments.map((d) => [d.id, d.name]));
  const personName = new Map(people.map((p) => [p.id, p.name]));
  const selected = indName.get(form.indicatorId);
  const isPercent = selected?.unit === "%";

  const scopeLabel = (t: KpiTarget) =>
    t.employeeId
      ? `${k.scopeEmployee}: ${personName.get(t.employeeId) ?? t.employeeId}`
      : t.departmentId
        ? `${k.scopeDepartment}: ${deptName.get(t.departmentId) ?? t.departmentId}`
        : k.scopeGlobal;
  const periodLabel = (t: KpiTarget) =>
    t.periodYear && t.periodMonth
      ? sp.monthLabel(t.periodYear, t.periodMonth)
      : t.periodYear
        ? String(t.periodYear)
        : t.periodMonth
          ? k.everyYear(sp.months[t.periodMonth - 1])
          : k.periodAll;

  // Ubah loads the row into the form above; percent units show as 95, not 0.95.
  const startEdit = (t: KpiTarget) => {
    const shown = indName.get(t.indicatorId)?.unit === "%" ? t.target * 100 : t.target;
    setEditingId(t.id);
    setForm({
      ...blankTarget(),
      indicatorId: t.indicatorId,
      scope: t.employeeId ? "employee" : t.departmentId ? "department" : "global",
      scopeId: t.employeeId ?? t.departmentId ?? "",
      period: t.periodMonth ? "month" : t.periodYear ? "year" : "all",
      ...(t.periodYear ? { year: String(t.periodYear) } : {}),
      ...(t.periodMonth ? { month: String(t.periodMonth) } : {}),
      target: String(Number(shown.toFixed(6))),
    });
    requestAnimationFrame(() => valueRef.current?.focus());
  };
  const resetForm = () => {
    setEditingId(null);
    setForm(blankTarget());
  };

  const submit = () => {
    const raw = Number(form.target.replace(",", "."));
    const input: TargetInput = {
      indicatorId: form.indicatorId,
      target: Number.isFinite(raw) && form.target !== "" ? (isPercent ? raw / 100 : raw) : Number.NaN,
      periodYear: form.period === "all" ? undefined : Number(form.year),
      periodMonth: form.period === "month" ? Number(form.month) : undefined,
      departmentId: form.scope === "department" && form.scopeId ? form.scopeId : undefined,
      employeeId: form.scope === "employee" && form.scopeId ? form.scopeId : undefined,
    };
    if (editingId) run(() => updateTargetAction(editingId, input), { success: k.targetSaved, onOk: resetForm });
    else run(() => createTargetAction(input), { success: k.targetSaved, onOk: () => set("target", "") });
  };

  return (
    <div className="flex flex-col gap-4">
      {canWrite ? (
        <Card className="gap-3 p-(--card-pad)">
          <p className="text-sm font-semibold">{editingId ? k.editTarget : k.newTarget}</p>
          <div className="grid gap-4 md:grid-cols-3">
            <FormRow label={k.indicator}>
              <Select value={form.indicatorId || ALL} onValueChange={(v) => set("indicatorId", v === ALL ? "" : v)}>
                <SelectTrigger className="w-full" aria-label={k.indicator}>
                  <SelectValue placeholder={k.pickIndicator} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL} disabled>
                    {k.pickIndicator}
                  </SelectItem>
                  {indicators.map((i) => (
                    <SelectItem key={i.id} value={i.id}>
                      {i.name}
                      {i.unit ? ` (${i.unit})` : ""}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormRow>
            <FormRow label={k.scope}>
              <Select
                value={form.scope}
                onValueChange={(v) => setForm((f) => ({ ...f, scope: v as typeof f.scope, scopeId: "" }))}
              >
                <SelectTrigger className="w-full" aria-label={k.scope}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="global">{k.scopeGlobal}</SelectItem>
                  <SelectItem value="department">{k.scopeDepartment}</SelectItem>
                  <SelectItem value="employee">{k.scopeEmployee}</SelectItem>
                </SelectContent>
              </Select>
            </FormRow>
            {form.scope !== "global" ? (
              <FormRow label={form.scope === "department" ? sp.common.department : sp.common.employee}>
                <Select value={form.scopeId || ALL} onValueChange={(v) => set("scopeId", v === ALL ? "" : v)}>
                  <SelectTrigger className="w-full" aria-label={k.scope}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={ALL} disabled>
                      {sp.common.none}
                    </SelectItem>
                    {(form.scope === "department" ? departments : people).map((o) => (
                      <SelectItem key={o.id} value={o.id}>
                        {o.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormRow>
            ) : (
              <div className="hidden md:block" />
            )}
            <FormRow label={sp.common.period}>
              <Select value={form.period} onValueChange={(v) => set("period", v as TargetForm["period"])}>
                <SelectTrigger className="w-full" aria-label={sp.common.period}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{k.periodAll}</SelectItem>
                  <SelectItem value="year">{k.periodYear}</SelectItem>
                  <SelectItem value="month">{k.periodMonth}</SelectItem>
                </SelectContent>
              </Select>
            </FormRow>
            {form.period !== "all" ? (
              <div className="flex gap-2">
                <FormRow label={sp.common.year} htmlFor="tg-year" className="w-24">
                  <Input id="tg-year" type="number" value={form.year} onChange={(e) => set("year", e.target.value)} />
                </FormRow>
                {form.period === "month" ? (
                  <FormRow label={sp.common.month} className="flex-1">
                    <Select value={form.month} onValueChange={(v) => set("month", v)}>
                      <SelectTrigger className="w-full" aria-label={sp.common.month}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {sp.months.map((m, i) => (
                          <SelectItem key={m} value={String(i + 1)}>
                            {m}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormRow>
                ) : null}
              </div>
            ) : (
              <div className="hidden md:block" />
            )}
            <FormRow label={k.target} htmlFor="tg-value" hint={isPercent ? k.percentHint : undefined}>
              <div className="flex gap-2">
                <Input ref={valueRef} id="tg-value" inputMode="decimal" value={form.target} onChange={(e) => set("target", e.target.value)} />
                {editingId ? (
                  <Button variant="outline" onClick={resetForm}>
                    {sp.common.cancel}
                  </Button>
                ) : null}
                <Button disabled={pending || !form.indicatorId || form.target === ""} onClick={submit}>
                  {editingId ? sp.common.save : sp.common.add}
                </Button>
              </div>
            </FormRow>
          </div>
        </Card>
      ) : null}
      {targets.length === 0 ? (
        <ErpEmpty title={k.noTargets} compact />
      ) : (
        <Card className="py-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{k.indicator}</TableHead>
                <TableHead>{k.scope}</TableHead>
                <TableHead>{sp.common.period}</TableHead>
                <TableHead className="text-right">{k.target}</TableHead>
                {canWrite ? <TableHead className="text-right">{sp.common.actions}</TableHead> : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {targets.map((t) => {
                const ind = indName.get(t.indicatorId);
                return (
                  <TableRow key={t.id}>
                    <TableCell>{ind?.name ?? t.indicatorId}</TableCell>
                    <TableCell className="text-muted-foreground">{scopeLabel(t)}</TableCell>
                    <TableCell className="text-muted-foreground">{periodLabel(t)}</TableCell>
                    <TableCell className="text-right tabular">
                      {ind?.unit === "%" ? `${formatNumber(t.target * 100, 1)}%` : formatNumber(t.target, 2)}
                    </TableCell>
                    {canWrite ? (
                      <TableCell>
                        <span className="flex justify-end gap-1">
                          <Button variant="ghost" size="icon-xs" aria-label={k.editTarget} onClick={() => startEdit(t)}>
                            <Pencil />
                          </Button>
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
                              run(() => deleteTargetAction(t.id), {
                                success: k.targetDeleted,
                                onOk: () => {
                                  if (t.id === editingId) resetForm();
                                },
                              })
                            }
                          />
                        </span>
                      </TableCell>
                    ) : null}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </Card>
      )}
    </div>
  );
}

