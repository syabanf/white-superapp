"use client";

import { FileText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import { formatDecimal } from "@/lib/format";
import { monthLabel, th } from "../strings";
import type { Payslip } from "../types";
import { FormDialog } from "./ui";

const p = th.payroll;
const L = p.lines;

const EARNINGS = [
  "baseSalary",
  "fixedAllowance",
  "variableAllowance",
  "transportAllowance",
  "mealAllowance",
  "housingAllowance",
  "overtimePay",
  "thr",
  "bonus",
] as const;
const DEDUCTIONS = [
  "bpjsTkJhtDeduction",
  "bpjsTkJpDeduction",
  "bpjsKesDeduction",
  "taperaDeduction",
  "pph21Deduction",
  "unpaidLeaveDeduction",
  "lateDeduction",
  "loanDeduction",
] as const;
const EMPLOYER = [
  "bpjsTkJhtEmployer",
  "bpjsTkJpEmployer",
  "bpjsTkJkkEmployer",
  "bpjsTkJkmEmployer",
  "bpjsKesEmployer",
  "taperaEmployer",
] as const;

function Lines({
  title,
  slip,
  keys,
  total,
  totalLabel,
}: {
  title: string;
  slip: Payslip;
  keys: readonly (keyof Payslip)[];
  total?: string;
  totalLabel?: string;
}) {
  const rows = keys.filter((k) => parseMoney(slip[k] as string) !== 0);
  return (
    <section className="flex flex-col gap-1.5">
      <h4 className="text-xs font-medium text-muted-foreground">{title}</h4>
      {rows.map((k) => (
        <div key={k} className="flex justify-between gap-3 text-sm">
          <span>{L[k]}</span>
          <Money value={slip[k] as string} />
        </div>
      ))}
      {total ? (
        <div className="flex justify-between gap-3 border-t pt-1.5 text-sm font-semibold">
          <span>{totalLabel}</span>
          <Money value={total} />
        </div>
      ) : null}
    </section>
  );
}

/** Full payslip breakdown (NüHabit payslip page and ESS slip detail). */
function PayslipBreakdown({ slip }: { slip: Payslip }) {
  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-2xl bg-brand/10 px-4 py-3">
        <p className="text-xs text-muted-foreground">{p.net}</p>
        <Money value={slip.netSalary} className="text-2xl font-bold" />
      </div>
      <Lines title={p.earnings} slip={slip} keys={EARNINGS} total={slip.grossSalary} totalLabel={p.gross} />
      <Lines title={p.deductionsTitle} slip={slip} keys={DEDUCTIONS} total={slip.totalDeductions} totalLabel={p.deductions} />
      <Separator />
      <Lines title={p.employerTitle} slip={slip} keys={EMPLOYER} />
      <section className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-3">
        {(["workingDays", "presentDays", "lateDays", "unpaidLeaveDays", "overtimeHours", "prorateFactor"] as const).map((k) => (
          <div key={k} className="rounded-lg border px-3 py-2">
            <p className="text-xs text-muted-foreground">{L[k]}</p>
            <p className="font-medium tabular">{formatDecimal(slip[k], k === "prorateFactor" ? 2 : 1)}</p>
          </div>
        ))}
      </section>
      <Lines title={p.taxTitle} slip={slip} keys={["taxableIncome", "ptkpAmount", "pph21Annual"]} />
      {slip.loanDetails.length ? (
        <section className="flex flex-col gap-1.5">
          <h4 className="text-xs font-medium text-muted-foreground">{p.loanTitle}</h4>
          {slip.loanDetails.map((d) => (
            <div key={`${d.loanId}-${d.installmentNo}`} className="flex justify-between gap-3 text-sm">
              <span>{d.label}</span>
              <Money value={d.amount} />
            </div>
          ))}
        </section>
      ) : null}
    </div>
  );
}

/** Opens the breakdown in a dialog with a PDF download. `pdfBase` differs for ESS. */
export function PayslipDialog({ slip, pdfBase = "/api/erp/v1/hr/payslips" }: { slip: Payslip; pdfBase?: string }) {
  return (
    <FormDialog
      title={`${p.payslip} ${monthLabel(slip.periodMonth, slip.periodYear)}`}
      description={`${slip.employee.fullName} · ${slip.employee.employeeNo}`}
      trigger={
        <Button size="xs" variant="outline">
          <FileText className="size-3.5" /> {p.payslip}
        </Button>
      }
    >
      {() => (
        <div className="flex flex-col gap-4">
          <PayslipBreakdown slip={slip} />
          <DownloadButton
            href={`${pdfBase}/${slip.id}/pdf`}
            fileName={`slip-gaji-${slip.periodYear}-${String(slip.periodMonth).padStart(2, "0")}.pdf`}
            size="default"
          >
            {p.pdf}
          </DownloadButton>
        </div>
      )}
    </FormDialog>
  );
}
