"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { sumMoney } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, type FilterOption } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import {
  createPayrollAdditionAction,
  deactivateSalaryAction,
  deletePayrollAdditionAction,
  saveSalaryAction,
  updatePayrollSettingsAction,
} from "../actions";
import { monthLabel, PTKP_STATUSES, th } from "../strings";
import type { PayrollAddition, PayrollSettings, Salary } from "../types";
import { PEOPLE_PARAMS, PeopleFilters } from "./list-filters";
import { ConfirmAction, UrlSwitch, EmployeePicker, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors, type EmployeeOption } from "./ui";

const s = th.salary;
const L = th.payroll.lines;

const ALLOWANCES = ["fixedAllowance", "variableAllowance", "transportAllowance", "mealAllowance", "housingAllowance"] as const;

// Salaries

export function SalariesTable({
  salaries,
  employees,
  departments,
  canWrite,
}: {
  salaries: Salary[];
  employees: EmployeeOption[];
  departments: FilterOption[];
  canWrite: boolean;
}) {
  const columns: ColumnDef<Salary, unknown>[] = [
    {
      id: "employee",
      accessorFn: (r) => r.employee.fullName,
      header: th.employee,
      cell: ({ row }) => (
        <span>
          <span className="block font-medium">{row.original.employee.fullName}</span>
          <span className="text-xs text-muted-foreground">{row.original.employee.departmentName}</span>
        </span>
      ),
    },
    { accessorKey: "baseSalary", header: L.baseSalary, meta: { align: "right" }, cell: ({ getValue }) => <Money value={getValue<string>()} /> },
    {
      id: "allowances",
      accessorFn: (r) => sumMoney(ALLOWANCES.map((k) => r[k])),
      header: s.allowances,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<number>()} muteZero />,
    },
    {
      id: "total",
      accessorFn: (r) => sumMoney([r.baseSalary, ...ALLOWANCES.map((k) => r[k])]),
      header: s.total,
      meta: { align: "right" },
      cell: ({ getValue }) => <Money value={getValue<number>()} className="font-semibold" />,
    },
    { accessorKey: "ptkpStatus", header: s.ptkp },
    {
      id: "flags",
      header: "BPJS",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground">
          {[row.original.bpjsTkEnrolled && s.bpjsTk, row.original.bpjsKesEnrolled && s.bpjsKes, row.original.taperaEnrolled && s.tapera]
            .filter(Boolean)
            .join(" · ") || "–"}
        </span>
      ),
    },
    { accessorKey: "effectiveDate", header: s.effectiveDate, cell: ({ getValue }) => <span className="tabular">{formatDate(getValue<string>())}</span> },
    {
      accessorKey: "isActive",
      header: th.status,
      cell: ({ getValue }) => <ErpStatusBadge status={getValue<boolean>() ? "ACTIVE" : "INACTIVE"} />,
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) =>
        canWrite && row.original.isActive ? (
          <div className="flex justify-end gap-1">
            <FormDialog
              title={s.editSalary}
              wide
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={th.edit}>
                  <Pencil />
                </Button>
              }
            >
              {(close) => <SalaryForm salary={row.original} employees={employees} close={close} />}
            </FormDialog>
            <ConfirmAction
              title={s.deactivateTitle}
              confirmLabel={s.deactivate}
              success={s.deactivated}
              action={() => deactivateSalaryAction(row.original.id)}
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={s.deactivate}>
                  <Trash2 />
                </Button>
              }
            />
          </div>
        ) : null,
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={salaries}
      getRowId={(r) => r.id}
      exportName="data-gaji"
      toolbar={
        <>
          <UrlSwitch param="all" label={th.employees.includeInactive} />
          <PeopleFilters departments={departments} employees={employees} />
          <ResetFilters params={PEOPLE_PARAMS} />
          {canWrite ? (
            <div className="ml-auto">
              <FormDialog
                title={s.newSalary}
                description={s.newSalaryBody}
                wide
                trigger={
                  <Button>
                    <Plus className="size-4" /> {s.newSalary}
                  </Button>
                }
              >
                {(close) => <SalaryForm employees={employees} close={close} />}
              </FormDialog>
            </div>
          ) : null}
        </>
      }
    />
  );
}

