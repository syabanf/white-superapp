"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { AlertTriangle, Calculator, CheckCircle2, Plus, Trash2, Wallet } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/lib/toast";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SelectFilter, type FilterOption } from "@/features/erp/shared/url-filters";
import { formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import {
  approvePayrollRunAction,
  calculatePayrollRunAction,
  createPayrollRunAction,
  deletePayrollRunAction,
  markPayrollRunPaidAction,
} from "../actions";
import { kinds, monthLabel, th } from "../strings";
import type { PayrollRun, Payslip } from "../types";
import { PayslipDialog } from "./payslip";
import { PEOPLE_PARAMS, PeopleFilters } from "./list-filters";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, type EmployeeOption } from "./ui";

const p = th.payroll;

export function RunsTable({ runs, year, canWrite }: { runs: PayrollRun[]; year: number; canWrite: boolean }) {
  const router = useRouter();
  const columns: ColumnDef<PayrollRun, unknown>[] = [
    {
      accessorKey: "runName",
      header: p.runName,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.runName}</span>
          <span className="text-xs text-muted-foreground">{monthLabel(row.original.periodMonth, row.original.periodYear)}</span>
        </span>
      ),
    },
    { accessorKey: "totalEmployees", header: p.employees, meta: { align: "right" } },
    { accessorKey: "totalGross", header: p.gross, meta: { align: "right" }, cell: ({ getValue }) => <Money value={getValue<string>()} /> },
    {
      accessorKey: "totalDeductions",
      header: p.deductions,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} />,
    },
    {
      accessorKey: "totalNet",
      header: p.net,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} className="font-semibold" />,
    },
    {
      accessorKey: "status",
      header: th.status,
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5">
          <ErpStatusBadge status={row.original.status} labels={p.statuses} kinds={kinds.payroll} />
          {row.original.needsRecalculation ? <AlertTriangle className="size-3.5 text-amber-500" aria-label={p.needsRecalc} /> : null}
        </span>
      ),
    },
  ];
  const now = new Date().getFullYear();
  return (
    <ErpDataTable
      columns={columns}
      data={runs}
      getRowId={(r) => r.id}
      emptyMessage={p.noRuns}
      onRowClick={(row) => router.push(`/erp/hr/payroll/runs/${row.original.id}`)}
      toolbar={
        <>
          <SelectFilter
            param="year"
            label={p.year}
            allLabel={String(now)}
            options={[now - 2, now - 1, now + 1].map((y) => ({ value: String(y), label: String(y) }))}
          />
          {canWrite ? (
            <div className="ml-auto">
              <FormDialog
                title={p.newRun}
                trigger={
                  <Button size="sm">
                    <Plus className="size-4" /> {p.newRun}
                  </Button>
                }
              >
                {(close) => <NewRunForm defaultYear={year} close={close} />}
              </FormDialog>
            </div>
          ) : null}
        </>
      }
    />
  );
}

function NewRunForm({ defaultYear, close }: { defaultYear: number; close: () => void }) {
  const router = useRouter();
  const now = new Date();
  const [month, setMonth] = React.useState(String(now.getMonth() + 1));
  const [year, setYear] = React.useState(String(defaultYear));
  const [name, setName] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createPayrollRunAction(Number(month), Number(year), name), {
          success: p.runCreated,
          onOk: (r) => {
            close();
            router.push(`/erp/hr/payroll/runs/${r.id}`);
          },
        });
      }}
    >
      <div className="grid grid-cols-2 gap-4">
        <Fld label={p.month}>
          <SimpleSelect value={month} onChange={setMonth} options={th.months.map((m, i) => ({ value: String(i + 1), label: m }))} />
        </Fld>
        <Fld label={p.year} htmlFor="year">
          <Input id="year" type="number" value={year} onChange={(e) => setYear(e.target.value)} />
        </Fld>
      </div>
      <Fld label={p.runName} htmlFor="name" hint={p.runNameHint}>
        <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={p.newRun} />
    </form>
  );
}

const FLOW = ["draft", "calculated", "approved", "paid"];

