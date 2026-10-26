"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Check, CircleDashed, Plus, X } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { DateRangeFilter, DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { PEOPLE_PARAMS, PeopleFilters } from "./list-filters";
import { formatDate, formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import { cancelLeaveAction, createLeaveAction, decideLeaveAction } from "../actions";
import { toQuery } from "../format";
import { kinds, th } from "../strings";
import type { Leave, LeaveBalance, LeaveType } from "../types";
import { ConfirmAction, DecisionButtons, EmployeePicker, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors, type EmployeeOption } from "./ui";

const l = th.leave;

/** Two-step approval trail: manager, then HR. */
function LeaveSteps({ leave }: { leave: Leave }) {
  const rejected = leave.status === "rejected";
  const steps = [
    { label: l.managerStep, done: Boolean(leave.managerApprovedAt), by: leave.managerApprovedByName, at: leave.managerApprovedAt },
    { label: l.hrStep, done: Boolean(leave.approvedAt), by: leave.approvedByName, at: leave.approvedAt },
  ];
  return (
    <span className="flex items-center gap-1">
      {steps.map((s, i) => {
        const failed = rejected && !s.done && (i === 0 || steps[0]!.done);
        return (
          <Tooltip key={s.label}>
            <TooltipTrigger asChild>
              <span
                className={cn(
                  "flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px]",
                  s.done && "border-transparent bg-positive/15 text-positive",
                  failed && "border-transparent bg-destructive/10 text-destructive",
                )}
              >
                {s.done ? <Check className="size-3" /> : failed ? <X className="size-3" /> : <CircleDashed className="size-3" />}
                {s.label}
              </span>
            </TooltipTrigger>
            <TooltipContent>{s.done ? `${l.approvedBy(s.by || "–")} · ${formatDateTime(s.at)}` : s.label}</TooltipContent>
          </Tooltip>
        );
      })}
    </span>
  );
}

export function leaveColumns(opts: {
  typeNames: Record<string, string>;
  showEmployee: boolean;
  onCancel: (id: string) => Promise<ActionResult<unknown>>;
}): ColumnDef<Leave, unknown>[] {
  const cols: ColumnDef<Leave, unknown>[] = [
    {
      accessorKey: "leaveType",
      header: l.leaveType,
      cell: ({ getValue }) => opts.typeNames[getValue<string>()] ?? getValue<string>(),
    },
    {
      id: "period",
      header: th.period,
      cell: ({ row }) => (
        <span className="whitespace-nowrap tabular">
          {formatDate(row.original.startDate)}
          {row.original.endDate !== row.original.startDate ? ` – ${formatDate(row.original.endDate)}` : ""}
        </span>
      ),
    },
    {
      accessorKey: "totalDays",
      header: l.duration,
      meta: { align: "right" },
      cell: ({ row }) => (
        <span title={row.original.excludedHolidays.length ? l.excluded(row.original.excludedHolidays.join(", ")) : undefined}>
          {th.days(row.original.totalDays)}
        </span>
      ),
    },
    {
      accessorKey: "reason",
      header: th.reason,
      cell: ({ row }) => (
        <span className="line-clamp-2 max-w-64 text-sm">
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
        <span className="flex flex-col items-start gap-1">
          <ErpStatusBadge status={row.original.status} labels={l.statuses} kinds={kinds.leave} />
          {row.original.status !== "cancelled" ? <LeaveSteps leave={row.original} /> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => {
        const v = row.original;
        return (
          <div className="flex justify-end gap-1.5">
            {v.canDecide ? (
              <DecisionButtons
                onApprove={() => decideLeaveAction(v.id, true)}
                onReject={(reason) => decideLeaveAction(v.id, false, reason)}
              />
            ) : null}
            {v.canCancel ? (
              <ConfirmAction
                title={l.cancelTitle}
                confirmLabel={l.cancel}
                success={l.cancelled}
                action={() => opts.onCancel(v.id)}
                trigger={
                  <Button size="xs" variant="ghost">
                    {l.cancel}
                  </Button>
                }
              />
            ) : null}
          </div>
        );
      },
    },
  ];
  if (opts.showEmployee)
    cols.unshift({
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <Link href={`/erp/hr/employees/${row.original.employee.id}?tab=leave`} className="hover:underline">
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">
            {row.original.employee.employeeNo} · {row.original.employee.departmentName}
          </span>
        </Link>
      ),
    });
  return cols;
}

export function LeaveTable({
  leaves,
  page,
  types,
  employees,
  canWrite,
  filters,
  approvals,
  departments,
}: {
  leaves: Leave[];
  page?: ServerPage;
  types: LeaveType[];
  employees: EmployeeOption[];
  departments: FilterOption[];
  canWrite: boolean;
  filters: Record<string, string | undefined>;
  approvals?: boolean;
}) {
  const typeNames = Object.fromEntries(types.map((t) => [t.code, t.name]));
  const columns = leaveColumns({ typeNames, showEmployee: true, onCancel: cancelLeaveAction });
  return (
    <ErpDataTable
      columns={columns}
      data={leaves}
      pagination={page}
      getRowId={(r) => r.id}
      emptyMessage={approvals ? l.approvalsEmpty : undefined}
      toolbar={
        approvals ? null : (
          <>
            <SelectFilter
              param="status"
              label={th.status}
              allLabel={th.allStatus}
              options={Object.entries(l.statuses).map(([value, label]) => ({ value, label }))}
            />
            <SelectFilter
              param="type"
              label={l.leaveType}
              allLabel={th.allTypes}
              options={types.map((t) => ({ value: t.code, label: t.name }))}
            />
            <DateRangeFilter />
            <PeopleFilters departments={departments} employees={employees} />
            <ResetFilters params={["status", "type", "from", "to", ...PEOPLE_PARAMS]} />
            <div className="ml-auto flex gap-2">
              <DownloadButton
                href={`/api/erp/v1/hr/leaves:export${toQuery({ status: filters.status, leaveType: filters.type, startDate: filters.from, endDate: filters.to })}`}
                fileName="cuti.csv"
              >
                {l.exportCsv}
              </DownloadButton>
              {canWrite ? (
                <FormDialog
                  title={l.newLeaveFor}
                  trigger={
                    <Button size="sm">
                      <Plus className="size-4" /> {l.newLeave}
                    </Button>
                  }
                >
                  {(close) => <LeaveForm types={types} employees={employees} submit={createLeaveAction} close={close} />}
                </FormDialog>
              ) : null}
            </div>
          </>
        )
      }
    />
  );
}

type LeaveInput = { employeeId: string; leaveType: string; startDate: string; endDate: string; reason: string };

/** Leave request form; `employees` adds a picker (HR filing for someone). */
export function LeaveForm({
  types,
  employees,
  balance,
  submit,
  close,
}: {
  types: LeaveType[];
  employees?: EmployeeOption[];
  balance?: LeaveBalance;
  submit: (v: LeaveInput) => Promise<ActionResult<unknown>>;
  close: () => void;
}) {
  const [v, setV] = React.useState<LeaveInput>({
    employeeId: "",
    leaveType: types[0]?.code ?? "annual",
    startDate: "",
    endDate: "",
    reason: "",
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  const usesQuota = types.find((t) => t.code === v.leaveType)?.usesAnnualQuota;
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => submit(v), { success: l.created, onOk: close, onFail });
      }}
    >
      {employees ? (
        <Fld label={th.employee} error={errors?.employeeId}>
          <EmployeePicker value={v.employeeId} onChange={(employeeId) => setV({ ...v, employeeId })} options={employees} />
        </Fld>
      ) : null}
      <Fld
        label={l.leaveType}
        error={errors?.leaveType}
        hint={usesQuota && balance ? `${th.balance.annualRemaining}: ${th.days(balance.annualLeaveRemaining)}` : undefined}
      >
        <SimpleSelect
          value={v.leaveType}
          onChange={(leaveType) => setV({ ...v, leaveType })}
          options={types.map((t) => ({ value: t.code, label: t.name }))}
        />
      </Fld>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={l.startDate} error={errors?.startDate}>
          <DatePicker
            value={v.startDate}
            onChange={(startDate) => setV({ ...v, startDate, endDate: v.endDate && v.endDate >= startDate ? v.endDate : startDate })}
            invalid={Boolean(errors?.startDate)}
          />
        </Fld>
        <Fld label={l.endDate} error={errors?.endDate}>
          <DatePicker value={v.endDate} onChange={(endDate) => setV({ ...v, endDate })} invalid={Boolean(errors?.endDate)} />
        </Fld>
      </div>
      <Fld label={th.reason} htmlFor="reason" error={errors?.reason}>
        <Textarea id="reason" rows={3} value={v.reason} onChange={(e) => setV({ ...v, reason: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={l.newLeave} />
    </form>
  );
}

export function BalancesTable({ rows }: { rows: { employee: EmployeeOption; balance: LeaveBalance }[] }) {
  const b = th.balance;
  const columns: ColumnDef<(typeof rows)[number], unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => r.employee.label,
      header: th.employee,
      cell: ({ row }) => (
        <Link href={`/erp/hr/employees/${row.original.employee.id}?tab=leave`} className="font-medium hover:underline">
          {row.original.employee.label}
        </Link>
      ),
    },
    { id: "total", accessorFn: (r) => r.balance.annualLeaveTotal, header: b.annualTotal, meta: { align: "right" } },
    { id: "used", accessorFn: (r) => r.balance.annualLeaveUsed, header: b.annualUsed, meta: { align: "right" } },
    {
      id: "remaining",
      accessorFn: (r) => r.balance.annualLeaveRemaining,
      header: b.annualRemaining,
      meta: { align: "right" },
      cell: ({ getValue }) => <span className="font-semibold">{getValue<number>()}</span>,
    },
    { id: "sick", accessorFn: (r) => r.balance.sickLeaveUsed, header: b.sick, meta: { align: "right" } },
    { id: "unpaid", accessorFn: (r) => r.balance.unpaidLeaveUsed, header: b.unpaid, meta: { align: "right" } },
    { id: "pending", accessorFn: (r) => r.balance.pendingDays, header: b.pending, meta: { align: "right" } },
  ];
  return <ErpDataTable columns={columns} data={rows} exportName="saldo-cuti" getRowId={(r) => r.employee.id} />;
}
