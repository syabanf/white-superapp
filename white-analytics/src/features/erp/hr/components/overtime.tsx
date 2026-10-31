"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Trash2 } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DatePicker, MonthFilter } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { PEOPLE_PARAMS, PeopleFilters } from "./list-filters";
import { formatDate } from "@/lib/format";
import { createOvertimeAction, decideOvertimeAction, deleteOvertimeAction, updateOvertimeAction } from "../actions";
import { sumHours } from "../format";
import { kinds, th } from "../strings";
import type { Overtime } from "../types";
import {
  ConfirmAction,
  DecisionButtons,
  EmployeePicker,
  Fld,
  FormDialog,
  FormFooter,
  useFieldErrors,
  type EmployeeOption,
} from "./ui";

const o = th.overtime;

export function overtimeColumns(opts: {
  showEmployee: boolean;
  actions: (row: Overtime) => React.ReactNode;
}): ColumnDef<Overtime, unknown>[] {
  const cols: ColumnDef<Overtime, unknown>[] = [
    { accessorKey: "date", header: th.date, cell: ({ getValue }) => <span className="tabular">{formatDate(getValue<string>())}</span> },
    {
      id: "time",
      header: `${o.start}–${o.end}`,
      cell: ({ row }) => (
        <span className="tabular">
          {row.original.startTime}–{row.original.endTime}
        </span>
      ),
    },
    { accessorKey: "hours", header: o.hours, meta: { align: "right" }, cell: ({ getValue }) => th.hours(getValue<number>()) },
    { accessorKey: "source", header: o.source, cell: ({ getValue }) => o.sources[getValue<string>()] ?? getValue<string>() },
    {
      accessorKey: "reason",
      header: th.reason,
      cell: ({ row }) => (
        <span className="line-clamp-2 max-w-64">
          {row.original.reason}
          {row.original.rejectionReason ? (
            <span className="block text-xs text-destructive">{row.original.rejectionReason}</span>
          ) : null}
        </span>
      ),
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ row }) => (
        <span className="flex flex-col items-start gap-0.5">
          <ErpStatusBadge status={row.original.status} labels={o.statuses} kinds={kinds.overtime} />
          {row.original.decidedByName ? (
            <span className="text-xs text-muted-foreground">{row.original.decidedByName}</span>
          ) : null}
        </span>
      ),
    },
    { id: "actions", header: "", cell: ({ row }) => opts.actions(row.original) },
  ];
  if (opts.showEmployee)
    cols.unshift({
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">{row.original.employee.departmentName}</span>
        </span>
      ),
    });
  return cols;
}

/**
 * HR / manager decision on a pending employee request, plus HR edit
 * (pending only) and delete (anything not approved, since approved hours
 * feed payroll).
 */
function OvertimeActions({ row, canDecide, canWrite }: { row: Overtime; canDecide: boolean; canWrite: boolean }) {
  const pending = row.status === "pending";
  return (
    <div className="flex items-center justify-end gap-1">
      {canDecide && pending && row.source === "employee" ? (
        <DecisionButtons
          onApprove={() => decideOvertimeAction(row.id, "approve")}
          onReject={(reason) => decideOvertimeAction(row.id, "reject", reason)}
        />
      ) : null}
      {canWrite && pending ? (
        <FormDialog
          title={o.edit}
          description={`${row.employee.fullName} · ${o.sources[row.source] ?? row.source}`}
          trigger={
            <Button size="icon-xs" variant="ghost" aria-label={o.edit}>
              <Pencil />
            </Button>
          }
        >
          {(close) => (
            <OvertimeForm
              initial={{
                employeeId: row.employee.id,
                date: row.date,
                startTime: row.startTime,
                endTime: row.endTime,
                reason: row.reason,
              }}
              success={o.updated}
              submit={(v) => updateOvertimeAction(row.id, v)}
              close={close}
            />
          )}
        </FormDialog>
      ) : null}
      {canWrite && row.status !== "approved" ? (
        <ConfirmAction
          title={o.deleteTitle}
          confirmLabel={th.delete}
          success={o.deletedOk}
          action={() => deleteOvertimeAction(row.id)}
          trigger={
            <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
              <Trash2 />
            </Button>
          }
        />
      ) : null}
    </div>
  );
}