/** Stepper plus the action that moves a run forward. */
export function RunActions({ run: r, canWrite }: { run: PayrollRun; canWrite: boolean }) {
  const router = useRouter();
  const { pending, run } = useErpAction();
  const [thr, setThr] = React.useState(false);
  const step = FLOW.indexOf(r.status);
  return (
    <div className="flex flex-col gap-3">
      <ol className="flex flex-wrap items-center gap-2 text-xs">
        {p.steps.map((label, i) => (
          <li
            key={label}
            className={cn(
              "flex items-center gap-1.5 rounded-full border px-3 py-1",
              i <= step && "border-transparent bg-brand/15 font-medium text-foreground",
              i > step && "text-muted-foreground",
            )}
          >
            {i < step ? <CheckCircle2 className="size-3.5" /> : <span className="tabular">{i + 1}</span>}
            {label}
          </li>
        ))}
      </ol>
      {r.needsRecalculation ? (
        <Alert className="rounded-xl">
          <AlertTriangle className="size-4" />
          <AlertDescription>{p.needsRecalc}</AlertDescription>
        </Alert>
      ) : null}
      {canWrite ? (
        <div className="flex flex-wrap items-center gap-2">
          {r.status === "draft" || r.status === "calculated" ? (
            <>
              <div className="flex items-center gap-2 pr-2">
                <Switch id="thr" checked={thr} onCheckedChange={setThr} />
                <Label htmlFor="thr" className="text-sm">
                  {p.includeThr}
                </Label>
              </div>
              <Button
                size="sm"
                variant={r.status === "draft" ? "default" : "outline"}
                disabled={pending}
                onClick={() =>
                  void run(() => calculatePayrollRunAction(r.id, thr), {
                    onOk: (res) => {
                      toast.success(res.message || p.calculated, {
                        description: res.skipped.length
                          ? `${p.skipped(res.skipped.length)}: ${res.skipped.map((s) => `${s.fullName} (${s.reason})`).join(", ")}`
                          : undefined,
                      });
                    },
                  })
                }
              >
                <Calculator className="size-4" /> {r.status === "draft" ? p.calculate : p.recalculate}
              </Button>
            </>
          ) : null}
          {r.status === "calculated" ? (
            <ConfirmAction
              title={p.approveRun}
              description={p.approveBody}
              confirmLabel={p.approveRun}
              success={p.approvedOk}
              destructive={false}
              action={() => approvePayrollRunAction(r.id)}
              trigger={
                <Button size="sm" disabled={r.needsRecalculation}>
                  <CheckCircle2 className="size-4" /> {p.approveRun}
                </Button>
              }
            />
          ) : null}
          {r.status === "approved" ? (
            <FormDialog
              title={p.markPaid}
              description={p.markPaidBody}
              trigger={
                <Button size="sm">
                  <Wallet className="size-4" /> {p.markPaid}
                </Button>
              }
            >
              {(close) => <MarkPaidForm runId={r.id} close={close} />}
            </FormDialog>
          ) : null}
          {r.status === "draft" || r.status === "calculated" ? (
            <ConfirmAction
              title={th.confirmDelete}
              confirmLabel={th.delete}
              success={th.deleted}
              action={async () => {
                const res = await deletePayrollRunAction(r.id);
                if (res.ok) router.push("/erp/hr/payroll");
                return res;
              }}
              trigger={
                <Button size="sm" variant="ghost" className="text-destructive">
                  <Trash2 className="size-4" />
                </Button>
              }
            />
          ) : null}
        </div>
      ) : null}
      <dl className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground">
        {r.calculatedAt ? <div>{p.calculatedAt}: {formatDateTime(r.calculatedAt)}</div> : null}
        {r.approvedAt ? (
          <div>
            {p.approvedAt}: {formatDateTime(r.approvedAt)}
            {r.approvedByName ? ` · ${r.approvedByName}` : ""}
          </div>
        ) : null}
        {r.paidAt ? <div>{p.paidAt}: {formatDateTime(r.paidAt)}</div> : null}
        {r.notes ? <div>{r.notes}</div> : null}
      </dl>
    </div>
  );
}

function MarkPaidForm({ runId, close }: { runId: string; close: () => void }) {
  const [notes, setNotes] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => markPayrollRunPaidAction(runId, notes), {
          onOk: (res) => {
            toast.success(res.message || p.paidOk);
            close();
          },
        });
      }}
    >
      <Fld label={th.notes} htmlFor="notes">
        <Textarea id="notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={p.markPaid} />
    </form>
  );
}

export function PayslipsTable({
  payslips,
  departments,
  employees,
}: {
  payslips: Payslip[];
  departments: FilterOption[];
  employees: EmployeeOption[];
}) {
  const columns: ColumnDef<Payslip, unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => `${r.employee.fullName} ${r.employee.employeeNo}`,
      header: th.employee,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">
            {row.original.employee.employeeNo} · {row.original.employee.departmentName}
          </span>
        </span>
      ),
    },
    { accessorKey: "grossSalary", header: p.gross, meta: { align: "right" }, cell: ({ getValue }) => <Money value={getValue<string>()} /> },
    {
      id: "bpjs",
      accessorFn: (r) =>
        Number(r.bpjsTkJhtDeduction) + Number(r.bpjsTkJpDeduction) + Number(r.bpjsKesDeduction) + Number(r.taperaDeduction),
      header: p.bpjsEmployee,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<number>()} muteZero />,
    },
    { accessorKey: "pph21Deduction", header: p.pph21, meta: { align: "right" }, cell: ({ getValue }) => <Money value={getValue<string>()} muteZero /> },
    {
      accessorKey: "totalDeductions",
      header: p.deductions,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} />,
    },
    {
      accessorKey: "netSalary",
      header: p.net,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<string>()} className="font-semibold" />,
    },
    { id: "slip", header: "", cell: ({ row }) => <PayslipDialog slip={row.original} /> },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={payslips}
      getRowId={(r) => r.id}
      emptyMessage={p.noPayslips}
      toolbar={
        <>
          <PeopleFilters departments={departments} employees={employees} />
          <ResetFilters params={PEOPLE_PARAMS} />
        </>
      }
      exportName="payroll"
    />
  );
}
