"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle, CheckCircle2, Loader2, Plus, Trash2 } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { toast } from "@/lib/toast";
import { useErpAction } from "@/features/erp/shared/client";
import { FormField } from "@/features/erp/shared/form-field";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatRupiah } from "@/features/erp/shared/money";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { fiscalCoverageAction, openPeriodAction, saveJournalEntryAction } from "../actions";
import { lineTotals, linesFromEntry, newLine, payloadLines, setLineAmount, type FormLine } from "../journal-form";
import type { FiscalCoverage, JournalEntry, PostableAccount } from "../types";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";

const j = fs.journals;

/** Banner telling whether the chosen date falls in an open period, with a shortcut to open it. */
function CoverageBanner({ date, canApprove }: { date: string; canApprove: boolean }) {
  const [coverage, setCoverage] = React.useState<FiscalCoverage | null>(null);
  const { pending, run } = useErpAction();
  const check = React.useCallback(async (d: string) => {
    const res = await fiscalCoverageAction(d);
    setCoverage(res.ok ? res.data : null);
  }, []);
  React.useEffect(() => {
    if (!date) return;
    let alive = true;
    fiscalCoverageAction(date).then((res) => alive && setCoverage(res.ok ? res.data : null));
    return () => {
      alive = false;
    };
  }, [date]);
  if (!coverage || !date) return null;
  if (coverage.ready)
    return (
      <p className="flex items-center gap-1.5 text-xs text-positive">
        <CheckCircle2 className="size-3.5" /> {j.coverageReady(coverage.period?.name ?? "")}
      </p>
    );
  const s = coverage.suggestion;
  return (
    <Alert className="rounded-xl">
      <AlertTriangle className="size-4" />
      <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
        <span>{s?.message || j.coverageNotReady}</span>
        {s?.canOpen && canApprove ? (
          <Button
            size="xs"
            variant="outline"
            disabled={pending}
            onClick={() =>
              run(() => openPeriodAction(s.period.id, s.closePrevious.length > 0), {
                success: j.periodOpened,
                onOk: () => void check(date),
              })
            }
          >
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
            {j.openPeriod} {s.period.name}
          </Button>
        ) : null}
      </AlertDescription>
    </Alert>
  );
}