export function OvertimeTable({
  rows,
  employees,
  canWrite,
  canDecide,
  month,
  approvals,
  departments,
}: {
  rows: Overtime[];
  employees: EmployeeOption[];
  departments: FilterOption[];
  canWrite: boolean;
  canDecide: boolean;
  month: string;
  approvals?: boolean;
}) {
  const columns = overtimeColumns({
    showEmployee: true,
    actions: (r) => (
      <OvertimeActions row={r} canDecide={canDecide || Boolean(approvals)} canWrite={canWrite && !approvals} />
    ),
  });
  return (
    <div className="flex flex-col gap-2">
      <ErpDataTable
        columns={columns}
        data={rows}
        getRowId={(r) => r.id}
        exportName="lembur"
        toolbar={
          approvals ? null : (
            <>
              <MonthFilter fallback={month} />
              <SelectFilter
                param="status"
                label={th.status}
                allLabel={th.allStatus}
                options={Object.entries(o.statuses).map(([value, label]) => ({ value, label }))}
              />
              <PeopleFilters departments={departments} employees={employees} />
              <ResetFilters params={["status", ...PEOPLE_PARAMS]} />
              {canWrite ? (
                <div className="ml-auto">
                  <FormDialog
                    title={o.assign}
                    description={o.assignBody}
                    trigger={
                      <Button>
                        <Plus className="size-4" /> {o.assign}
                      </Button>
                    }
                  >
                    {(close) => <OvertimeForm employees={employees} submit={createOvertimeAction} close={close} />}
                  </FormDialog>
                </div>
              ) : null}
            </>
          )
        }
      />
      <p className="text-right text-xs text-muted-foreground">
        {o.totalHours(sumHours(rows.filter((r) => r.status === "approved").map((r) => r.hours)))}
      </p>
    </div>
  );
}

type OvertimeInput = { employeeId: string; date: string; startTime: string; endTime: string; reason: string };

export function OvertimeForm({
  employees,
  initial = { employeeId: "", date: "", startTime: "18:00", endTime: "20:00", reason: "" },
  success = o.created,
  submit,
  close,
}: {
  employees?: EmployeeOption[];
  initial?: OvertimeInput;
  success?: string;
  submit: (v: OvertimeInput) => Promise<ActionResult<unknown>>;
  close: () => void;
}) {
  const [v, setV] = React.useState(initial);
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => submit(v), { success, onOk: close, onFail });
      }}
    >
      {employees ? (
        <Fld label={th.employee} error={errors?.employeeId}>
          <EmployeePicker value={v.employeeId} onChange={(employeeId) => setV({ ...v, employeeId })} options={employees} />
        </Fld>
      ) : null}
      <Fld label={th.date} error={errors?.date}>
        <DatePicker value={v.date} onChange={(date) => setV({ ...v, date })} invalid={Boolean(errors?.date)} />
      </Fld>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={o.start} htmlFor="start" error={errors?.startTime}>
          <Input id="start" type="time" value={v.startTime} onChange={(e) => setV({ ...v, startTime: e.target.value })} />
        </Fld>
        <Fld label={o.end} htmlFor="end" error={errors?.endTime}>
          <Input id="end" type="time" value={v.endTime} onChange={(e) => setV({ ...v, endTime: e.target.value })} />
        </Fld>
      </div>
      <Fld label={th.reason} htmlFor="reason" error={errors?.reason}>
        <Textarea id="reason" rows={3} value={v.reason} onChange={(e) => setV({ ...v, reason: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}
