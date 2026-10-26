"use client";

import * as React from "react";
import { Ban, ChevronDown, Loader2, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ChartCard } from "@/components/dashboard/chart-card";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { InfoHint } from "@/components/dashboard/info-hint";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { MarkdownLite } from "@/features/reports/components/ai-insight-card";
import type { Delta } from "@/lib/metrics";
import { formatDate, formatDateTime, formatNumber, formatPercent } from "@/lib/format";
import { cn } from "@/lib/utils";
import { useErpAction } from "@/features/erp/shared/client";
import { Money } from "@/features/erp/shared/money-display";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { ErpEmpty } from "@/features/erp/shared/states";
import { generateInsightAction } from "../actions";
import type { ApiDelta, ExpenseItem, FinancialAnalysis, FinancialInsight, Metric } from "../types";
import { fs } from "../strings";

const an = fs.analysis;

export function formatMetric(value: number, unit: string): string {
  switch (unit) {
    case "pct":
      return formatPercent(value, 1);
    case "ratio":
      return `${formatNumber(value, 2)}x`;
    case "days":
      return `${formatNumber(value, 0)} hari`;
    case "months":
      return `${formatNumber(value, 1)} bln`;
    case "idr":
      return formatRupiah(value);
    default:
      return formatNumber(value, 2);
  }
}

function toDelta(d: ApiDelta | null | undefined): Delta | null {
  if (!d) return null;
  const direction = d.direction === "UP" ? "up" : d.direction === "DOWN" ? "down" : "flat";
  return { abs: d.abs, pct: d.pct ?? null, direction };
}

function MetricCard({ m }: { m: Metric }) {
  const delta = toDelta(m.delta);
  return (
    <Card className={cn("gap-2 px-4 py-3.5", m.unavailable && "bg-muted/30")}>
      <span className="label-mono line-clamp-2 text-muted-foreground">{m.label}</span>
      {m.unavailable ? (
        <span className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Ban className="size-3.5" /> {an.unavailable}
          <InfoHint>{an.unavailableHint}</InfoHint>
        </span>
      ) : (
        <span className="text-xl font-bold tracking-tight tabular">{formatMetric(m.value, m.unit)}</span>
      )}
      {!m.unavailable && delta && m.delta ? (
        <span className="flex flex-wrap items-center gap-1.5">
          <DeltaBadge delta={delta} lowerIsBetter={m.lowerIsBetter} mode={delta.pct == null ? "abs" : "pct"} digits={delta.pct == null ? 2 : 1} />
          <span className="text-xs text-muted-foreground">{an.previous(formatMetric(m.delta.previous, m.unit))}</span>
        </span>
      ) : null}
    </Card>
  );
}

const GROUPS = ["profitability", "liquidity", "leverage", "efficiency", "cash"] as const;

export function AnalysisView({ data }: { data: FinancialAnalysis }) {
  const hasAny = GROUPS.some((g) => data[g].length > 0);
  if (!hasAny) return <ErpEmpty title={an.empty} />;
  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-3 sm:grid-cols-3">
        <Card className="gap-1 px-5 py-4">
          <span className="label-mono text-muted-foreground">{an.revenue}</span>
          <Money value={data.revenue} className="text-2xl font-bold" />
        </Card>
        <Card className="gap-1 px-5 py-4">
          <span className="label-mono text-muted-foreground">{an.netIncome}</span>
          <Money value={data.netIncome} signed className="text-2xl font-bold" />
        </Card>
        <Card className="gap-1 px-5 py-4">
          <span className="label-mono text-muted-foreground">{an.cash}</span>
          <Money value={data.cashBalance} signed className="text-2xl font-bold" />
        </Card>
      </div>
      {data.previous ? (
        <p className="-mt-3 text-xs text-muted-foreground">{an.vsRange(formatDate(data.previous.from), formatDate(data.previous.to))}</p>
      ) : null}
      {GROUPS.map((g) =>
        data[g].length ? (
          <section key={g} className="space-y-3">
            <h2 className="text-base font-semibold">{an.groups[g]}</h2>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
              {data[g].map((m) => (
                <MetricCard key={m.key} m={m} />
              ))}
            </div>
          </section>
        ) : null,
      )}
      <div className="grid gap-5 lg:grid-cols-2">
        <ChartCard title={an.expenseBreakdown}>
          <HBarChart
            items={data.expenseBreakdown.map((e) => ({
              label: `${e.accountCode} ${e.accountName}`,
              value: parseMoney(e.amount),
              secondary: formatPercent(e.sharePct, 1),
            }))}
            format={(v) => formatRupiah(v)}
            maxItems={10}
            emptyLabel={an.empty}
          />
        </ChartCard>
        <ChartCard title={an.topMovers} description={an.topMoversHint}>
          <MoversTable items={data.topMovers} />
        </ChartCard>
      </div>
    </div>
  );
}

