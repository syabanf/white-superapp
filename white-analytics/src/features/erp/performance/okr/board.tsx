"use client";

import * as React from "react";
import Link from "next/link";
import { ChevronLeft, ChevronRight, Pencil, Plus, Trash2, TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Textarea } from "@/components/ui/textarea";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, type FieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatCurrency, formatNumber } from "@/lib/format";
import { checkInAction, deleteKeyResultAction, deleteObjectiveAction, saveKeyResultAction, saveObjectiveAction } from "./actions";
import { shiftQuarter, to } from "./strings";
import type { KeyResult, Objective, OkrBoard } from "./types";

const statusKinds = { active: "info", done: "good", cancelled: "neutral" } as const;
const opts = (r: Record<string, string>) => Object.entries(r).map(([value, label]) => ({ value, label }));
const pct = (v: number) => `${Math.round(v * 100)}%`;
const num = (v: string) => Number(v.replace(/\./g, "").replace(",", "."));

function show(k: KeyResult, v: number): string {
  if (k.metric === "currency") return formatCurrency(v);
  return `${formatNumber(v, 2)}${k.metric === "percent" ? "%" : k.unit ? ` ${k.unit}` : ""}`;
}

function ObjectiveForm({ board, period, objective, close }: { board: OkrBoard; period: string; objective?: Objective; close: () => void }) {
  const levels = opts(to.levels).filter(
    (l) => (l.value === "company" && board.canCompany) || (l.value === "department" && board.departments.length > 0) || (l.value === "employee" && board.employees.length > 0),
  );
  const [v, setV] = React.useState({
    id: objective?.id ?? "",
    period,
    level: objective?.level ?? levels.at(-1)?.value ?? "employee",
    departmentId: objective?.departmentId ?? board.departments[0]?.id ?? "",
    employeeId: objective?.employeeId ?? board.employees[0]?.id ?? "",
    parentId: objective?.parentId ?? "",
    title: objective?.title ?? "",
    description: objective?.description ?? "",
    status: objective?.status ?? "active",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  const parents = board.objectives.filter((o) => o.id !== v.id && o.level !== "employee").map((o) => ({ value: o.id, label: o.title }));
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveObjectiveAction(v), { success: to.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      {objective ? null : (
        <>
          <Fld label={to.level} htmlFor="ob-level" error={errors?.level}>
            <SimpleSelect id="ob-level" value={v.level} onChange={(x) => set({ level: x as Objective["level"] })} options={levels} />
          </Fld>
          {v.level === "department" ? (
            <Fld label={to.department} htmlFor="ob-dept" error={errors?.departmentId}>
              <SimpleSelect id="ob-dept" value={v.departmentId} onChange={(x) => set({ departmentId: x })} options={board.departments.map((d) => ({ value: d.id, label: d.name }))} />
            </Fld>
          ) : null}
          {v.level === "employee" ? (
            <Fld label={to.employee} htmlFor="ob-emp" error={errors?.employeeId}>
              <SimpleSelect id="ob-emp" value={v.employeeId} onChange={(x) => set({ employeeId: x })} options={board.employees.map((d) => ({ value: d.id, label: d.name }))} />
            </Fld>
          ) : null}
        </>
      )}
      <Fld label={to.titleField} htmlFor="ob-title" error={errors?.title}>
        <Input id="ob-title" value={v.title} onChange={(e) => set({ title: e.target.value })} required />
      </Fld>
      <Fld label={to.description} htmlFor="ob-desc">
        <Textarea id="ob-desc" rows={2} value={v.description} onChange={(e) => set({ description: e.target.value })} />
      </Fld>
      <Fld label={to.parent} htmlFor="ob-parent" error={errors?.parentId}>
        <SimpleSelect id="ob-parent" value={v.parentId} onChange={(x) => set({ parentId: x })} options={parents} emptyLabel={to.noParent} />
      </Fld>
      {objective ? (
        <Fld label={to.status} htmlFor="ob-status">
          <SimpleSelect id="ob-status" value={v.status} onChange={(x) => set({ status: x as Objective["status"] })} options={opts(to.statuses)} />
        </Fld>
      ) : null}
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function KeyResultForm({ objectiveId, kr, close }: { objectiveId: string; kr?: KeyResult; close: () => void }) {
  const [v, setV] = React.useState({
    title: kr?.title ?? "",
    metric: kr?.metric ?? "number",
    start: kr ? String(kr.startValue) : "0",
    target: kr ? String(kr.targetValue) : "",
    unit: kr?.unit ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () => saveKeyResultAction({ id: kr?.id ?? "", objectiveId, title: v.title, metric: v.metric, startValue: num(v.start), targetValue: num(v.target), unit: v.unit }),
          { success: to.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) },
        );
      }}
    >
      <Fld label={to.krTitle} htmlFor="kr-title" error={errors?.title} className="sm:col-span-2">
        <Input id="kr-title" value={v.title} onChange={(e) => set({ title: e.target.value })} required />
      </Fld>
      <Fld label={to.metric} htmlFor="kr-metric">
        <SimpleSelect id="kr-metric" value={v.metric} onChange={(x) => set({ metric: x as KeyResult["metric"] })} options={opts(to.metrics)} />
      </Fld>
      <Fld label={to.unit} htmlFor="kr-unit">
        <Input id="kr-unit" value={v.unit} onChange={(e) => set({ unit: e.target.value })} disabled={v.metric !== "number"} />
      </Fld>
      <Fld label={to.start} htmlFor="kr-start">
        <Input id="kr-start" inputMode="decimal" value={v.start} onChange={(e) => set({ start: e.target.value })} />
      </Fld>
      <Fld label={to.target} htmlFor="kr-target" error={errors?.targetValue}>
        <Input id="kr-target" inputMode="decimal" value={v.target} onChange={(e) => set({ target: e.target.value })} required />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function CheckInForm({ kr, close }: { kr: KeyResult; close: () => void }) {
  const [v, setV] = React.useState({ value: String(kr.currentValue), confidence: String(kr.confidence || 7), note: "" });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => checkInAction(kr.id, { value: num(v.value), confidence: Number(v.confidence), note: v.note }), {
          success: to.checkedIn,
          onOk: close,
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <p className="text-sm text-muted-foreground">
        {show(kr, kr.startValue)} → {show(kr, kr.targetValue)}
      </p>
      <Fld label={to.value} htmlFor="ci-value">
        <Input id="ci-value" inputMode="decimal" value={v.value} onChange={(e) => setV({ ...v, value: e.target.value })} autoFocus />
      </Fld>
      <Fld label={to.confidence} htmlFor="ci-conf" error={errors?.confidence}>
        <SimpleSelect id="ci-conf" value={v.confidence} onChange={(x) => setV({ ...v, confidence: x })} options={Array.from({ length: 10 }, (_, i) => ({ value: String(i + 1), label: String(i + 1) }))} />
      </Fld>
      <Fld label={to.note} htmlFor="ci-note">
        <Textarea id="ci-note" rows={2} value={v.note} onChange={(e) => setV({ ...v, note: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={to.checkIn} />
    </form>
  );
}

function KeyResultRow({ kr, canEdit, active }: { kr: KeyResult; canEdit: boolean; active: boolean }) {
  return (
    <li className="flex flex-col gap-1.5 py-2.5">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-sm">{kr.title}</p>
          <p className="text-xs text-muted-foreground tabular-nums">
            {show(kr, kr.currentValue)} / {show(kr, kr.targetValue)}
            {kr.confidence ? ` · ${to.confidenceShort(kr.confidence)}` : ""}
          </p>
        </div>
        {canEdit ? (
          <span className="flex shrink-0 gap-0.5">
            {active ? (
              <FormDialog title={`${to.checkInTitle} · ${kr.title}`} trigger={<Button size="xs" variant="ghost"><TrendingUp className="size-3.5" /> {to.checkIn}</Button>}>
                {(close) => <CheckInForm kr={kr} close={close} />}
              </FormDialog>
            ) : null}
            <FormDialog title={to.editKr} wide trigger={<Button size="icon-xs" variant="ghost" aria-label={to.editKr}><Pencil className="size-3.5" /></Button>}>
              {(close) => <KeyResultForm objectiveId={kr.objectiveId} kr={kr} close={close} />}
            </FormDialog>
            <ConfirmAction title={to.deleteKr} success={to.deleted} action={() => deleteKeyResultAction(kr.id)} trigger={<Button size="icon-xs" variant="ghost" aria-label={to.delete}><Trash2 className="size-3.5" /></Button>} />
          </span>
        ) : null}
      </div>
      <div className="flex items-center gap-2">
        <Progress value={kr.progress * 100} className="h-1.5" aria-label={to.progress} />
        <span className="w-10 text-right text-xs tabular-nums text-muted-foreground">{pct(kr.progress)}</span>
      </div>
    </li>
  );
}

function ObjectiveCard({ o, board }: { o: Objective; board: OkrBoard }) {
  const parent = board.objectives.find((p) => p.id === o.parentId);
  return (
    <Card className="gap-2 p-(--card-pad)">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-xs text-muted-foreground">
            {to.levels[o.level]}
            {o.ownerName ? ` · ${o.ownerName}` : ""}
            {parent ? ` · ↑ ${parent.title}` : ""}
          </p>
          <h3 className="font-semibold">{o.title}</h3>
          {o.description ? <p className="text-sm text-muted-foreground">{o.description}</p> : null}
        </div>
        <div className="flex items-center gap-1">
          <ErpStatusBadge status={o.status} labels={to.statuses} kinds={statusKinds} />
          <span className="text-lg font-semibold tabular-nums">{pct(o.progress)}</span>
        </div>
      </div>
      {o.keyResults.length === 0 ? (
        <p className="text-sm text-muted-foreground">{to.noKr}</p>
      ) : (
        <ul className="divide-y">
          {o.keyResults.map((k) => (
            <KeyResultRow key={k.id} kr={k} canEdit={o.canEdit} active={o.status === "active"} />
          ))}
        </ul>
      )}
      {o.canEdit ? (
        <div className="flex flex-wrap gap-1">
          <FormDialog title={to.addKr} wide trigger={<Button size="xs" variant="outline"><Plus className="size-3.5" /> {to.addKr}</Button>}>
            {(close) => <KeyResultForm objectiveId={o.id} close={close} />}
          </FormDialog>
          <FormDialog title={to.editObjective} trigger={<Button size="xs" variant="ghost"><Pencil className="size-3.5" /> {to.editObjective}</Button>}>
            {(close) => <ObjectiveForm board={board} period={o.period} objective={o} close={close} />}
          </FormDialog>
          <ConfirmAction title={to.deleteObjective} success={to.deleted} action={() => deleteObjectiveAction(o.id)} trigger={<Button size="xs" variant="ghost"><Trash2 className="size-3.5" /> {to.delete}</Button>} />
        </div>
      ) : null}
    </Card>
  );
}

export function OkrBoardView({ board, period }: { board: OkrBoard; period: string }) {
  const canAdd = board.canCompany || board.departments.length > 0 || board.employees.length > 0;
  const groups = (["company", "department", "employee"] as const)
    .map((level) => ({ level, items: board.objectives.filter((o) => o.level === level) }))
    .filter((g) => g.items.length > 0);
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-1">
          <Button size="icon-sm" variant="outline" asChild aria-label={to.period}>
            <Link href={`?period=${shiftQuarter(period, -1)}`}>
              <ChevronLeft className="size-4" />
            </Link>
          </Button>
          <span className="px-2 text-sm font-medium tabular-nums">{period.replace("-", " ")}</span>
          <Button size="icon-sm" variant="outline" asChild aria-label={to.period}>
            <Link href={`?period=${shiftQuarter(period, 1)}`}>
              <ChevronRight className="size-4" />
            </Link>
          </Button>
        </div>
        {canAdd ? (
          <FormDialog title={to.addObjective} trigger={<Button size="sm"><Plus className="size-4" /> {to.addObjective}</Button>}>
            {(close) => <ObjectiveForm board={board} period={period} close={close} />}
          </FormDialog>
        ) : null}
      </div>
      {groups.length === 0 ? <p className="text-sm text-muted-foreground">{to.empty}</p> : null}
      {groups.map((g) => (
        <section key={g.level} className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold">{to.levels[g.level]}</h2>
          <div className="grid gap-3 lg:grid-cols-2">
            {g.items.map((o) => (
              <ObjectiveCard key={o.id} o={o} board={board} />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
