"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Eye, Loader2, Plus } from "lucide-react";
import type { ActionResult } from "@/lib/action-result";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { erpClientFetch, useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { cancelLoanAction, createLoanAction, decideLoanAction } from "../actions";
import { kinds, monthLabel, th } from "../strings";
import type { Loan } from "../types";
import {
  ConfirmAction,
  DecisionButtons,
  EmployeePicker,
  Fld,
  FormDialog,
  FormFooter,
  SimpleSelect,
  useFieldErrors,
  type EmployeeOption,
} from "./ui";

const n = th.loans;

export function loanColumns(opts: {
  showEmployee: boolean;
  canDecide: boolean;
  /** Cancels a pending loan; approved loans are repaid through payroll. */
  onCancel?: (id: string) => Promise<ActionResult<unknown>>;
}): ColumnDef<Loan, unknown>[] {
  const { onCancel } = opts;
  const cols: ColumnDef<Loan, unknown>[] = [
    {
      accessorKey: "loanType",
      header: n.loanType,
      cell: ({ row }) => (
        <span>
          <span className="block">{n.types[row.original.loanType] ?? row.original.loanType}</span>
          <span className="line-clamp-1 max-w-56 text-xs text-muted-foreground">{row.original.purpose}</span>
        </span>
      ),
    },
    {
      accessorKey: "principalAmount",
      header: n.principal,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} />,
    },
    {
      id: "tenor",
      header: n.installment,
      meta: { align: "right" },
      cell: ({ row }) => (
        <span className="whitespace-nowrap">
          <Money value={row.original.monthlyInstallment} /> × {row.original.tenorMonths}
        </span>
      ),
    },
    {
      accessorKey: "remainingBalance",
      header: n.remaining,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} muteZero />,
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ getValue }) => <ErpStatusBadge status={getValue<string>()} labels={n.statuses} kinds={kinds.loan} />,
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <div className="flex justify-end gap-1.5">
          {opts.canDecide && row.original.status === "pending" ? (
            <DecisionButtons
              onApprove={() => decideLoanAction(row.original.id, true)}
              onReject={(reason) => decideLoanAction(row.original.id, false, reason)}
            />
          ) : null}
          {onCancel && row.original.status === "pending" ? (
            <ConfirmAction
              title={n.cancelTitle}
              description={n.cancelBody}
              confirmLabel={n.cancel}
              success={n.cancelled}
              action={() => onCancel(row.original.id)}
              trigger={
                <Button size="xs" variant="ghost">
                  {n.cancel}
                </Button>
              }
            />
          ) : null}
          <LoanDetailDialog loan={row.original} />
        </div>
      ),
    },
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

export function LoansTable({
  loans,
  employees,
  canWrite,
  canDecide,
}: {
  loans: Loan[];
  employees: EmployeeOption[];
  canWrite: boolean;
  canDecide: boolean;
}) {
  return (
    <ErpDataTable
      columns={loanColumns({
        showEmployee: true,
        canDecide,
        onCancel: canWrite || canDecide ? cancelLoanAction : undefined,
      })}
      data={loans}
      getRowId={(r) => r.id}
      toolbar={
        <>
          <SelectFilter
            param="status"
            label={th.status}
            allLabel={th.allStatus}
            options={Object.entries(n.statuses).map(([value, label]) => ({ value, label }))}
          />
          {canWrite ? (
            <div className="ml-auto">
              <FormDialog
                title={n.newLoan}
                trigger={
                  <Button>
                    <Plus className="size-4" /> {n.newLoan}
                  </Button>
                }
              >
                {(close) => <LoanForm employees={employees} withInterest submit={createLoanAction} close={close} />}
              </FormDialog>
            </div>
          ) : null}
        </>
      }
    />
  );
}

type LoanInput = {
  employeeId: string;
  loanType: "kasbon" | "loan" | "emergency";
  principalAmount: string;
  interestRate: string;
  tenorMonths: number;
  purpose: string;
};