function MoversTable({ items }: { items: ExpenseItem[] }) {
  if (items.length === 0) return <p className="py-6 text-center text-sm text-muted-foreground">{an.empty}</p>;
  return (
    <table className="w-full text-sm">
      <thead className="text-xs text-muted-foreground">
        <tr className="border-b">
          <th className="py-2 text-left font-medium">{fs.account}</th>
          <th className="py-2 text-right font-medium">{an.previousAmount}</th>
          <th className="py-2 text-right font-medium">{fs.amount}</th>
          <th className="py-2 text-right font-medium">{an.change}</th>
        </tr>
      </thead>
      <tbody>
        {items.map((e) => {
          const d = toDelta(e.delta);
          return (
            <tr key={e.accountId} className="border-b border-border/50">
              <td className="py-1.5 pr-2">
                <span className="mr-1.5 font-mono text-xs text-muted-foreground">{e.accountCode}</span>
                {e.accountName}
              </td>
              <td className="py-1.5 text-right text-muted-foreground">
                <Money value={e.previousAmount} />
              </td>
              <td className="py-1.5 text-right">
                <Money value={e.amount} />
              </td>
              <td className="py-1.5 text-right">{d ? <DeltaBadge delta={d} lowerIsBetter /> : "–"}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

/** AI CFO memo for the selected range, with a generate button and earlier memos. */
export function InsightPanel({ from, to, insights }: { from: string; to: string; insights: FinancialInsight[] }) {
  const [language, setLanguage] = React.useState<"id" | "en">("id");
  const { pending, run } = useErpAction();
  const current = insights.find((i) => i.dateFrom === from && i.dateTo === to);
  const others = insights.filter((i) => i !== current);
  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_320px]">
      <Card className="gap-3 p-5">
        <div className="flex flex-wrap items-center gap-2">
          <Sparkles className="size-4 text-brand" />
          <h2 className="font-semibold">{an.insight}</h2>
          <InfoHint>{an.insightHint}</InfoHint>
          <span className="ml-auto flex items-center gap-2">
            <Select value={language} onValueChange={(v) => setLanguage(v as "id" | "en")}>
              <SelectTrigger size="sm" className="w-32" aria-label={an.language}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="id">{an.langId}</SelectItem>
                <SelectItem value="en">{an.langEn}</SelectItem>
              </SelectContent>
            </Select>
            <Button size="sm" disabled={pending} onClick={() => run(() => generateInsightAction(from, to, language), { success: an.generated })}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : <Sparkles className="size-4" />}
              {pending ? an.generating : current ? an.regenerate : an.generate}
            </Button>
          </span>
        </div>
        {current ? (
          <>
            <p className="text-xs text-muted-foreground">
              {formatDate(current.dateFrom)} – {formatDate(current.dateTo)} · {formatDateTime(current.createdAt)}
              {current.model ? ` · ${an.model} ${current.model}` : ""}
            </p>
            <div className="text-sm leading-relaxed">
              <MarkdownLite text={current.content} />
            </div>
          </>
        ) : (
          <p className="py-6 text-center text-sm text-muted-foreground">{an.noInsight}</p>
        )}
      </Card>
      <Card className="h-fit gap-2 p-5">
        <h3 className="text-sm font-semibold">{an.history}</h3>
        {others.length === 0 ? <p className="text-xs text-muted-foreground">{an.noHistory}</p> : null}
        {others.map((i) => (
          <Collapsible key={i.id} className="rounded-lg border">
            <CollapsibleTrigger className="group flex w-full items-center gap-2 px-3 py-2 text-left text-xs">
              <span className="min-w-0 flex-1">
                <span className="block font-medium">
                  {formatDate(i.dateFrom)} – {formatDate(i.dateTo)}
                </span>
                <span className="text-muted-foreground">{formatDateTime(i.createdAt)}</span>
              </span>
              <ChevronDown className="size-3.5 transition-transform group-data-[state=open]:rotate-180" />
            </CollapsibleTrigger>
            <CollapsibleContent className="border-t px-3 py-2 text-xs leading-relaxed">
              <MarkdownLite text={i.content} />
            </CollapsibleContent>
          </Collapsible>
        ))}
      </Card>
    </div>
  );
}
