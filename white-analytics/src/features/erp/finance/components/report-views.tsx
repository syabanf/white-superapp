import type * as React from "react";
import type { RatioTrendSeries } from "../dashboard/types";
import Link from "next/link";
import { ArrowRight, BookOpen, FileBarChart, HeartPulse, Landmark, Percent, Scale, Target, Users, Wallet } from "lucide-react";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/format";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import type { TaxRegister, BudgetVsActual, ClientEconomicsReport, EquityChanges, AccountBalance, BalanceSheet, CashFlow, CashFlowSection, IncomeStatement, SubsidiaryLedger, TrialBalance } from "../types";
import { fs } from "../strings";

const r = fs.reports;

export const REPORTS = [
  { key: "trial-balance", label: r.trialBalance, icon: Landmark },
  { key: "balance-sheet", label: r.balanceSheet, icon: Scale },
  { key: "income-statement", label: r.incomeStatement, icon: FileBarChart },
  { key: "cash-flow", label: r.cashFlow, icon: Wallet },
  { key: "general-ledger", label: r.generalLedger, icon: BookOpen },
  { key: "subsidiary-ledger", label: r.subsidiaryLedger, icon: Users },
  { key: "project-profit", label: fs.projects.profitTitle, icon: Target },
  { key: "subledger-recon", label: fs.subledgerRecon.title, icon: Scale },
  { key: "tax-register", label: r.taxRegister, icon: Percent },
  { key: "budget-vs-actual", label: r.budgetVsActual, icon: Target },
  { key: "equity-changes", label: r.equityChanges, icon: Scale },
  { key: "client-economics", label: r.clientEconomics, icon: Users },
  { key: "balance-health", label: r.balanceHealth, icon: HeartPulse },
] as const;

export type ReportKey = (typeof REPORTS)[number]["key"];