export function JournalForm({
  entry,
  accounts,
  canApprove,
}: {
  entry?: JournalEntry;
  accounts: PostableAccount[];
  canApprove: boolean;
}) {
  const router = useRouter();
  const [entryDate, setEntryDate] = React.useState(entry?.entryDate ?? toIsoDate(new Date()));
  const [description, setDescription] = React.useState(entry?.description ?? "");
  const [isRecon, setIsRecon] = React.useState(entry?.isRecon ?? false);
  const [lines, setLines] = React.useState<FormLine[]>(() => (entry ? linesFromEntry(entry) : [newLine(), newLine()]));
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const totals = lineTotals(lines);
  const balanced = totals.diff === 0 && totals.debit > 0;

  const patch = (key: string, p: Partial<FormLine>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...p } : l)));

  const submit = (post: boolean) => {
    const local: Record<string, string[]> = {};
    if (!entryDate) local.entryDate = [j.dateRequired];
    if (!description.trim()) local.description = [j.descriptionRequired];
    const built = payloadLines(lines);
    if ("error" in built) local.lines = [built.error];
    setErrors(local);
    if (Object.keys(local).length || "error" in built) {
      toast.error(Object.values(local)[0][0]);
      return;
    }
    void run(
      () => saveJournalEntryAction(entry?.id ?? null, { entryDate, description, isRecon, lines: built.lines, post }),
      {
        success: post ? j.posted : fs.saved,
        onOk: (saved) => router.push(`/erp/finance/journals/${saved.id}`),
        onFail: (res) => res.fieldErrors && setErrors(res.fieldErrors),
      },
    );
  };

  return (
    <div className="flex flex-col gap-5">
      <Card className="gap-4 p-5">
        <div className="grid gap-4 md:grid-cols-[220px_1fr]">
          <FormField label={fs.date} htmlFor="entryDate" error={errors.entryDate} required>
            <DatePicker id="entryDate" value={entryDate} onChange={setEntryDate} invalid={!!errors.entryDate} />
          </FormField>
          <FormField label={fs.description} htmlFor="description" error={errors.description} required>
            <Textarea id="description" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
          </FormField>
        </div>
        <CoverageBanner date={entryDate} canApprove={canApprove} />
        <div className="flex items-center gap-2">
          <Switch id="isRecon" checked={isRecon} onCheckedChange={setIsRecon} />
          <Label htmlFor="isRecon" className="font-normal">
            {j.reconHint}
          </Label>
        </div>
      </Card>

      <Card className="gap-0 overflow-hidden p-0">
        <div className="flex items-center justify-between px-5 py-3">
          <h2 className="text-sm font-semibold">{j.lines}</h2>
          <Button size="sm" variant="outline" onClick={() => setLines((ls) => [...ls, newLine()])}>
            <Plus className="size-3.5" /> {j.addLine}
          </Button>
        </div>
        <div className="overflow-x-auto border-t">
          <table className="w-full min-w-[760px] text-sm">
            <thead className="bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="w-8 px-3 py-2 text-left font-medium">#</th>
                <th className="px-3 py-2 text-left font-medium">{fs.account}</th>
                <th className="w-44 px-3 py-2 text-right font-medium">{fs.debit}</th>
                <th className="w-44 px-3 py-2 text-right font-medium">{fs.credit}</th>
                <th className="w-56 px-3 py-2 text-left font-medium">{fs.memo}</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody>
              {lines.map((l, i) => (
                <tr key={l.key} className="border-t align-top">
                  <td className="px-3 py-2 pt-4 text-xs text-muted-foreground tabular">{i + 1}</td>
                  <td className="px-3 py-2">
                    <AccountPicker accounts={accounts} value={l.accountId} onChange={(v) => patch(l.key, { accountId: v })} />
                  </td>
                  <td className="px-3 py-2">
                    <MoneyInput value={l.debit} onChange={(v) => setLines((ls) => setLineAmount(ls, l.key, "debit", v))} />
                  </td>
                  <td className="px-3 py-2">
                    <MoneyInput value={l.credit} onChange={(v) => setLines((ls) => setLineAmount(ls, l.key, "credit", v))} />
                  </td>
                  <td className="px-3 py-2">
                    <Input value={l.memo} onChange={(e) => patch(l.key, { memo: e.target.value })} aria-label={fs.memo} />
                  </td>
                  <td className="px-1 py-2">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={j.removeLine}
                      disabled={lines.length <= 2}
                      onClick={() => setLines((ls) => ls.filter((x) => x.key !== l.key))}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t bg-muted/30 font-medium">
              <tr>
                <td colSpan={2} className="px-3 py-2.5 text-right text-xs text-muted-foreground">
                  {fs.total}
                </td>
                <td className="px-3 py-2.5 text-right tabular">{formatRupiah(totals.debit)}</td>
                <td className="px-3 py-2.5 text-right tabular">{formatRupiah(totals.credit)}</td>
                <td colSpan={2} className="px-3 py-2.5">
                  <span className={cn("text-xs", balanced ? "text-positive" : "text-negative")}>
                    {balanced ? j.balanced : `${j.unbalanced} · ${j.diff} ${formatRupiah(Math.abs(totals.diff))}`}
                  </span>
                </td>
              </tr>
            </tfoot>
          </table>
        </div>
        {errors.lines ? <p className="px-5 py-2 text-xs text-destructive">{errors.lines[0]}</p> : null}
      </Card>

      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="ghost" onClick={() => router.back()} disabled={pending}>
          {fs.cancel}
        </Button>
        <Button variant="outline" onClick={() => submit(false)} disabled={pending}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {fs.saveDraft}
        </Button>
        {canApprove ? (
          <Button onClick={() => submit(true)} disabled={pending || !balanced}>
            {fs.saveAndPost}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