export function LoanForm({
  employees,
  withInterest,
  submit,
  close,
}: {
  employees?: EmployeeOption[];
  withInterest?: boolean;
  submit: (v: LoanInput) => Promise<ActionResult<unknown>>;
  close: () => void;
}) {
  const [v, setV] = React.useState<LoanInput>({
    employeeId: "",
    loanType: "kasbon",
    principalAmount: "",
    interestRate: "",
    tenorMonths: 3,
    purpose: "",
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  const rate = withInterest ? parseMoney(v.interestRate) : 0;
  const principal = parseMoney(v.principalAmount);
  const installment = v.tenorMonths > 0 ? principal / v.tenorMonths + (principal * rate) / 100 : 0;
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => submit(v), { success: n.created, onOk: close, onFail });
      }}
    >
      {employees ? (
        <Fld label={th.employee} error={errors?.employeeId}>
          <EmployeePicker value={v.employeeId} onChange={(employeeId) => setV({ ...v, employeeId })} options={employees} />
        </Fld>
      ) : null}
      <div className="grid grid-cols-2 gap-4">
        <Fld label={n.loanType}>
          <SimpleSelect
            value={v.loanType}
            onChange={(t) => setV({ ...v, loanType: t as LoanInput["loanType"] })}
            options={Object.entries(n.types).map(([value, label]) => ({ value, label }))}
          />
        </Fld>
        <Fld label={n.tenor} htmlFor="tenor" error={errors?.tenorMonths}>
          <Input
            id="tenor"
            type="number"
            min={1}
            max={24}
            value={v.tenorMonths}
            onChange={(e) => setV({ ...v, tenorMonths: Number(e.target.value) })}
          />
        </Fld>
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={n.principal} error={errors?.principalAmount} className={withInterest ? "" : "col-span-2"}>
          <MoneyInput value={v.principalAmount} onChange={(principalAmount) => setV({ ...v, principalAmount })} />
        </Fld>
        {withInterest ? (
          <Fld label={n.interest} htmlFor="rate" error={errors?.interestRate}>
            <Input
              id="rate"
              inputMode="decimal"
              value={v.interestRate}
              onChange={(e) => setV({ ...v, interestRate: e.target.value.replace(",", ".") })}
            />
          </Fld>
        ) : null}
      </div>
      {principal > 0 ? (
        <p className="rounded-xl bg-muted/60 px-3 py-2 text-sm">
          {n.installment}: <span className="font-semibold">{formatRupiah(installment)}</span>
        </p>
      ) : null}
      <Fld label={n.purpose} htmlFor="purpose" error={errors?.purpose}>
        <Textarea id="purpose" rows={2} value={v.purpose} onChange={(e) => setV({ ...v, purpose: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={n.newLoan} />
    </form>
  );
}

/** Loan details with its installment schedule, loaded when opened. */
function LoanDetailDialog({ loan }: { loan: Loan }) {
  return (
    <FormDialog
      title={`${n.types[loan.loanType] ?? loan.loanType} · ${loan.employee.fullName}`}
      description={loan.purpose}
      trigger={
        <Button size="icon-xs" variant="ghost" aria-label={n.schedule}>
          <Eye />
        </Button>
      }
    >
      {() => <LoanDetail loan={loan} />}
    </FormDialog>
  );
}

function LoanDetail({ loan: initial }: { loan: Loan }) {
  const [loan, setLoan] = React.useState(initial);
  const [loading, setLoading] = React.useState(initial.schedule.length === 0 && initial.status !== "pending");
  React.useEffect(() => {
    if (!loading) return;
    erpClientFetch<Loan>(`/v1/hr/loans/${initial.id}`)
      .then(setLoan)
      .catch(() => undefined)
      .finally(() => setLoading(false));
  }, [initial.id, loading]);
  const facts: [string, React.ReactNode][] = [
    [n.principal, <Money key="p" value={loan.principalAmount} />],
    [n.installment, <Money key="i" value={loan.monthlyInstallment} />],
    [n.paid, <Money key="pd" value={loan.paidAmount} />],
    [n.remaining, <Money key="r" value={loan.remainingBalance} />],
    [n.tenor, loan.tenorMonths],
    [
      n.firstInstallment,
      loan.firstInstallmentMonth ? monthLabel(loan.firstInstallmentMonth, loan.firstInstallmentYear) : "–",
    ],
  ];
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <ErpStatusBadge status={loan.status} labels={n.statuses} kinds={kinds.loan} />
        {loan.approvedByName ? (
          <span className="text-xs text-muted-foreground">
            {loan.approvedByName} · {formatDate(loan.approvedAt)}
          </span>
        ) : null}
      </div>
      {loan.rejectionReason ? <p className="text-sm text-destructive">{loan.rejectionReason}</p> : null}
      <dl className="grid grid-cols-2 gap-3 text-sm">
        {facts.map(([label, value]) => (
          <div key={label} className="rounded-lg border px-3 py-2">
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="font-medium tabular">{value}</dd>
          </div>
        ))}
      </dl>
      <div>
        <h4 className="mb-2 text-xs font-medium text-muted-foreground">{n.schedule}</h4>
        {loading ? (
          <Loader2 className="size-4 animate-spin text-muted-foreground" />
        ) : loan.schedule.length === 0 ? (
          <p className="text-sm text-muted-foreground">{n.scheduleEmpty}</p>
        ) : (
          <ul className="flex flex-col divide-y rounded-lg border">
            {loan.schedule.map((s) => (
              <li key={s.installmentNo} className="flex items-center gap-3 px-3 py-2 text-sm">
                <span className="w-8 text-muted-foreground tabular">
                  {n.installmentNo}
                  {s.installmentNo}
                </span>
                <span className="flex-1">{monthLabel(s.periodMonth, s.periodYear)}</span>
                <Money value={s.amount} />
                <Badge variant={s.isPaid ? "default" : "outline"} className="w-14 justify-center text-[10px]">
                  {s.isPaid ? n.paidBadge : n.unpaidBadge}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