export function ReportHub() {
  return (
    <div className="grid gap-(--gap) sm:grid-cols-2 xl:grid-cols-3">
      {REPORTS.map((rep) => (
        <Link
          key={rep.key}
          href={`/erp/finance/reports?r=${rep.key}`}
          className="group lift flex items-start gap-3 rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
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
    <Card className="gap-0 pb-0 print:border-0 print:shadow-none">
      <div className="border-b px-(--card-pad) pb-4 text-center">
        <h2 className="text-lg font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{subtitle}</p>
      </div>
      <div className="overflow-x-auto px-(--card-pad) py-3">{children}</div>
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
      <div className="grid gap-(--gap) lg:grid-cols-2 print:grid-cols-2">
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

export function SubsidiaryLedgerView({ data, hidden = [] }: { data: SubsidiaryLedger; hidden?: string[] }) {
  const showDoc = !hidden.includes("doc");
  const showDesc = !hidden.includes("desc");
  const lead = 1 + Number(showDoc) + Number(showDesc);
  return (
    <ReportPaper
      title={`${r.subsidiaryLedger} · ${data.partyName || r.allPartiesLabel}`}
      subtitle={`${data.kind === "AP" ? r.kindAP : r.kindAR} · ${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}
    >
      <table className="w-full min-w-[720px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="py-2 text-left font-medium">{fs.date}</th>
            {showDoc ? <th className="py-2 text-left font-medium">{r.docNo}</th> : null}
            {showDesc ? <th className="py-2 text-left font-medium">{fs.description}</th> : null}
            <th className="py-2 text-right font-medium">{fs.debit}</th>
            <th className="py-2 text-right font-medium">{fs.credit}</th>
            <th className="py-2 text-right font-medium">{fs.balance}</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b border-border/50 text-muted-foreground">
            <td colSpan={lead + 2} className="py-1.5">
              {r.opening}
            </td>
            <td className="py-1.5 text-right">
              <Money value={data.openingBalance} signed />
            </td>
          </tr>
          {data.lines.map((l) => (
            <tr key={l.lineId} className="border-b border-border/50">
              <td className="py-1.5 whitespace-nowrap">{formatDate(l.entryDate)}</td>
              {showDoc ? <td className="py-1.5">
                <span className="font-mono text-xs">{l.docNo}</span>
                <span className="block text-xs text-muted-foreground">{r.docType[l.docType] ?? l.docType}</span>
              </td> : null}
              {showDesc ? <td className="py-1.5">{l.description}</td> : null}
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
            <td colSpan={lead} className="py-2">
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

export function BudgetVsActualView({ data }: { data: BudgetVsActual }) {
  const rows = data.rows ?? [];
  const groups = [
    { title: r.totalRevenue, rows: rows.filter((x) => x.typeCode === "REVENUE" || x.typeCode === "OTHER_INCOME"), budget: data.totalBudgetRevenue, actual: data.totalActualRevenue, income: true },
    { title: r.totalCost, rows: rows.filter((x) => x.typeCode !== "REVENUE" && x.typeCode !== "OTHER_INCOME"), budget: data.totalBudgetCost, actual: data.totalActualCost, income: false },
  ];
  const profitBudget = parseMoney(data.totalBudgetRevenue) - parseMoney(data.totalBudgetCost);
  const profitActual = parseMoney(data.totalActualRevenue) - parseMoney(data.totalActualCost);
  return (
    <ReportPaper title={r.budgetVsActual} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      {rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{r.noBudget}</p>
      ) : (
        <table className="w-full min-w-[640px] text-sm">
          <thead className="text-xs text-muted-foreground">
            <tr className="border-b">
              <th className="py-2 text-left font-medium">{fs.account}</th>
              <th className="py-2 text-right font-medium">{r.budget}</th>
              <th className="py-2 text-right font-medium">{r.actual}</th>
              <th className="py-2 text-right font-medium">{r.variance}</th>
            </tr>
          </thead>
          {groups.map((g) => (
            <tbody key={g.title}>
              {g.rows.map((x) => (
                <tr key={x.accountId} className="border-b border-border/50">
                  <td className="py-1.5">
                    <span className="font-mono text-xs text-muted-foreground">{x.codeDisplay}</span> {x.name}
                  </td>
                  <td className="py-1.5 text-right"><Money value={x.budget} muteZero /></td>
                  <td className="py-1.5 text-right"><Money value={x.actual} muteZero /></td>
                  <td className={cn("py-1.5 text-right", parseMoney(x.variance) < 0 && "text-negative")}><Money value={x.variance} signed /></td>
                </tr>
              ))}
              <tr className="border-b font-semibold">
                <td className="py-2">{g.title}</td>
                <td className="py-2 text-right"><Money value={g.budget} /></td>
                <td className="py-2 text-right"><Money value={g.actual} /></td>
                <td className="py-2 text-right">
                  <Money value={g.income ? parseMoney(g.actual) - parseMoney(g.budget) : parseMoney(g.budget) - parseMoney(g.actual)} signed />
                </td>
              </tr>
            </tbody>
          ))}
          <tfoot className="font-semibold">
            <tr className="border-t-2">
              <td className="py-2">{r.netPlan}</td>
              <td className="py-2 text-right"><Money value={profitBudget} signed /></td>
              <td className="py-2 text-right"><Money value={profitActual} signed /></td>
              <td className="py-2 text-right"><Money value={profitActual - profitBudget} signed /></td>
            </tr>
          </tfoot>
        </table>
      )}
      <p className="pt-3 text-xs text-muted-foreground">{r.varianceHint}</p>
    </ReportPaper>
  );
}

export function EquityChangesView({ data }: { data: EquityChanges }) {
  return (
    <ReportPaper title={r.equityChanges} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      <table className="w-full min-w-[560px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="py-2 text-left font-medium">{fs.account}</th>
            <th className="py-2 text-right font-medium">{r.openingEquity}</th>
            <th className="py-2 text-right font-medium">{r.movement}</th>
            <th className="py-2 text-right font-medium">{r.closingEquity}</th>
          </tr>
        </thead>
        <tbody>
          {(data.rows ?? []).map((x) => (
            <tr key={x.codeDisplay + x.name} className="border-b border-border/50">
              <td className="py-1.5">
                {x.codeDisplay ? <span className="font-mono text-xs text-muted-foreground">{x.codeDisplay} </span> : null}
                {x.name}
              </td>
              <td className="py-1.5 text-right"><Money value={x.opening} signed /></td>
              <td className="py-1.5 text-right"><Money value={x.movement} signed /></td>
              <td className="py-1.5 text-right"><Money value={x.closing} signed /></td>
            </tr>
          ))}
        </tbody>
        <tfoot className="font-semibold">
          <tr className="border-t-2">
            <td className="py-2">{r.totalEquity}</td>
            <td className="py-2 text-right"><Money value={data.totalOpening} signed /></td>
            <td className="py-2 text-right"><Money value={data.totalMovement} signed /></td>
            <td className="py-2 text-right"><Money value={data.totalClosing} signed /></td>
          </tr>
        </tfoot>
      </table>
    </ReportPaper>
  );
}

function Stat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <p className="label-mono text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold">{value}</p>
    </div>
  );
}

export function ClientEconomicsView({ data }: { data: ClientEconomicsReport }) {
  const clients = data.clients ?? [];
  return (
    <ReportPaper title={r.clientEconomics} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      {clients.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{r.noClients}</p>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-4 pb-4 md:grid-cols-5">
            <Stat label={r.avgPerClient} value={<Money value={data.averagePerClient} />} />
            <Stat label={r.collectionRate} value={`${data.collectionRate.toFixed(0)}%`} />
            <Stat label={r.daysToPay} value={data.hasPayments ? r.days(data.avgDaysToPay) : "–"} />
            <Stat label={r.topClient} value={`${data.topClientShare.toFixed(0)}%`} />
            <Stat label={r.topThree} value={`${data.topThreeShare.toFixed(0)}%`} />
          </div>
          <table className="w-full min-w-[640px] text-sm">
            <thead className="text-xs text-muted-foreground">
              <tr className="border-b">
                <th className="py-2 text-left font-medium">{r.client}</th>
                <th className="py-2 text-right font-medium">{r.invoiceCount}</th>
                <th className="py-2 text-right font-medium">{r.invoiced}</th>
                <th className="py-2 text-right font-medium">{r.collected}</th>
                <th className="py-2 text-right font-medium">{r.outstanding}</th>
                <th className="py-2 text-right font-medium">{r.daysToPay}</th>
              </tr>
            </thead>
            <tbody>
              {clients.map((c) => (
                <tr key={c.party} className="border-b border-border/50">
                  <td className="py-1.5">{c.party}</td>
                  <td className="py-1.5 text-right tabular">{c.invoiceCount}</td>
                  <td className="py-1.5 text-right"><Money value={c.invoiced} /></td>
                  <td className="py-1.5 text-right"><Money value={c.collected} muteZero /></td>
                  <td className="py-1.5 text-right"><Money value={c.outstanding} muteZero /></td>
                  <td className="py-1.5 text-right tabular">{c.hasPayments ? r.days(c.avgDaysToPay) : "–"}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="font-semibold">
              <tr className="border-t-2">
                <td className="py-2" colSpan={2}>{fs.total}</td>
                <td className="py-2 text-right"><Money value={data.totalInvoiced} /></td>
                <td className="py-2 text-right"><Money value={data.totalCollected} /></td>
                <td className="py-2 text-right"><Money value={parseMoney(data.totalInvoiced) - parseMoney(data.totalCollected)} /></td>
                <td />
              </tr>
            </tfoot>
          </table>
        </>
      )}
      <p className="pt-3 text-xs text-muted-foreground">{r.economicsNote}</p>
    </ReportPaper>
  );
}

const HEALTH_KEYS = ["current_ratio", "quick_ratio", "cash_ratio", "debt_to_equity", "dso", "dpo", "runway_months"];

/** The latest reading of the balance ratios against the agency benchmarks. */
export function BalanceHealthView({ series, asOf }: { series: RatioTrendSeries[]; asOf: string }) {
  const latest = (s?: RatioTrendSeries) => {
    const p = s?.points.findLast((x) => !x.unavailable);
    return p?.value;
  };
  const rows = HEALTH_KEYS.flatMap((k) => series.filter((s) => s.key === k));
  const dso = latest(series.find((s) => s.key === "dso"));
  const dpo = latest(series.find((s) => s.key === "dpo"));
  const show = (v: number, unit: string) => (unit === "days" ? r.days(v) : unit === "months" ? `${v.toFixed(1)} bulan` : v.toFixed(2));
  return (
    <ReportPaper title={r.balanceHealth} subtitle={formatDate(asOf)}>
      {rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{r.noHealth}</p>
      ) : (
        <table className="w-full min-w-[520px] text-sm">
          <thead className="text-xs text-muted-foreground">
            <tr className="border-b">
              <th className="py-2 text-left font-medium">{r.ratio}</th>
              <th className="py-2 text-right font-medium">{r.latest}</th>
              <th className="py-2 text-right font-medium">{r.benchmark}</th>
              <th className="py-2 text-right font-medium">{r.health}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((s) => {
              const v = latest(s);
              const benchmark = s.benchmark ?? undefined;
              const known = v !== undefined && benchmark !== undefined;
              const good = known && (s.lowerIsBetter ? v <= benchmark : v >= benchmark);
              return (
                <tr key={s.key} className="border-b border-border/50">
                  <td className="py-1.5">{s.label}</td>
                  <td className="py-1.5 text-right tabular">{v === undefined ? "–" : show(v, s.unit)}</td>
                  <td className="py-1.5 text-right tabular text-muted-foreground">{benchmark === undefined ? "–" : `${s.lowerIsBetter ? "≤" : "≥"} ${show(benchmark, s.unit)}`}</td>
                  <td className={cn("py-1.5 text-right", known && !good && "text-negative")}>{known ? (good ? r.healthy : r.watch) : "–"}</td>
                </tr>
              );
            })}
            {dso !== undefined && dpo !== undefined ? (
              <tr>
                <td className="py-1.5">{r.cashCycle}</td>
                <td className="py-1.5 text-right tabular">{r.days(dso - dpo)}</td>
                <td />
                <td />
              </tr>
            ) : null}
          </tbody>
        </table>
      )}
    </ReportPaper>
  );
}

export function TaxRegisterView({ data }: { data: TaxRegister }) {
  const rows = data.rows ?? [];
  const groups = [
    { title: r.sales, rows: rows.filter((x) => x.kind === "AR") },
    { title: r.purchases, rows: rows.filter((x) => x.kind === "AP") },
  ].filter((g) => g.rows.length > 0);
  return (
    <ReportPaper title={r.taxRegister} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      {rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{r.noTaxes}</p>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-4 pb-4 md:grid-cols-5">
            <Stat label={r.outputVat} value={<Money value={data.outputVat} />} />
            <Stat label={r.inputVat} value={<Money value={data.inputVat} />} />
            <Stat label={r.vatPosition} value={<Money value={data.vatPosition} signed />} />
            <Stat label={r.withheldByClients} value={<Money value={data.withheldByClients} />} />
            <Stat label={r.withheldFromVendors} value={<Money value={data.withheldFromVendors} />} />
          </div>
          <table className="w-full min-w-[560px] text-sm">
            <thead className="text-xs text-muted-foreground">
              <tr className="border-b">
                <th className="py-2 text-left font-medium">{r.taxCode}</th>
                <th className="py-2 text-right font-medium">{r.invoices}</th>
                <th className="py-2 text-right font-medium">{r.taxBase}</th>
                <th className="py-2 text-right font-medium">{r.taxAmount}</th>
              </tr>
            </thead>
            {groups.map((g) => (
              <tbody key={g.title}>
                <tr>
                  <td colSpan={4} className="pt-3 pb-1 text-xs font-semibold">{g.title}</td>
                </tr>
                {g.rows.map((x) => (
                  <tr key={x.kind + x.taxCode} className="border-b border-border/50">
                    <td className="py-1.5">
                      {x.taxCode ? <span className="font-mono text-xs text-muted-foreground">{x.taxCode} </span> : null}
                      {x.taxName || r.noTaxCode}
                    </td>
                    <td className="py-1.5 text-right tabular">{x.invoiceCount}</td>
                    <td className="py-1.5 text-right"><Money value={x.base} /></td>
                    <td className="py-1.5 text-right"><Money value={x.amount} /></td>
                  </tr>
                ))}
              </tbody>
            ))}
          </table>
        </>
      )}
      <p className="pt-3 text-xs text-muted-foreground">{r.taxRegisterNote}</p>
    </ReportPaper>
  );
}
