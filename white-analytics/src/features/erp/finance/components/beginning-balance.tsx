"use client";

import * as React from "react";
import Link from "next/link";
import { Info, Loader2, Search, Wand2 } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Toolbar, ToolbarActions } from "@/components/dashboard/layout";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn } from "@/lib/utils";
import { useErpAction } from "@/features/erp/shared/client";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatRupiah, parseMoney, toMoneyString } from "@/features/erp/shared/money";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { saveBeginningBalanceAction } from "../actions";
import type { BeginningBalance, BeginningBalanceLine } from "../types";
import { fs } from "../strings";

const b = fs.fiscal.bb;

type Row = { line: BeginningBalanceLine; debit: string; credit: string };

function toRow(line: BeginningBalanceLine, amount: string, side: string): Row {
  const has = parseMoney(amount) !== 0;
  return {
    line,
    debit: has && side === "DEBIT" ? toMoneyString(parseMoney(amount)) : "",
    credit: has && side === "CREDIT" ? toMoneyString(parseMoney(amount)) : "",
  };
}

const cents = (v: string) => Math.round(parseMoney(v) * 100);

export function BeginningBalanceEditor({ data, canApprove, canWrite }: { data: BeginningBalance; canApprove: boolean; canWrite: boolean }) {
  const [rows, setRows] = React.useState<Row[]>(() => data.lines.map((l) => toRow(l, l.amount, l.entrySide)));
  const [query, setQuery] = React.useState("");
  const [onlyFilled, setOnlyFilled] = React.useState(false);
  const { pending, run } = useErpAction();
  const editable = data.canEdit && canWrite;

  const debit = rows.reduce((s, r) => s + cents(r.debit), 0) / 100;
  const credit = rows.reduce((s, r) => s + cents(r.credit), 0) / 100;
  const diff = Math.round((debit - credit) * 100) / 100;

  const setAmount = (id: string, side: "debit" | "credit", v: string) =>
    setRows((rs) =>
      rs.map((r) =>
        r.line.accountId === id ? { ...r, [side]: v, [side === "debit" ? "credit" : "debit"]: parseMoney(v) > 0 ? "" : r[side === "debit" ? "credit" : "debit"] } : r,
      ),
    );

  const useSuggested = () =>
    setRows((rs) => rs.map((r) => toRow(r.line, r.line.suggestedAmount, r.line.entrySide !== "ENTRY_SIDE_UNSPECIFIED" ? r.line.entrySide : r.line.normalBalance)));

  const save = (post: boolean) => {
    const lines = rows
      .filter((r) => parseMoney(r.debit) > 0 || parseMoney(r.credit) > 0)
      .map((r) => {
        const isDebit = parseMoney(r.debit) > 0;
        return {
          accountId: r.line.accountId,
          entrySide: (isDebit ? "DEBIT" : "CREDIT") as "DEBIT" | "CREDIT",
          amount: toMoneyString(parseMoney(isDebit ? r.debit : r.credit)),
          memo: "",
        };
      });
    void run(() => saveBeginningBalanceAction(data.fiscalYearId, { lines, post }), { success: b.saved });
  };

  const q = query.trim().toLowerCase();
  const visible = rows.filter(
    (r) =>
      (!onlyFilled || parseMoney(r.debit) > 0 || parseMoney(r.credit) > 0) &&
      (!q || `${r.line.accountCode} ${r.line.accountName}`.toLowerCase().includes(q)),
  );

  return (
    <div className="page-stack">
      <Alert className="rounded-xl">
        <Info className="size-4" />
        <AlertDescription className="grid gap-0.5">
          {data.message ? <span>{data.message}</span> : null}
          <span>{data.isFirstYear ? b.firstYear : data.priorFiscalYear ? b.fromPrior(data.priorFiscalYear.name) : null}</span>
          {!data.isFirstYear && !data.priorFullyClosed ? <span>{b.priorOpen}</span> : null}
          {data.retainedEarningsAccount ? (
            <span>
              {b.retained}: {data.retainedEarningsAccount.codeDisplay} {data.retainedEarningsAccount.name}
            </span>
          ) : null}
          {data.existingEntryId ? (
            <span className="flex items-center gap-2">
              {b.existing}:
              <Link className="underline" href={`/erp/finance/journals/${data.existingEntryId}`}>
                {fs.detail}
              </Link>
              <ErpStatusBadge status={data.existingStatus} />
            </span>
          ) : null}
        </AlertDescription>
      </Alert>
      {!data.canEdit ? <p className="text-sm text-muted-foreground">{b.readonly}</p> : null}

      <Toolbar label={fs.searchAccount}>
        <div className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder={fs.searchAccount} className="w-64 pl-8" aria-label={fs.searchAccount} />
        </div>
        <div className="flex items-center gap-2">
          <Switch id="bb-filled" checked={onlyFilled} onCheckedChange={setOnlyFilled} />
          <Label htmlFor="bb-filled" className="text-xs font-normal text-muted-foreground">
            {b.onlyFilled}
          </Label>
        </div>
        {editable && !data.isFirstYear ? (
          <ToolbarActions>
            <Button variant="outline" onClick={useSuggested}>
              <Wand2 className="size-4" /> {b.useSuggested}
            </Button>
          </ToolbarActions>
        ) : null}
      </Toolbar>

      <Card className="gap-0 overflow-hidden p-0">
        <div className="max-h-[60vh] overflow-auto scrollbar-thin">
          <table className="w-full min-w-[820px] text-sm">
            <thead className="sticky top-0 z-10 bg-card text-xs text-muted-foreground shadow-[0_1px_0_var(--border)]">
              <tr>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-left font-medium">{fs.account}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-left font-medium">{fs.type}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-right font-medium">{b.suggested}</th>
                <th className="w-44 px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-right font-medium">{fs.debit}</th>
                <th className="w-44 px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-right font-medium">{fs.credit}</th>
                <th className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2 text-left font-medium">{b.source}</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((r) => (
                <tr key={r.line.accountId} className="border-t">
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5">
                    <span className="mr-2 font-mono text-xs text-muted-foreground">{r.line.accountCode}</span>
                    {r.line.accountName}
                  </td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5 text-xs text-muted-foreground">{r.line.accountTypeCode}</td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5 text-right text-xs text-muted-foreground">
                    <Money value={r.line.suggestedAmount} muteZero />
                  </td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5">
                    {editable ? (
                      <MoneyInput value={r.debit} onChange={(v) => setAmount(r.line.accountId, "debit", v)} />
                    ) : (
                      <span className="block text-right"><Money value={r.debit} muteZero /></span>
                    )}
                  </td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5">
                    {editable ? (
                      <MoneyInput value={r.credit} onChange={(v) => setAmount(r.line.accountId, "credit", v)} />
                    ) : (
                      <span className="block text-right"><Money value={r.credit} muteZero /></span>
                    )}
                  </td>
                  <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-1.5 text-xs text-muted-foreground">{b.sources[r.line.source] ?? r.line.source}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="sticky bottom-0 bg-card font-semibold shadow-[0_-1px_0_var(--border)]">
              <tr>
                <td colSpan={3} className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right text-xs text-muted-foreground">
                  {fs.total}
                </td>
                <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right tabular">{formatRupiah(debit)}</td>
                <td className="px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-right tabular">{formatRupiah(credit)}</td>
                <td className={cn("px-3 first:pl-(--card-pad) last:pr-(--card-pad) py-2.5 text-xs", diff === 0 ? "text-positive" : "text-negative")}>
                  {diff === 0 ? fs.journals.balanced : b.notBalanced(formatRupiah(Math.abs(diff)))}
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>

      {editable ? (
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={() => save(false)} disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {b.saveDraft}
          </Button>
          {canApprove ? (
            <Button onClick={() => save(true)} disabled={pending || diff !== 0}>
              {b.post}
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
