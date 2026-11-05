"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Ban, Plus, Scale, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate } from "@/lib/format";
import { closeVatAction, createDeferralAction, deleteDeferralAction, remitTaxAction, setDeferralActiveAction, setSeparateApproverAction, voidTaxRemittanceAction } from "../actions";
import { fs } from "../strings";
import type { Deferral, PostableAccount, TaxPosition, TaxRemittance } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.periodEnd;
const lastMonth = () => {
  const d = new Date();
  d.setDate(0);
  return toIsoDate(d).slice(0, 7);
};
type Errors = Record<string, string[]> | undefined;

function DeferralForm({ accounts, close }: { accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState({ kind: "REVENUE", name: "", partyName: "", reference: "", deferredAccountId: accounts.find((a) => a.code === "2105001")?.id ?? "", targetAccountId: "", startDate: toIsoDate(new Date()), months: 12, amount: "" });
  const [errors, setErrors] = React.useState<Errors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createDeferralAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.kind} htmlFor="df-kind" className="sm:col-span-2" error={errors?.kind}>
        <SimpleSelect
          id="df-kind"
          value={v.kind}
          onChange={(x) => set({ kind: x, deferredAccountId: accounts.find((a) => a.code === (x === "REVENUE" ? "2105001" : "1105001"))?.id ?? "", targetAccountId: "" })}
          options={Object.entries(s.deferralKinds).map(([value, label]) => ({ value, label }))}
        />
      </Fld>
      <Fld label={s.name} htmlFor="df-name" className="sm:col-span-2" error={errors?.name}>
        <Input id="df-name" value={v.name} onChange={(e) => set({ name: e.target.value })} />
      </Fld>
      <Fld label={s.party} htmlFor="df-party">
        <Input id="df-party" value={v.partyName} onChange={(e) => set({ partyName: e.target.value })} />
      </Fld>
      <Fld label={s.reference} htmlFor="df-ref">
        <Input id="df-ref" value={v.reference} onChange={(e) => set({ reference: e.target.value })} />
      </Fld>
      <Fld label={s.deferredAccount[v.kind]} htmlFor="df-deferred" error={errors?.deferred_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={v.deferredAccountId} onChange={(x) => set({ deferredAccountId: x })} />
      </Fld>
      <Fld label={s.targetAccount[v.kind]} htmlFor="df-target" error={errors?.target_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={v.targetAccountId} onChange={(x) => set({ targetAccountId: x })} />
      </Fld>
      <Fld label={s.amount} htmlFor="df-amount" error={errors?.amount}>
        <MoneyInput id="df-amount" value={v.amount} onChange={(x) => set({ amount: x })} />
      </Fld>
      <Fld label={s.months} htmlFor="df-months" error={errors?.months}>
        <Input id="df-months" type="number" min={1} max={120} value={v.months} onChange={(e) => set({ months: Number(e.target.value) })} />
      </Fld>
      <Fld label={s.startDate} htmlFor="df-start" error={errors?.start_date}>
        <DatePicker id="df-start" value={v.startDate} onChange={(x) => set({ startDate: x })} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function DeferralDialog({ accounts }: { accounts: PostableAccount[] }) {
  return (
    <FormDialog
      title={s.deferralNew}
      description={s.deferralHint}
      size="lg"
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {s.deferralNew}
        </Button>
      }
    >
      {(close) => <DeferralForm accounts={accounts} close={close} />}
    </FormDialog>
  );
}

export function DeferralsTable({ rows, canApprove }: { rows: Deferral[]; canApprove: boolean }) {
  const { run } = useErpAction();
  const columns: ColumnDef<Deferral>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row: { original: d } }) => (
        <span>
          {d.name}
          <span className="block text-xs text-muted-foreground">{[s.deferralKinds[d.kind], d.partyName, d.reference].filter(Boolean).join(" · ")}</span>
        </span>
      ),
    },
    { accessorKey: "targetAccountName", header: s.account, cell: ({ row: { original: d } }) => <span className="text-xs">{d.deferredAccountName} → {d.targetAccountName}</span> },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "recognized", header: s.recognized, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.recognized} muteZero /> },
    { id: "progress", header: s.progress, cell: ({ row: { original: d } }) => <span className="tabular">{d.postedMonths ?? 0}/{d.months}</span> },
    {
      id: "status",
      header: s.status,
      cell: ({ row: { original: d } }) => (
        <span>
          {(d.postedMonths ?? 0) >= d.months ? s.finished : d.isActive ? s.running : s.stopped}
          {d.lastError ? <span className="block text-xs text-destructive">{d.lastError}</span> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: d } }) =>
        canApprove ? (
          <span className="flex justify-end gap-0.5">
            {(d.postedMonths ?? 0) < d.months ? (
              <Button size="xs" variant="ghost" onClick={() => void run(() => setDeferralActiveAction(d.id, !d.isActive), { success: s.updated })}>
                {d.isActive ? s.stop : s.resume}
              </Button>
            ) : null}
            {!d.postedMonths ? (
              <Button size="xs" variant="ghost" className="text-destructive" onClick={() => void run(() => deleteDeferralAction(d.id), { success: s.deleted })}>
                <Trash2 className="size-3.5" /> {s.delete}
              </Button>
            ) : null}
          </span>
        ) : null,
    },
  ];
  return <ErpDataTable columns={columns} data={rows} getRowId={(d) => d.id} emptyMessage={s.deferralEmpty} exportName="jadwal-pengakuan" />;
}

