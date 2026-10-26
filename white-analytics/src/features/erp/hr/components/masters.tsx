"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Trash2 } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { useErpAction } from "@/features/erp/shared/client";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import {
  deleteDepartmentAction,
  deleteEmploymentStatusAction,
  deletePositionAction,
  saveDepartmentAction,
  saveEmploymentStatusAction,
  savePositionAction,
} from "../actions";
import { th } from "../strings";
import type { Department, EmploymentStatusOption, Position } from "../types";
import {
  ConfirmAction,
  EmployeePicker,
  Fld,
  FormDialog,
  FormFooter,
  SimpleSelect,
  useFieldErrors,
  type EmployeeOption,
} from "./ui";

const m = th.masters;

function ActiveBadge({ active }: { active: boolean }) {
  return <ErpStatusBadge status={active ? "ACTIVE" : "INACTIVE"} />;
}

function ActiveSwitch({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-center gap-2">
      <Switch id="active" checked={value} onCheckedChange={onChange} />
      <Label htmlFor="active">{th.active}</Label>
    </div>
  );
}

function RowActions({
  canWrite,
  edit,
  onDelete,
}: {
  canWrite: boolean;
  edit: React.ReactNode;
  onDelete: () => Promise<ActionResult<unknown>>;
}) {
  if (!canWrite) return null;
  return (
    <div className="flex justify-end gap-1" onClick={(e) => e.stopPropagation()}>
      {edit}
      <ConfirmAction
        title={th.confirmDelete}
        success={th.deleted}
        confirmLabel={th.delete}
        action={onDelete}
        trigger={
          <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
            <Trash2 />
          </Button>
        }
      />
    </div>
  );
}

const editTrigger = (
  <Button size="icon-xs" variant="ghost" aria-label={th.edit}>
    <Pencil />
  </Button>
);

// Departments

export function DepartmentsPanel({
  departments,
  employees,
  canWrite,
}: {
  departments: Department[];
  employees: EmployeeOption[];
  canWrite: boolean;
}) {
  const names = Object.fromEntries(departments.map((d) => [d.id, d.name]));
  const empNames = Object.fromEntries(employees.map((e) => [e.id, e.label]));
  const dialog = (d?: Department) => (
    <FormDialog trigger={d ? editTrigger : <NewButton label={m.newDepartment} />} title={d ? d.name : m.newDepartment}>
      {(close) => <DepartmentForm dept={d} departments={departments} employees={employees} close={close} />}
    </FormDialog>
  );
  const columns: ColumnDef<Department, unknown>[] = [
    { accessorKey: "code", header: m.code, meta: { mono: true } },
    { accessorKey: "name", header: m.name },
    { id: "parent", header: m.parent, cell: ({ row }) => names[row.original.parentId ?? ""] ?? "–" },
    { id: "head", header: m.head, cell: ({ row }) => empNames[row.original.headEmployeeId ?? ""] ?? "–" },
    { accessorKey: "isActive", header: th.status, cell: ({ row }) => <ActiveBadge active={row.original.isActive} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <RowActions
          canWrite={canWrite}
          edit={dialog(row.original)}
          onDelete={() => deleteDepartmentAction(row.original.id)}
        />
      ),
    },
  ];
  return <ErpDataTable columns={columns} data={departments} toolbar={canWrite ? dialog() : null} />;
}

function NewButton({ label, ...props }: { label: string } & React.ComponentProps<typeof Button>) {
  return (
    <Button size="sm" {...props}>
      <Plus className="size-4" /> {label}
    </Button>
  );
}

