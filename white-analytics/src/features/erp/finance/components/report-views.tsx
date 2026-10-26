import Link from "next/link";
import { ArrowRight, BookOpen, FileBarChart, Landmark, Scale, Users, Wallet } from "lucide-react";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/format";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import type { AccountBalance, BalanceSheet, CashFlow, CashFlowSection, IncomeStatement, SubsidiaryLedger, TrialBalance } from "../types";
import { fs } from "../strings";

const r = fs.reports;

export const REPORTS = [
  { key: "trial-balance", label: r.trialBalance, icon: Landmark },
  { key: "balance-sheet", label: r.balanceSheet, icon: Scale },
  { key: "income-statement", label: r.incomeStatement, icon: FileBarChart },
  { key: "cash-flow", label: r.cashFlow, icon: Wallet },
  { key: "general-ledger", label: r.generalLedger, icon: BookOpen },
  { key: "subsidiary-ledger", label: r.subsidiaryLedger, icon: Users },
] as const;

export type ReportKey = (typeof REPORTS)[number]["key"];

export function ReportHub() {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {REPORTS.map((rep) => (
        <Link
          key={rep.key}
          href={`/erp/finance/reports?r=${rep.key}`}
          className="group lift flex items-start gap-3 rounded-2xl bg-card p-5 shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          <span className="flex size-10 items-center justify-center rounded-xl bg-brand/10 text-brand">
            <rep.icon className="size-5" />
          </span>
          <span className="min-w-0 flex-1">
            <span className="block font-semibold">{rep.label}</span>
            <span className="block text-sm text-muted-foreground">{r.descriptions[rep.key]}</span>
          </span>
          <ArrowRight className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
        </Link>
      ))}
    </div>
  );
}

/** Paper-like container used by every statement so they print cleanly. */
export function ReportPaper({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <Card className="gap-0 p-0 print:border-0 print:shadow-none">
      <div className="border-b px-6 py-4 text-center">
        <h2 className="text-lg font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{subtitle}</p>
      </div>
      <div className="overflow-x-auto px-2 py-3 sm:px-6">{children}</div>
    </Card>
  );
}

function Row({
  label,
  code,
  value,
  level = 0,
  strong,
  border,
}: {
  label: string;
  code?: string;
  value: string;
  level?: number;
  strong?: boolean;
  border?: boolean;
}) {
  return (
    <tr className={cn(border && "border-t", strong && "font-semibold")}>
      <td className="py-1.5 pr-3" style={{ paddingLeft: level * 16 }}>
        {code ? <span className="mr-2 font-mono text-xs text-muted-foreground">{code}</span> : null}
        {label}
      </td>
      <td className="py-1.5 text-right">
        <Money value={value} signed />
      </td>
    </tr>
  );
}

function Section({ title, rows, total, totalLabel }: { title: string; rows: AccountBalance[]; total: string; totalLabel: string }) {
  return (
    <>
      <tr>
        <td colSpan={2} className="pt-4 pb-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
          {title}
        </td>
      </tr>
      {rows.length === 0 ? (
        <tr>
          <td colSpan={2} className="py-1 pl-4 text-xs text-muted-foreground">
            {r.empty}
          </td>
        </tr>
      ) : (
        rows.map((a) => <Row key={a.id} label={a.name} code={a.codeDisplay || a.code} value={a.balance} level={1} />)
      )}
      <Row label={totalLabel} value={total} strong border />
    </>
  );
}