function SalaryForm({ salary, employees, close }: { salary?: Salary; employees: EmployeeOption[]; close: () => void }) {
  const [v, setV] = React.useState({
    employeeId: salary?.employee.id ?? "",
    baseSalary: salary?.baseSalary ?? "",
    fixedAllowance: salary?.fixedAllowance ?? "0.00",
    variableAllowance: salary?.variableAllowance ?? "0.00",
    transportAllowance: salary?.transportAllowance ?? "0.00",
    mealAllowance: salary?.mealAllowance ?? "0.00",
    housingAllowance: salary?.housingAllowance ?? "0.00",
    ptkpStatus: salary?.ptkpStatus ?? "TK/0",
    isTaxable: salary?.isTaxable ?? true,
    bpjsTkEnrolled: salary?.bpjsTkEnrolled ?? true,
    bpjsKesEnrolled: salary?.bpjsKesEnrolled ?? true,
    taperaEnrolled: salary?.taperaEnrolled ?? false,
    effectiveDate: salary?.effectiveDate ?? "",
    notes: salary?.notes ?? "",
  });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  const total = sumMoney([v.baseSalary, ...ALLOWANCES.map((k) => v[k])]);
  const flags = [
    ["isTaxable", s.taxable],
    ["bpjsTkEnrolled", s.bpjsTk],
    ["bpjsKesEnrolled", s.bpjsKes],
    ["taperaEnrolled", s.tapera],
  ] as const;
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveSalaryAction(salary?.id ?? null, { ...v, baseSalary: v.baseSalary || "0.00" }), {
          success: s.created,
          onOk: close,
          onFail,
        });
      }}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <Fld label={th.employee} error={errors?.employeeId}>
          <EmployeePicker
            value={v.employeeId}
            onChange={(employeeId) => setV({ ...v, employeeId })}
            options={employees}
            invalid={Boolean(errors?.employeeId)}
          />
        </Fld>
        <Fld label={s.effectiveDate} error={errors?.effectiveDate}>
          <DatePicker value={v.effectiveDate} onChange={(effectiveDate) => setV({ ...v, effectiveDate })} />
        </Fld>
        <Fld label={L.baseSalary} error={errors?.baseSalary}>
          <MoneyInput value={v.baseSalary} onChange={(baseSalary) => setV({ ...v, baseSalary })} invalid={Boolean(errors?.baseSalary)} />
        </Fld>
        {ALLOWANCES.map((k) => (
          <Fld key={k} label={L[k]} error={errors?.[k]}>
            <MoneyInput value={v[k]} onChange={(val) => setV({ ...v, [k]: val || "0.00" })} />
          </Fld>
        ))}
        <Fld label={s.ptkp}>
          <SimpleSelect
            value={v.ptkpStatus}
            onChange={(ptkpStatus) => setV({ ...v, ptkpStatus })}
            options={PTKP_STATUSES.map((x) => ({ value: x, label: x }))}
          />
        </Fld>
      </div>
      <div className="flex flex-wrap gap-4">
        {flags.map(([k, label]) => (
          <label key={k} className="flex items-center gap-2 text-sm">
            <Checkbox checked={v[k]} onCheckedChange={(on) => setV({ ...v, [k]: on === true })} />
            {label}
          </label>
        ))}
      </div>
      <p className="rounded-xl bg-muted/60 px-3 py-2 text-sm">
        {s.total}: <Money value={total} className="font-semibold" />
      </p>
      <Fld label={th.notes} htmlFor="notes">
        <Textarea id="notes" rows={2} value={v.notes} onChange={(e) => setV({ ...v, notes: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Additions

export function AdditionsTable({
  additions,
  employees,
  canWrite,
}: {
  additions: PayrollAddition[];
  employees: EmployeeOption[];
  canWrite: boolean;
}) {
  const a = th.additions;
  const columns: ColumnDef<PayrollAddition, unknown>[] = [
    { id: "employee", accessorFn: (r) => r.employee.fullName, header: th.employee },
    { accessorKey: "label", header: a.label },
    { id: "period", header: th.period, cell: ({ row }) => monthLabel(row.original.periodMonth, row.original.periodYear) },
    { accessorKey: "amount", header: a.amount, meta: { align: "right" }, cell: ({ getValue }) => <Money value={getValue<string>()} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row }) =>
        canWrite ? (
          <ConfirmAction
            title={th.confirmDelete}
            confirmLabel={th.delete}
            success={th.deleted}
            action={() => deletePayrollAdditionAction(row.original.id)}
            trigger={
              <Button size="icon-xs" variant="ghost" aria-label={th.delete}>
                <Trash2 />
              </Button>
            }
          />
        ) : null,
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={additions}
      getRowId={(r) => r.id}
      toolbar={
        canWrite ? (
          <div className="ml-auto">
            <FormDialog
              title={a.newAddition}
              description={a.newAdditionBody}
              trigger={
                <Button>
                  <Plus className="size-4" /> {a.newAddition}
                </Button>
              }
            >
              {(close) => <AdditionForm employees={employees} close={close} />}
            </FormDialog>
          </div>
        ) : null
      }
    />
  );
}

function AdditionForm({ employees, close }: { employees: EmployeeOption[]; close: () => void }) {
  const a = th.additions;
  const [v, setV] = React.useState({ employeeId: "", amount: "", label: "", periodMonth: "", periodYear: "" });
  const { errors, onFail } = useFieldErrors();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            createPayrollAdditionAction({
              ...v,
              amount: v.amount || "0.00",
              periodMonth: Number(v.periodMonth) || 0,
              periodYear: Number(v.periodYear) || 0,
            }),
          { success: a.created, onOk: close, onFail },
        );
      }}
    >
      <Fld label={th.employee} error={errors?.employeeId}>
        <EmployeePicker value={v.employeeId} onChange={(employeeId) => setV({ ...v, employeeId })} options={employees} />
      </Fld>
      <Fld label={a.label} htmlFor="label" error={errors?.label}>
        <Input id="label" value={v.label} onChange={(e) => setV({ ...v, label: e.target.value })} />
      </Fld>
      <Fld label={a.amount} error={errors?.amount}>
        <MoneyInput value={v.amount} onChange={(amount) => setV({ ...v, amount })} />
      </Fld>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={th.payroll.month} hint={a.periodHint}>
          <SimpleSelect
            value={v.periodMonth}
            onChange={(periodMonth) => setV({ ...v, periodMonth, periodYear: v.periodYear || String(new Date().getFullYear()) })}
            emptyLabel="–"
            options={th.months.map((m, i) => ({ value: String(i + 1), label: m }))}
          />
        </Fld>
        <Fld label={th.payroll.year} htmlFor="py">
          <Input id="py" type="number" value={v.periodYear} onChange={(e) => setV({ ...v, periodYear: e.target.value })} />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

