"use client";

import * as React from "react";
import { Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useErpAction } from "@/features/erp/shared/client";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import { saveBudgetYearAction } from "../actions";
import { fs } from "../strings";
import type { BudgetEntry, PostableAccount } from "../types";

const s = fs.budgets;
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
const key = (accountId: string, month: string) => `${accountId}|${month}`;

/** Twelve months by account. Only changed cells are sent, so a save cannot overwrite a colleague's other cells. */
export function BudgetEditor({ year, entries, accounts, canWrite }: { year: number; entries: BudgetEntry[]; accounts: PostableAccount[]; canWrite: boolean }) {
  const months = MONTHS.map((_, i) => `${year}-${String(i + 1).padStart(2, "0")}`);
  const original = React.useMemo(() => new Map(entries.map((e) => [key(e.accountId, e.month), String(parseMoney(e.amount))])), [entries]);
  const [cells, setCells] = React.useState<Record<string, string>>({});
  const { pending, run } = useErpAction();
  const value = (k: string) => cells[k] ?? original.get(k) ?? "";
  const changed = Object.entries(cells).filter(([k, v]) => (original.get(k) ?? "") !== v);

  function save() {
    const payload = changed.map(([k, v]) => {
      const [accountId, month] = k.split("|");
      return { accountId, month, amount: v === "" ? "0" : v };
    });
    void run(() => saveBudgetYearAction(year, payload), { success: s.saved, onOk: () => setCells({}) });
  }

  return (
    <Card className="gap-0 overflow-hidden py-0">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[1100px] text-sm">
          <thead className="bg-muted/40 text-xs text-muted-foreground">
            <tr>
              <th className="sticky left-0 z-10 bg-muted px-3 py-2 text-left font-medium">{s.account}</th>
              {MONTHS.map((m) => (
                <th key={m} className="px-1 py-2 text-right font-medium">{m}</th>
              ))}
              <th className="px-3 py-2 text-right font-medium">{s.total}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {accounts.map((a) => (
              <tr key={a.id}>
                <td className="sticky left-0 z-10 max-w-56 truncate bg-card px-3 py-1" title={a.name}>
                  <span className="font-mono text-xs text-muted-foreground">{a.codeDisplay}</span> {a.name}
                </td>
                {months.map((m) => {
                  const k = key(a.id, m);
                  return (
                    <td key={m} className="px-1 py-1">
                      <input
                        inputMode="numeric"
                        aria-label={`${a.name} ${m}`}
                        disabled={!canWrite}
                        value={value(k)}
                        onChange={(e) => setCells((c) => ({ ...c, [k]: e.target.value.replace(/\D/g, "") }))}
                        className="h-8 w-20 rounded-md border bg-background px-1.5 text-right text-xs tabular disabled:opacity-60"
                      />
                    </td>
                  );
                })}
                <td className="px-3 py-1 text-right font-medium">
                  <Money value={months.reduce((sum, m) => sum + (Number(value(key(a.id, m))) || 0), 0)} muteZero />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {canWrite ? (
        <div className="flex justify-end border-t p-3">
          <Button size="sm" disabled={pending || changed.length === 0} onClick={save}>
            <Save className="size-4" /> {fs.save}
          </Button>
        </div>
      ) : null}
    </Card>
  );
}