function RemitForm({ positions, accounts, close }: { positions: TaxPosition[]; accounts: PostableAccount[]; close: () => void }) {
  const payable = positions.filter((p) => p.kind === "PAYABLE");
  const [v, setV] = React.useState({ period: lastMonth(), remitDate: toIsoDate(new Date()), accountId: payable.find((p) => parseMoney(p.balance) > 0)?.accountId ?? "", cashAccountId: "", amount: "", ntpn: "", notes: "" });
  const [errors, setErrors] = React.useState<Errors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => remitTaxAction(v), { success: s.remitted, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.remitAccount} htmlFor="rm-account" className="sm:col-span-2" error={errors?.account_id}>
        <SimpleSelect id="rm-account" value={v.accountId} onChange={(x) => set({ accountId: x })} options={payable.map((p) => ({ value: p.accountId, label: `${p.code} ${p.name} · ${formatRupiah(p.balance)}` }))} />
      </Fld>
      <Fld label={s.period} htmlFor="rm-period" error={errors?.period}>
        <Input id="rm-period" value={v.period} onChange={(e) => set({ period: e.target.value })} />
      </Fld>
      <Fld label={s.date} htmlFor="rm-date" error={errors?.remit_date}>
        <DatePicker id="rm-date" value={v.remitDate} onChange={(x) => set({ remitDate: x })} />
      </Fld>
      <Fld label={s.amount} htmlFor="rm-amount" error={errors?.amount}>
        <MoneyInput id="rm-amount" value={v.amount} onChange={(x) => set({ amount: x })} />
      </Fld>
      <Fld label={s.cashAccount} htmlFor="rm-cash" error={errors?.cash_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={v.cashAccountId} onChange={(x) => set({ cashAccountId: x })} />
      </Fld>
      <Fld label={s.ntpn} htmlFor="rm-ntpn">
        <Input id="rm-ntpn" value={v.ntpn} onChange={(e) => set({ ntpn: e.target.value })} />
      </Fld>
      <Fld label={s.notes} htmlFor="rm-notes">
        <Input id="rm-notes" value={v.notes} onChange={(e) => set({ notes: e.target.value })} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function CloseVatForm({ close }: { close: () => void }) {
  const [period, setPeriod] = React.useState(lastMonth());
  const [errors, setErrors] = React.useState<Errors>();
  const [done, setDone] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => closeVatAction(period), {
          onOk: (r) => setDone(s.closeVatDone(formatRupiah(r.offset), formatRupiah(r.payable), formatRupiah(r.compensation))),
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={s.period} htmlFor="vat-period" error={errors?.period}>
        <Input id="vat-period" value={period} onChange={(e) => setPeriod(e.target.value)} />
      </Fld>
      {done ? <p className="text-sm">{done}</p> : null}
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

export function TaxActions({ positions, accounts }: { positions: TaxPosition[]; accounts: PostableAccount[] }) {
  return (
    <div className="flex flex-wrap gap-2">
      <FormDialog
        title={s.closeVat}
        description={s.closeVatHint}
        trigger={
          <Button size="sm" variant="outline">
            <Scale className="size-4" /> {s.closeVat}
          </Button>
        }
      >
        {(close) => <CloseVatForm close={close} />}
      </FormDialog>
      <FormDialog
        title={s.remit}
        description={s.remitHint}
        size="lg"
        trigger={
          <Button size="sm">
            <Plus className="size-4" /> {s.remit}
          </Button>
        }
      >
        {(close) => <RemitForm positions={positions} accounts={accounts} close={close} />}
      </FormDialog>
    </div>
  );
}

export function PositionsTable({ rows }: { rows: TaxPosition[] }) {
  const columns: ColumnDef<TaxPosition>[] = [
    { accessorKey: "name", header: s.account, cell: ({ row: { original: p } }) => <span><span className="tabular text-muted-foreground">{p.code}</span> {p.name}</span> },
    { accessorKey: "kind", header: s.kind, cell: ({ row }) => s.positionKinds[row.original.kind] },
    { accessorKey: "balance", header: s.balance, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.balance} muteZero /> },
  ];
  return <ErpDataTable columns={columns} data={rows} getRowId={(p) => p.accountId} emptyMessage={s.positionsEmpty} dense />;
}

export function RemittancesTable({ rows, canApprove }: { rows: TaxRemittance[]; canApprove: boolean }) {
  const { run } = useErpAction();
  const columns: ColumnDef<TaxRemittance>[] = [
    { accessorKey: "remitNo", header: s.remitNo, cell: ({ row: { original: r } }) => <Link className="underline-offset-2 hover:underline" href={`/erp/finance/journals/${r.journalEntryId}`}>{r.remitNo}</Link> },
    { accessorKey: "period", header: s.period },
    { accessorKey: "remitDate", header: s.date, cell: ({ row }) => formatDate(row.original.remitDate) },
    { accessorKey: "accountName", header: s.account, cell: ({ row: { original: r } }) => <span>{r.accountName}<span className="block text-xs text-muted-foreground">{r.cashAccountName}</span></span> },
    { accessorKey: "ntpn", header: s.ntpn },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "status", header: s.status, cell: ({ row: { original: r } }) => <span><ErpStatusBadge status={r.status} />{r.voidReason ? <span className="block text-xs text-muted-foreground">{r.voidReason}</span> : null}</span> },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: r } }) =>
        canApprove && r.status === "POSTED" ? (
          <ReasonAction
            trigger={
              <Button size="xs" variant="ghost" className="text-destructive">
                <Ban className="size-3.5" /> {s.void}
              </Button>
            }
            title={s.voidTitle}
            description={s.voidHint}
            confirmLabel={s.void}
            reasonLabel={s.voidReason}
            onConfirm={async (reason: string) => (await run(() => voidTaxRemittanceAction(r.id, reason), { success: s.voided })).ok}
          />
        ) : null,
    },
  ];
  return <ErpDataTable columns={columns} data={rows} getRowId={(r) => r.id} emptyMessage={s.remitEmpty} exportName="setoran-pajak" />;
}

export function ControlsCard({ separateApprover, canApprove }: { separateApprover: boolean; canApprove: boolean }) {
  const { pending, run } = useErpAction();
  return (
    <Card className="flex flex-row items-start justify-between gap-6 p-5">
      <div>
        <h2 className="text-sm font-semibold">{s.separateTitle}</h2>
        <p className="mt-1 max-w-prose text-sm text-muted-foreground">{s.separateHint}</p>
      </div>
      <Switch aria-label={s.separateTitle} checked={separateApprover} disabled={!canApprove || pending} onCheckedChange={(on) => void run(() => setSeparateApproverAction(on), { success: s.separateSaved })} />
    </Card>
  );
}