// Settings

type RateKey = Exclude<
  keyof PayrollSettings,
  "ptkp" | "pph21BracketLimits" | "loanMaxActivePerEmployee" | "payrollDay" | "updatedAt" | "lateDeductionMode" | "pph21Method"
>;

const BPJS_KEYS: RateKey[] = [
  "bpjsTkJhtEmployee",
  "bpjsTkJhtEmployer",
  "bpjsTkJpEmployee",
  "bpjsTkJpEmployer",
  "bpjsTkJkk",
  "bpjsTkJkm",
  "bpjsKesEmployee",
  "bpjsKesEmployer",
  "taperaEmployee",
  "taperaEmployer",
];

export function SettingsForm({ settings, canWrite }: { settings: PayrollSettings; canWrite: boolean }) {
  const t = th.settings;
  const F = t.fields;
  const [v, setV] = React.useState(settings);
  const { pending, run } = useErpAction();
  const rate = (k: RateKey) => (
    <Fld key={k} label={F[k] ?? k} htmlFor={k}>
      <Input
        id={k}
        inputMode="decimal"
        disabled={!canWrite}
        value={v[k]}
        onChange={(e) => setV({ ...v, [k]: e.target.value.replace(",", ".") })}
      />
    </Fld>
  );
  const money = (k: RateKey) => (
    <Fld key={k} label={F[k] ?? k}>
      <MoneyInput value={v[k]} disabled={!canWrite} onChange={(val) => setV({ ...v, [k]: val || "0.00" })} />
    </Fld>
  );
  return (
    <form
      className="grid gap-(--gap) lg:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => updatePayrollSettingsAction(v), { success: t.saved });
      }}
    >
      <Card className="gap-4 lg:col-span-2">
        <CardHeader>
          <CardTitle className="text-[15px]">{t.bpjs}</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          {BPJS_KEYS.map(rate)}
          {money("bpjsKesMaxUpah")}
        </CardContent>
      </Card>
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-[15px]">{t.ptkp}</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          {PTKP_STATUSES.map((k) => (
            <Fld key={k} label={k}>
              <MoneyInput
                value={v.ptkp[k] ?? ""}
                disabled={!canWrite}
                onChange={(val) => setV({ ...v, ptkp: { ...v.ptkp, [k]: val || "0.00" } })}
              />
            </Fld>
          ))}
        </CardContent>
      </Card>
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-[15px]">{t.pph21}</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          {v.pph21BracketLimits.map((b, i) => (
            <Fld key={i} label={t.bracket(i)}>
              <MoneyInput
                value={b}
                disabled={!canWrite}
                onChange={(val) =>
                  setV({ ...v, pph21BracketLimits: v.pph21BracketLimits.map((x, j) => (j === i ? val || "0.00" : x)) })
                }
              />
            </Fld>
          ))}
          <Fld label={F.pph21Method}>
            <SimpleSelect
              value={v.pph21Method}
              onChange={(pph21Method) => setV({ ...v, pph21Method })}
              disabled
              options={Object.entries(t.pph21Methods).map(([value, label]) => ({ value, label }))}
            />
          </Fld>
        </CardContent>
      </Card>
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-[15px]">
            {t.overtime} · {t.late}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          {rate("overtimeMultiplier")}
          {rate("overtimeMultiplierHoliday")}
          {rate("overtimeHourlyDivisor")}
          <Fld label={F.lateDeductionMode}>
            <SimpleSelect
              value={v.lateDeductionMode}
              disabled={!canWrite}
              onChange={(lateDeductionMode) => setV({ ...v, lateDeductionMode })}
              options={Object.entries(t.lateModes).map(([value, label]) => ({ value, label }))}
            />
          </Fld>
          {v.lateDeductionMode !== "off" ? money("lateDeductionAmount") : null}
        </CardContent>
      </Card>
      <Card className="gap-4">
        <CardHeader>
          <CardTitle className="text-[15px]">
            {t.loans} · {t.general}
          </CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4">
          {rate("loanMaxInstallmentPercent")}
          <Fld label={F.loanMaxActivePerEmployee} htmlFor="maxActive">
            <Input
              id="maxActive"
              type="number"
              min={0}
              disabled={!canWrite}
              value={v.loanMaxActivePerEmployee}
              onChange={(e) => setV({ ...v, loanMaxActivePerEmployee: Number(e.target.value) })}
            />
          </Fld>
          <Fld label={F.payrollDay} htmlFor="payDay">
            <Input
              id="payDay"
              type="number"
              min={1}
              max={31}
              disabled={!canWrite}
              value={v.payrollDay}
              onChange={(e) => setV({ ...v, payrollDay: Number(e.target.value) })}
            />
          </Fld>
        </CardContent>
      </Card>
      {canWrite ? (
        <div className="flex justify-end lg:col-span-2">
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {th.save}
          </Button>
        </div>
      ) : null}
    </form>
  );
}