export function TrialBalanceView({ data }: { data: TrialBalance }) {
  const balanced = Math.abs(parseMoney(data.totalDebit) - parseMoney(data.totalCredit)) < 0.01;
  return (
    <ReportPaper title={r.trialBalance} subtitle={`${fs.asOf} ${formatDate(data.asOf)}`}>
      <table className="w-full min-w-[560px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="py-2 text-left font-medium">{fs.account}</th>
            <th className="py-2 text-right font-medium">{fs.debit}</th>
            <th className="py-2 text-right font-medium">{fs.credit}</th>
          </tr>
        </thead>
        <tbody>
          {data.rows.map((a) => (
            <tr key={a.id} className="border-b border-border/50">
              <td className="py-1.5 pr-3">
                <span className="mr-2 font-mono text-xs text-muted-foreground">{a.codeDisplay || a.code}</span>
                {a.name}
              </td>
              <td className="py-1.5 text-right">
                <Money value={a.debit} muteZero />
              </td>
              <td className="py-1.5 text-right">
                <Money value={a.credit} muteZero />
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot className="font-semibold">
          <tr className="border-t-2">
            <td className="py-2">
              {fs.total}{" "}
              <StatusBadge kind={balanced ? "good" : "critical"} className="ml-2">
                {balanced ? r.balanced : r.unbalanced}
              </StatusBadge>
            </td>
            <td className="py-2 text-right">
              <Money value={data.totalDebit} />
            </td>
            <td className="py-2 text-right">
              <Money value={data.totalCredit} />
            </td>
          </tr>
        </tfoot>
      </table>
    </ReportPaper>
  );
}

export function BalanceSheetView({ data }: { data: BalanceSheet }) {
  const balanced = Math.abs(parseMoney(data.totalAssets) - parseMoney(data.totalLiabilitiesAndEquity)) < 0.01;
  return (
    <ReportPaper title={r.balanceSheet} subtitle={`${fs.asOf} ${formatDate(data.asOf)}`}>
      <div className="grid gap-6 lg:grid-cols-2 print:grid-cols-2">
        <table className="w-full text-sm">
          <tbody>
            <Section title={r.assets} rows={data.assets} total={data.totalAssets} totalLabel={r.totalAssets} />
          </tbody>
        </table>
        <table className="w-full text-sm">
          <tbody>
            <Section title={r.liabilities} rows={data.liabilities} total={data.totalLiabilities} totalLabel={r.totalLiabilities} />
            <Section
              title={r.equity}
              rows={[
                ...data.equity,
                {
                  id: "cye",
                  code: "",
                  codeDisplay: "",
                  name: r.currentYearEarnings,
                  balance: data.currentYearEarnings,
                } as AccountBalance,
              ]}
              total={data.totalEquity}
              totalLabel={r.totalEquity}
            />
            <Row label={r.totalLE} value={data.totalLiabilitiesAndEquity} strong border />
          </tbody>
        </table>
      </div>
      <p className="mt-3 text-center">
        <StatusBadge kind={balanced ? "good" : "critical"}>{balanced ? r.balanced : r.unbalanced}</StatusBadge>
      </p>
    </ReportPaper>
  );
}

export function IncomeStatementView({ data }: { data: IncomeStatement }) {
  return (
    <ReportPaper title={r.incomeStatement} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      <table className="mx-auto w-full max-w-3xl text-sm">
        <tbody>
          <Section title={r.revenue} rows={data.revenue} total={data.totalRevenue} totalLabel={`${fs.total} ${r.revenue.toLowerCase()}`} />
          <Section title={r.cogs} rows={data.cogs} total={data.totalCogs} totalLabel={`${fs.total} ${r.cogs.toLowerCase()}`} />
          <Row label={r.grossProfit} value={data.grossProfit} strong border />
          <Section title={r.expenses} rows={data.expenses} total={data.totalExpenses} totalLabel={`${fs.total} ${r.expenses.toLowerCase()}`} />
          <Row label={r.operatingIncome} value={data.operatingIncome} strong border />
          <Section title={r.otherIncome} rows={data.otherIncome} total={data.totalOtherIncome} totalLabel={`${fs.total} ${r.otherIncome.toLowerCase()}`} />
          <Section title={r.otherExpenses} rows={data.otherExpenses} total={data.totalOtherExpenses} totalLabel={`${fs.total} ${r.otherExpenses.toLowerCase()}`} />
          <tr className="border-t-2 text-base font-bold">
            <td className="py-2.5">{r.netIncome}</td>
            <td className="py-2.5 text-right">
              <Money value={data.netIncome} signed />
            </td>
          </tr>
        </tbody>
      </table>
    </ReportPaper>
  );
}

export function CashFlowView({ data }: { data: CashFlow }) {
  const sections: [string, CashFlowSection | null][] = [
    [r.operating, data.operating],
    [r.investing, data.investing],
    [r.financing, data.financing],
    [r.nonCash, data.nonCash],
  ];
  return (
    <ReportPaper title={r.cashFlow} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      <table className="mx-auto w-full max-w-3xl text-sm">
        <tbody>
          {sections.map(([title, sec]) =>
            sec ? <Section key={title} title={title} rows={sec.rows} total={sec.total} totalLabel={`${fs.total} ${title.toLowerCase()}`} /> : null,
          )}
          <Row label={r.netCashChange} value={data.netCashChange} strong border />
          <Row label={r.cashOpening} value={data.cashOpening} />
          <tr className="border-t-2 text-base font-bold">
            <td className="py-2.5">{r.cashClosing}</td>
            <td className="py-2.5 text-right">
              <Money value={data.cashClosing} signed />
            </td>
          </tr>
        </tbody>
      </table>
    </ReportPaper>
  );
}

export function SubsidiaryLedgerView({ data }: { data: SubsidiaryLedger }) {
  return (
    <ReportPaper
      title={`${r.subsidiaryLedger} · ${data.partyName}`}
      subtitle={`${data.kind === "AP" ? r.kindAP : r.kindAR} · ${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}
    >
      <table className="w-full min-w-[720px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="py-2 text-left font-medium">{fs.date}</th>
            <th className="py-2 text-left font-medium">{r.docNo}</th>
            <th className="py-2 text-left font-medium">{fs.description}</th>
            <th className="py-2 text-right font-medium">{fs.debit}</th>
            <th className="py-2 text-right font-medium">{fs.credit}</th>
            <th className="py-2 text-right font-medium">{fs.balance}</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b border-border/50 text-muted-foreground">
            <td colSpan={5} className="py-1.5">
              {r.opening}
            </td>
            <td className="py-1.5 text-right">
              <Money value={data.openingBalance} signed />
            </td>
          </tr>
          {data.lines.map((l) => (
            <tr key={l.lineId} className="border-b border-border/50">
              <td className="py-1.5 whitespace-nowrap">{formatDate(l.entryDate)}</td>
              <td className="py-1.5">
                <span className="font-mono text-xs">{l.docNo}</span>
                <span className="block text-xs text-muted-foreground">{r.docType[l.docType] ?? l.docType}</span>
              </td>
              <td className="py-1.5">{l.description}</td>
              <td className="py-1.5 text-right">
                <Money value={l.debit} muteZero />
              </td>
              <td className="py-1.5 text-right">
                <Money value={l.credit} muteZero />
              </td>
              <td className="py-1.5 text-right">
                <Money value={l.runningBalance} signed />
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot className="font-semibold">
          <tr className="border-t-2">
            <td colSpan={3} className="py-2">
              {r.closing}
            </td>
            <td className="py-2 text-right">
              <Money value={data.totalDebit} />
            </td>
            <td className="py-2 text-right">
              <Money value={data.totalCredit} />
            </td>
            <td className="py-2 text-right">
              <Money value={data.closingBalance} signed />
            </td>
          </tr>
        </tfoot>
      </table>
    </ReportPaper>
  );
}
