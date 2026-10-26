"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { FileText, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { PhotoButton } from "@/features/erp/hr/components/attendance";
import { leaveColumns, LeaveForm } from "@/features/erp/hr/components/leave";
import { LoanForm, loanColumns } from "@/features/erp/hr/components/loans";
import { overtimeColumns, OvertimeForm } from "@/features/erp/hr/components/overtime";
import { PayslipDialog } from "@/features/erp/hr/components/payslip";
import { ConfirmAction, DecisionButtons, FormDialog } from "@/features/erp/hr/components/ui";
import { timeWib } from "@/features/erp/hr/format";
import { kinds, monthLabel, th } from "@/features/erp/hr/strings";
import type { AttendanceRecord, Leave, LeaveBalance, LeaveType, Loan, Overtime, Payslip } from "@/features/erp/hr/types";
import { formatDate } from "@/lib/format";
import {
  cancelMyLeaveAction,
  createMyLeaveAction,
  createMyLoanAction,
  createMyOvertimeAction,
  decideMyOvertimeAction,
} from "../actions";
import { tm } from "../strings";

const a = th.attendance;

export function MyAttendanceTable({ records }: { records: AttendanceRecord[] }) {
  const columns: ColumnDef<AttendanceRecord, unknown>[] = [
    { accessorKey: "date", header: th.date, cell: ({ getValue }) => <span className="tabular">{formatDate(getValue<string>())}</span> },
    { accessorKey: "shiftName", header: a.shift, cell: ({ getValue }) => getValue<string>() || "–" },
    {
      id: "in",
      header: a.clockIn,
      cell: ({ row }) => (
        <span className="flex items-center justify-end gap-1 tabular sm:justify-start">
          {timeWib(row.original.clockIn)}
          {row.original.hasClockInPhoto ? (
            <PhotoButton
              attendanceId={row.original.id}
              kind="in"
              name={formatDate(row.original.date)}
              time={timeWib(row.original.clockIn)}
            />
          ) : null}
        </span>
      ),
    },
    { id: "out", header: a.clockOut, cell: ({ row }) => <span className="tabular">{timeWib(row.original.clockOut)}</span> },
    {
      accessorKey: "workHours",
      header: a.workHours,
      meta: { align: "right" },
      cell: ({ getValue }) => (getValue<number>() ? th.hours(getValue<number>()) : "–"),
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ row }) => (
        <span className="flex items-center justify-end gap-1.5 sm:justify-start">
          <ErpStatusBadge status={row.original.status} labels={a.statuses} kinds={kinds.attendance} />
          {row.original.lateMinutes ? (
            <span className="text-xs text-muted-foreground">{th.minutes(row.original.lateMinutes)}</span>
          ) : null}
        </span>
      ),
    },
  ];
  return <ErpDataTable columns={columns} data={records} getRowId={(r) => r.id} />;
}

export function MyLeaveTable({ leaves, types, balance }: { leaves: Leave[]; types: LeaveType[]; balance: LeaveBalance }) {
  const typeNames = Object.fromEntries(types.map((t) => [t.code, t.name]));
  return (
    <ErpDataTable
      columns={leaveColumns({ typeNames, showEmployee: false, onCancel: cancelMyLeaveAction })}
      data={leaves}
      getRowId={(r) => r.id}
      toolbar={
        <div className="ml-auto">
          <FormDialog
            title={th.leave.newLeave}
            trigger={
              <Button size="sm">
                <Plus className="size-4" /> {th.leave.newLeave}
              </Button>
            }
          >
            {(close) => <LeaveForm types={types} balance={balance} submit={createMyLeaveAction} close={close} />}
          </FormDialog>
        </div>
      }
    />
  );
}

function MyOvertimeActions({ row }: { row: Overtime }) {
  if (row.status !== "pending") return null;
  if (row.source === "company")
    return (
      <DecisionButtons
        onApprove={() => decideMyOvertimeAction(row.id, "approve")}
        onReject={(reason) => decideMyOvertimeAction(row.id, "reject", reason)}
      />
    );
  return (
    <ConfirmAction
      title={tm.overtime.cancelTitle}
      confirmLabel={tm.overtime.cancel}
      success={th.overtime.decided}
      action={() => decideMyOvertimeAction(row.id, "cancel")}
      trigger={
        <Button size="xs" variant="ghost">
          {tm.overtime.cancel}
        </Button>
      }
    />
  );
}

export function MyOvertimeTable({ rows }: { rows: Overtime[] }) {
  return (
    <ErpDataTable
      columns={overtimeColumns({ showEmployee: false, actions: (r) => <MyOvertimeActions row={r} /> })}
      data={rows}
      getRowId={(r) => r.id}
      toolbar={
        <div className="ml-auto">
          <FormDialog
            title={tm.overtime.request}
            trigger={
              <Button size="sm">
                <Plus className="size-4" /> {tm.overtime.request}
              </Button>
            }
          >
            {(close) => <OvertimeForm submit={createMyOvertimeAction} close={close} />}
          </FormDialog>
        </div>
      }
    />
  );
}

export function MyLoansTable({ loans }: { loans: Loan[] }) {
  return (
    <ErpDataTable
      columns={loanColumns({ showEmployee: false, canDecide: false })}
      data={loans}
      getRowId={(r) => r.id}
      toolbar={
        <div className="ml-auto">
          <FormDialog
            title={tm.loans.request}
            trigger={
              <Button size="sm">
                <Plus className="size-4" /> {tm.loans.request}
              </Button>
            }
          >
            {(close) => <LoanForm submit={createMyLoanAction} close={close} />}
          </FormDialog>
        </div>
      }
    />
  );
}

/** Payslip cards; each opens the breakdown with a PDF download. */
export function MyPayslips({ payslips }: { payslips: Payslip[] }) {
  if (!payslips.length) return <ErpEmpty icon={<FileText className="size-5" />} title={tm.payslips.empty} />;
  return (
    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {payslips.map((s) => (
        <li key={s.id}>
          <Card className="gap-3 p-4">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="font-semibold">{monthLabel(s.periodMonth, s.periodYear)}</p>
                <p className="text-xs text-muted-foreground">
                  {th.payroll.gross} <Money value={s.grossSalary} />
                </p>
              </div>
              <PayslipDialog slip={s} pdfBase="/api/erp/v1/hr/me/payslips" />
            </div>
            <div>
              <p className="text-xs text-muted-foreground">{th.payroll.net}</p>
              <Money value={s.netSalary} className="text-xl font-bold" />
            </div>
          </Card>
        </li>
      ))}
    </ul>
  );
}
