"use client";

import * as React from "react";
import Link from "next/link";
import { CheckCircle2, Loader2, Lock, LockOpen, Pencil, Plus, Scale, Trash2, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { formatDate } from "@/lib/format";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { FormField } from "@/features/erp/shared/form-field";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import {
  closePeriodAction,
  closePeriodPreviewAction,
  deleteFiscalYearAction,
  openPeriodAction,
  saveFiscalYearAction,
} from "../actions";
import type { FiscalPeriod, FiscalYear, PeriodClosePreview } from "../types";
import type { FinancePerms } from "../perms";
import { fs } from "../strings";

const f = fs.fiscal;

function YearDialog({ year, onClose }: { year?: FiscalYear; onClose: () => void }) {
  const now = new Date().getFullYear();
  const [form, setForm] = React.useState({
    code: year?.code ?? `FY${now}`,
    name: year?.name ?? `Tahun Fiskal ${now}`,
    startDate: year?.startDate ?? `${now}-01-01`,
    endDate: year?.endDate ?? `${now}-12-31`,
    isActive: year?.isActive ?? true,
  });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => saveFiscalYearAction(year?.id ?? null, form), {
              success: fs.saved,
              onOk: onClose,
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{year ? f.editYear : f.newYear}</DialogTitle>
            {!year ? <DialogDescription>{f.autoPeriods}</DialogDescription> : null}
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-[140px_1fr]">
            <FormField label={f.code} htmlFor="fy-code" error={errors.code} required>
              <Input id="fy-code" value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value })} />
            </FormField>
            <FormField label={fs.name} htmlFor="fy-name" error={errors.name} required>
              <Input id="fy-name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={f.start} error={errors.startDate} required>
              <DatePicker value={form.startDate} onChange={(v) => setForm({ ...form, startDate: v })} disabled={!!year} />
            </FormField>
            <FormField label={f.end} error={errors.endDate} required>
              <DatePicker value={form.endDate} onChange={(v) => setForm({ ...form, endDate: v })} disabled={!!year} />
            </FormField>
          </div>
          <div className="flex items-center gap-2">
            <Switch id="fy-active" checked={form.isActive} onCheckedChange={(v) => setForm({ ...form, isActive: v })} />
            <Label htmlFor="fy-active">{f.activeYear}</Label>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** Close dialog: loads the blockers first and only enables "close" when the period is clean. */
