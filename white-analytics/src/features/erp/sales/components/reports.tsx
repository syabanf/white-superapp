"use client";

import * as React from "react";
import { Target } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatNumber, formatPercent } from "@/lib/format";
import { cn } from "@/lib/utils";
import { saveTargetsAction } from "../actions";
import { ss } from "../strings";
import type { Forecast, ForecastRow, SalesTarget } from "../types";
import { FormDialog, FormFooter, type Option } from "./ui";

const r = ss.reports;

/** Forecast vs target per salesperson (NüHabit forecast-page). */
export function ForecastTable({ forecast }: { forecast: Forecast }) {
  const rows = forecast.total ? [...forecast.rows, forecast.total] : forecast.rows;
  const cell = "px-3 py-2 text-right tabular whitespace-nowrap";
  return (
    <div className="overflow-x-auto rounded-lg border">
      <table className="w-full text-sm">
        <thead className="bg-muted/40 text-xs text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-left font-medium">{r.salesperson}</th>
            <th className="px-3 py-2 text-right font-medium">{r.targetValue}</th>
            <th className="px-3 py-2 text-right font-medium">{ss.dashboard.won}</th>
            <th className="px-3 py-2 text-right font-medium">{ss.dashboard.attainment}</th>
            <th className="px-3 py-2 text-right font-medium">{ss.dashboard.commit}</th>
            <th className="px-3 py-2 text-right font-medium">{r.bestCase}</th>
            <th className="px-3 py-2 text-right font-medium">{r.pipeline}</th>
            <th className="px-3 py-2 text-right font-medium">{ss.dashboard.weighted}</th>
            <th className="px-3 py-2 text-right font-medium">{r.openDeals}</th>
            <th className="px-3 py-2 text-right font-medium">{r.gap}</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((row) => {
            const total = row.userId === "total";
            return (
              <tr key={row.userId} className={cn(total && "bg-muted/30 font-semibold")}>
                <td className="px-3 py-2">{row.userName}</td>
                <td className={cell}>
                  {formatRupiah(row.targetValue)}
                  {row.targetDeals ? <span className="block text-xs text-muted-foreground">{row.targetDeals} deal</span> : null}
                </td>
                <td className={cell}>
                  {formatRupiah(row.wonValue)}
                  <span className="block text-xs text-muted-foreground">{row.wonDeals} deal</span>
                </td>
                <td className={cell}>
                  <Attainment row={row} />
                </td>
                <td className={cell}>{formatRupiah(row.commitValue)}</td>
                <td className={cell}>{formatRupiah(row.bestCaseValue)}</td>
                <td className={cell}>{formatRupiah(row.pipelineValue)}</td>
                <td className={cell}>{formatRupiah(row.weightedValue)}</td>
                <td className={cell}>{formatNumber(row.openDeals)}</td>
                <td className={cn(cell, parseMoney(row.gap) > 0 && "text-destructive")}>{formatRupiah(row.gap)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function Attainment({ row }: { row: ForecastRow }) {
  if (!parseMoney(row.targetValue)) return <span className="text-muted-foreground">{ss.dash}</span>;
  const pct = row.attainmentPercent;
  return (
    <span className="inline-flex items-center gap-2">
      <span className="h-1.5 w-14 overflow-hidden rounded-full bg-muted">
        <span className={cn("block h-full rounded-full", pct >= 100 ? "bg-positive" : "bg-brand")} style={{ width: `${Math.min(100, pct)}%` }} />
      </span>
      {formatPercent(pct, 0)}
    </span>
  );
}

/** sales.approve: replaces the month's targets (empty user = company target). */
export function TargetsDialog({ month, targets, owners }: { month: string; targets: SalesTarget[]; owners: Option[] }) {
  return (
    <FormDialog
      title={r.targetsTitle(month)}
      size="lg"
      trigger={
        <Button size="sm" variant="outline">
          <Target className="size-3.5" /> {r.editTargets}
        </Button>
      }
    >
      {(close) => <TargetsForm month={month} targets={targets} owners={owners} close={close} />}
    </FormDialog>
  );
}

function TargetsForm({ month, targets, owners, close }: { month: string; targets: SalesTarget[]; owners: Option[]; close: () => void }) {
  const people = [{ value: "", label: r.company }, ...owners];
  const [rows, setRows] = React.useState<SalesTarget[]>(() =>
    people.map((p) => targets.find((t) => t.userId === p.value) ?? { userId: p.value, targetValue: "", targetDeals: 0 }),
  );
  const { pending, run } = useErpAction();
  const patch = (userId: string, p: Partial<SalesTarget>) => setRows((l) => l.map((t) => (t.userId === userId ? { ...t, ...p } : t)));
  return (
    <form
      className="flex flex-col gap-3"
      onSubmit={(e) => {
        e.preventDefault();
        const payload = rows
          .filter((t) => parseMoney(t.targetValue) > 0 || t.targetDeals > 0)
          .map((t) => ({ ...t, targetValue: t.targetValue || "0.00" }));
        void run(() => saveTargetsAction(month, payload), { success: r.targetsSaved, onOk: close });
      }}
    >
      <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)_64px] gap-2 text-xs text-muted-foreground sm:grid-cols-[1fr_180px_96px]">
        <span>{r.salesperson}</span>
        <span>{r.targetValue}</span>
        <span>{r.targetDeals}</span>
      </div>
      {rows.map((t) => (
        <div key={t.userId || "company"} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)_64px] items-center gap-2 sm:grid-cols-[1fr_180px_96px]">
          <span className={cn("truncate text-sm", !t.userId && "font-medium")}>{people.find((p) => p.value === t.userId)?.label}</span>
          <MoneyInput value={t.targetValue} onChange={(v) => patch(t.userId, { targetValue: v })} />
          <Input
            type="number"
            min={0}
            aria-label={r.targetDeals}
            value={t.targetDeals || ""}
            onChange={(e) => patch(t.userId, { targetDeals: Number(e.target.value) || 0 })}
          />
        </div>
      ))}
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}