function DepartmentForm({
  dept,
  departments,
  employees,
  close,
}: {
  dept?: Department;
  departments: Department[];
  employees: EmployeeOption[];
  close: () => void;
}) {
  const [v, setV] = React.useState({
    code: dept?.code ?? "",
    name: dept?.name ?? "",
    parentId: dept?.parentId ?? "",
    headEmployeeId: dept?.headEmployeeId ?? "",
    isActive: dept?.isActive ?? true,
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveDepartmentAction(dept?.id ?? null, v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <div className="grid gap-4 sm:grid-cols-[120px_1fr]">
        <Fld label={m.code} htmlFor="code" error={errors?.code}>
          <Input id="code" value={v.code} onChange={(e) => setV({ ...v, code: e.target.value.toUpperCase() })} />
        </Fld>
        <Fld label={m.name} htmlFor="name" error={errors?.name}>
          <Input id="name" value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} />
        </Fld>
      </div>
      <Fld label={m.parent}>
        <SimpleSelect
          value={v.parentId}
          onChange={(parentId) => setV({ ...v, parentId })}
          emptyLabel={th.none}
          options={departments.filter((d) => d.id !== dept?.id).map((d) => ({ value: d.id, label: d.name }))}
        />
      </Fld>
      <Fld label={m.head}>
        <EmployeePicker
          value={v.headEmployeeId}
          onChange={(headEmployeeId) => setV({ ...v, headEmployeeId })}
          options={employees}
          allowEmpty={th.none}
        />
      </Fld>
      <ActiveSwitch value={v.isActive} onChange={(isActive) => setV({ ...v, isActive })} />
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Positions

export function PositionsPanel({
  positions,
  departments,
  canWrite,
}: {
  positions: Position[];
  departments: Department[];
  canWrite: boolean;
}) {
  const dialog = (p?: Position) => (
    <FormDialog trigger={p ? editTrigger : <NewButton label={m.newPosition} />} title={p ? p.title : m.newPosition}>
      {(close) => <PositionForm pos={p} departments={departments} close={close} />}
    </FormDialog>
  );
  const columns: ColumnDef<Position, unknown>[] = [
    { accessorKey: "title", header: m.title },
    { accessorKey: "departmentName", header: th.department, cell: ({ getValue }) => getValue<string>() || "–" },
    { accessorKey: "level", header: m.level, cell: ({ getValue }) => m.levels[getValue<string>()] ?? getValue<string>() },
    {
      id: "range",
      header: m.salaryRange,
      cell: ({ row }) =>
        Number(row.original.salaryMax) > 0 ? (
          <span className="whitespace-nowrap">
            <Money value={row.original.salaryMin} /> – <Money value={row.original.salaryMax} />
          </span>
        ) : (
          "–"
        ),
    },
    { accessorKey: "isActive", header: th.status, cell: ({ row }) => <ActiveBadge active={row.original.isActive} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <RowActions canWrite={canWrite} edit={dialog(row.original)} onDelete={() => deletePositionAction(row.original.id)} />
      ),
    },
  ];
  return <ErpDataTable columns={columns} data={positions} toolbar={canWrite ? dialog() : null} />;
}

function PositionForm({ pos, departments, close }: { pos?: Position; departments: Department[]; close: () => void }) {
  const [v, setV] = React.useState({
    title: pos?.title ?? "",
    departmentId: pos?.departmentId ?? "",
    level: pos?.level || "staff",
    salaryMin: Number(pos?.salaryMin) ? (pos?.salaryMin ?? "") : "",
    salaryMax: Number(pos?.salaryMax) ? (pos?.salaryMax ?? "") : "",
    isActive: pos?.isActive ?? true,
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => savePositionAction(pos?.id ?? null, v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <Fld label={m.title} htmlFor="title" error={errors?.title}>
        <Input id="title" value={v.title} onChange={(e) => setV({ ...v, title: e.target.value })} />
      </Fld>
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={th.department}>
          <SimpleSelect
            value={v.departmentId}
            onChange={(departmentId) => setV({ ...v, departmentId })}
            emptyLabel={th.none}
            options={departments.map((d) => ({ value: d.id, label: d.name }))}
          />
        </Fld>
        <Fld label={m.level} error={errors?.level}>
          <SimpleSelect
            value={v.level}
            onChange={(level) => setV({ ...v, level })}
            options={Object.entries(m.levels).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={m.salaryMin} error={errors?.salaryMin}>
          <MoneyInput value={v.salaryMin} onChange={(salaryMin) => setV({ ...v, salaryMin })} />
        </Fld>
        <Fld label={m.salaryMax} error={errors?.salaryMax}>
          <MoneyInput value={v.salaryMax} onChange={(salaryMax) => setV({ ...v, salaryMax })} />
        </Fld>
      </div>
      <ActiveSwitch value={v.isActive} onChange={(isActive) => setV({ ...v, isActive })} />
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Employment statuses

export function StatusesPanel({ statuses, canWrite }: { statuses: EmploymentStatusOption[]; canWrite: boolean }) {
  const dialog = (s?: EmploymentStatusOption) => (
    <FormDialog trigger={s ? editTrigger : <NewButton label={m.newStatus} />} title={s ? s.name : m.newStatus}>
      {(close) => <StatusForm status={s} nextOrder={statuses.length + 1} close={close} />}
    </FormDialog>
  );
  const columns: ColumnDef<EmploymentStatusOption, unknown>[] = [
    { accessorKey: "sortOrder", header: m.sortOrder, meta: { align: "right" }, size: 70 },
    { accessorKey: "code", header: m.code, meta: { mono: true } },
    { accessorKey: "name", header: m.name },
    { accessorKey: "isActive", header: th.status, cell: ({ row }) => <ActiveBadge active={row.original.isActive} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <RowActions
          canWrite={canWrite}
          edit={dialog(row.original)}
          onDelete={() => deleteEmploymentStatusAction(row.original.code)}
        />
      ),
    },
  ];
  return <ErpDataTable columns={columns} data={statuses} toolbar={canWrite ? dialog() : null} />;
}

function StatusForm({
  status,
  nextOrder,
  close,
}: {
  status?: EmploymentStatusOption;
  nextOrder: number;
  close: () => void;
}) {
  const [v, setV] = React.useState({
    code: status?.code ?? "",
    name: status?.name ?? "",
    isActive: status?.isActive ?? true,
    sortOrder: status?.sortOrder ?? nextOrder,
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveEmploymentStatusAction(v), { success: th.saved, onOk: close, onFail });
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={m.code} htmlFor="code" error={errors?.code}>
          <Input
            id="code"
            value={v.code}
            disabled={Boolean(status)}
            onChange={(e) => setV({ ...v, code: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, "_") })}
          />
        </Fld>
        <Fld label={m.sortOrder} htmlFor="order">
          <Input
            id="order"
            type="number"
            min={0}
            value={v.sortOrder}
            onChange={(e) => setV({ ...v, sortOrder: Number(e.target.value) })}
          />
        </Fld>
      </div>
      <Fld label={m.name} htmlFor="name" error={errors?.name}>
        <Input id="name" value={v.name} onChange={(e) => setV({ ...v, name: e.target.value })} />
      </Fld>
      <ActiveSwitch value={v.isActive} onChange={(isActive) => setV({ ...v, isActive })} />
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}