function CloseDialog({ period, onClose }: { period: FiscalPeriod; onClose: () => void }) {
  const [preview, setPreview] = React.useState<PeriodClosePreview | null>(null);
  const [error, setError] = React.useState("");
  const { pending, run } = useErpAction();
  React.useEffect(() => {
    let alive = true;
    closePeriodPreviewAction(period.id).then((res) => {
      if (!alive) return;
      if (res.ok) setPreview(res.data);
      else setError(res.error);
    });
    return () => {
      alive = false;
    };
  }, [period.id]);
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{f.closeTitle(period.name)}</DialogTitle>
          <DialogDescription>{f.closeDesc}</DialogDescription>
        </DialogHeader>
        {!preview && !error ? (
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" /> {f.loadingPreview}
          </p>
        ) : null}
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        {preview ? (
          <div className="grid gap-3 text-sm">
            <p className="text-muted-foreground">
              {f.postedCount(preview.postedCount)} · {f.draftCount(preview.draftCount)}
            </p>
            {preview.canClose ? (
              <p className="flex items-center gap-1.5 text-positive">
                <CheckCircle2 className="size-4" /> {f.canClose}
              </p>
            ) : (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3">
                <p className="mb-1.5 font-medium">{f.blockers}</p>
                <ul className="space-y-1">
                  {preview.blockers.map((b) => (
                    <li key={b} className="flex gap-1.5 text-xs">
                      <XCircle className="mt-0.5 size-3.5 shrink-0 text-destructive" /> {b}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {preview.draftEntries.length ? (
              <div>
                <p className="mb-1.5 text-xs font-medium text-muted-foreground">{f.drafts}</p>
                <ul className="max-h-40 space-y-1 overflow-auto text-xs scrollbar-thin">
                  {preview.draftEntries.map((d) => (
                    <li key={d.id}>
                      <Link href={`/erp/finance/journals/${d.id}`} className="hover:underline">
                        <span className="font-mono">{d.entryNo}</span> · {formatDate(d.entryDate)} · {d.description}
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </div>
        ) : null}
        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            {fs.cancel}
          </Button>
          <Button
            disabled={!preview?.canClose || pending}
            onClick={() => run(() => closePeriodAction(period.id), { success: f.closed, onOk: onClose })}
          >
            {pending ? <Loader2 className="size-4 animate-spin" /> : <Lock className="size-4" />}
            {f.closePeriod}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function OpenButton({ period }: { period: FiscalPeriod }) {
  const [closePrevious, setClosePrevious] = React.useState(false);
  const { run } = useErpAction();
  return (
    <ConfirmAction
      trigger={
        <Button size="xs" variant="outline">
          <LockOpen className="size-3.5" /> {f.openPeriod}
        </Button>
      }
      title={`${f.openPeriod} ${period.name}?`}
      confirmLabel={f.openPeriod}
      onConfirm={async () => (await run(() => openPeriodAction(period.id, closePrevious), { success: f.opened })).ok}
    >
      <div className="flex items-center gap-2">
        <Checkbox id={`cp-${period.id}`} checked={closePrevious} onCheckedChange={(v) => setClosePrevious(v === true)} />
        <Label htmlFor={`cp-${period.id}`} className="font-normal">
          {f.closePrevious}
        </Label>
      </div>
    </ConfirmAction>
  );
}

export function FiscalYears({ years, perms }: { years: FiscalYear[]; perms: FinancePerms }) {
  const [dialog, setDialog] = React.useState<{ year?: FiscalYear } | null>(null);
  const [closing, setClosing] = React.useState<FiscalPeriod | null>(null);
  const { run } = useErpAction();
  const sorted = [...years].sort((x, y) => y.startDate.localeCompare(x.startDate));
  return (
    <div className="flex flex-col gap-5">
      {perms.canWrite ? (
        <div className="flex justify-end">
          <Button size="sm" onClick={() => setDialog({})}>
            <Plus className="size-4" /> {f.newYear}
          </Button>
        </div>
      ) : null}
      {sorted.length === 0 ? <ErpEmpty title={f.empty} /> : null}
      {sorted.map((y) => (
        <Card key={y.id} className="gap-0 overflow-hidden p-0">
          <div className="flex flex-wrap items-center gap-3 px-5 py-4">
            <div className="min-w-0 flex-1">
              <h2 className="flex items-center gap-2 font-semibold">
                {y.name}
                <span className="font-mono text-xs text-muted-foreground">{y.code}</span>
                {y.isActive ? <Badge variant="secondary">{f.activeYear}</Badge> : null}
              </h2>
              <p className="text-xs text-muted-foreground">
                {formatDate(y.startDate)} – {formatDate(y.endDate)} · {f.openPeriods(y.openPeriodsCount)}
              </p>
            </div>
            <Button asChild size="sm" variant="outline">
              <Link href={`/erp/finance/fiscal/${y.id}/beginning-balance`}>
                <Scale className="size-3.5" /> {f.beginningBalance}
              </Link>
            </Button>
            {perms.canWrite ? (
              <>
                <Button size="icon-sm" variant="ghost" aria-label={fs.edit} onClick={() => setDialog({ year: y })}>
                  <Pencil className="size-4" />
                </Button>
                <ConfirmAction
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={fs.delete} className="text-destructive">
                      <Trash2 className="size-4" />
                    </Button>
                  }
                  title={fs.confirmDeleteTitle}
                  description={f.deleteDesc(y.name)}
                  confirmLabel={fs.delete}
                  destructive
                  onConfirm={async () => (await run(() => deleteFiscalYearAction(y.id), { success: fs.deleted })).ok}
                />
              </>
            ) : null}
          </div>
          <div className="overflow-x-auto border-t">
            <table className="w-full min-w-[640px] text-sm">
              <thead className="bg-muted/40 text-xs text-muted-foreground">
                <tr>
                  <th className="w-12 px-4 py-2 text-left font-medium">#</th>
                  <th className="px-4 py-2 text-left font-medium">{f.periods}</th>
                  <th className="px-4 py-2 text-left font-medium">{fs.date}</th>
                  <th className="px-4 py-2 text-right font-medium">{fs.dashboard.posted}</th>
                  <th className="px-4 py-2 text-right font-medium">{fs.dashboard.drafts}</th>
                  <th className="px-4 py-2 text-left font-medium">{fs.status}</th>
                  <th className="w-28" />
                </tr>
              </thead>
              <tbody>
                {[...y.periods]
                  .sort((p, q) => p.periodNo - q.periodNo)
                  .map((p) => (
                    <tr key={p.id} className="border-t">
                      <td className="px-4 py-2 text-muted-foreground tabular">{p.periodNo}</td>
                      <td className="px-4 py-2 font-medium">{p.name}</td>
                      <td className="px-4 py-2 text-muted-foreground">
                        {formatDate(p.startDate)} – {formatDate(p.endDate)}
                      </td>
                      <td className="px-4 py-2 text-right tabular">{p.postedCount}</td>
                      <td className="px-4 py-2 text-right tabular">{p.draftCount}</td>
                      <td className="px-4 py-2">
                        <ErpStatusBadge status={p.status} />
                      </td>
                      <td className="px-4 py-2 text-right">
                        {perms.canApprove ? (
                          p.status === "OPEN" ? (
                            <Button size="xs" variant="outline" onClick={() => setClosing(p)}>
                              <Lock className="size-3.5" /> {f.closePeriod}
                            </Button>
                          ) : (
                            <OpenButton period={p} />
                          )
                        ) : null}
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </Card>
      ))}
      {dialog ? <YearDialog year={dialog.year} onClose={() => setDialog(null)} /> : null}
      {closing ? <CloseDialog period={closing} onClose={() => setClosing(null)} /> : null}
    </div>
  );
}
