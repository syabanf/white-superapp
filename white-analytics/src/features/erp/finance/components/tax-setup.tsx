"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { saveChargeTypeAction, saveTaxCodeAction } from "../actions";
import { fs } from "../strings";
import type { ChargeType, PostableAccount, TaxCode } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.taxSetup;
const options = (map: Record<string, string>) => Object.entries(map).map(([value, label]) => ({ value, label }));
const percentText = (v: number) => `${Number(v.toFixed(2))}%`;

const effectiveRate = (t: Pick<TaxCode, "rate" | "baseNumerator" | "baseDenominator">) => (Number(t.rate) * t.baseNumerator) / (t.baseDenominator || 1);

function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="flex items-center gap-2 text-sm">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} /> {label}
    </label>
  );
}

const BLANK_TAX: TaxCode = {
  id: "", code: "", name: "", kind: "VAT", scope: "BOTH", rate: "", baseNumerator: 1, baseDenominator: 1, priceIncluded: false, accountSaleId: "", accountPurchaseId: "",
  reportCode: "", validFrom: "", validTo: "", isDefaultSale: false, isDefaultPurchase: false, isActive: true, notes: "",
};

function TaxForm({ tax, accounts, close }: { tax?: TaxCode; accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState<TaxCode>(tax ?? BLANK_TAX);
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof TaxCode>(k: K, x: TaxCode[K]) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveTaxCodeAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.code} *`} htmlFor="tx-code" error={errors?.code}>
        <Input id="tx-code" value={v.code} onChange={(e) => set("code", e.target.value)} />
      </Fld>
      <Fld label={`${s.name} *`} htmlFor="tx-name" error={errors?.name}>
        <Input id="tx-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.kind} htmlFor="tx-kind" error={errors?.kind}>
        <SimpleSelect id="tx-kind" value={v.kind} onChange={(x) => setV((p) => ({ ...p, kind: x as TaxCode["kind"], priceIncluded: x === "VAT" && p.priceIncluded }))} options={options(s.kinds)} />
      </Fld>
      <Fld label={s.scope} htmlFor="tx-scope" error={errors?.scope}>
        <SimpleSelect id="tx-scope" value={v.scope} onChange={(x) => set("scope", x as TaxCode["scope"])} options={options(s.scopes)} />
      </Fld>
      <Fld label={s.rate} htmlFor="tx-rate" error={errors?.rate}>
        <Input id="tx-rate" inputMode="decimal" value={v.rate} onChange={(e) => set("rate", e.target.value.replace(",", "."))} />
      </Fld>
      <Fld label={s.baseRatio} htmlFor="tx-num" error={errors?.base_numerator} hint={`${s.baseHint} ${s.effective}: ${percentText(effectiveRate(v))}.`}>
        <div className="flex items-center gap-2">
          <Input id="tx-num" aria-label={s.baseNumerator} type="number" min={1} value={v.baseNumerator} onChange={(e) => set("baseNumerator", Number(e.target.value))} />
          <span>/</span>
          <Input aria-label={s.baseDenominator} type="number" min={1} value={v.baseDenominator} onChange={(e) => set("baseDenominator", Number(e.target.value))} />
        </div>
      </Fld>
      <Fld label={s.accountSale} htmlFor="tx-sale" error={errors?.account_sale_id} hint={v.kind === "VAT" ? s.accountHintVat : s.accountHintWh} className="sm:col-span-2">
        <AccountPicker id="tx-sale" accounts={accounts} value={v.accountSaleId} onChange={(x) => set("accountSaleId", x)} clearable />
      </Fld>
      <Fld label={s.accountPurchase} htmlFor="tx-purchase" className="sm:col-span-2">
        <AccountPicker id="tx-purchase" accounts={accounts} value={v.accountPurchaseId} onChange={(x) => set("accountPurchaseId", x)} clearable />
      </Fld>
      <Fld label={s.validFrom} htmlFor="tx-from" error={errors?.valid_from}>
        <DatePicker id="tx-from" value={v.validFrom} onChange={(x) => set("validFrom", x)} clearable />
      </Fld>
      <Fld label={s.validTo} htmlFor="tx-to" error={errors?.valid_to}>
        <DatePicker id="tx-to" value={v.validTo} onChange={(x) => set("validTo", x)} clearable />
      </Fld>
      <Fld label={s.reportCode} htmlFor="tx-report" className="sm:col-span-2">
        <Input id="tx-report" value={v.reportCode} onChange={(e) => set("reportCode", e.target.value)} />
      </Fld>
      <Fld label={s.notes} htmlFor="tx-notes" className="sm:col-span-2">
        <Input id="tx-notes" value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="flex flex-wrap gap-x-4 gap-y-2 sm:col-span-2">
        {v.kind === "VAT" ? <Check label={s.priceIncluded} checked={v.priceIncluded} onChange={(x) => set("priceIncluded", x)} /> : null}
        <Check label={s.defaultSale} checked={v.isDefaultSale} onChange={(x) => set("isDefaultSale", x)} />
        <Check label={s.defaultPurchase} checked={v.isDefaultPurchase} onChange={(x) => set("isDefaultPurchase", x)} />
        <Check label={s.active} checked={v.isActive} onChange={(x) => set("isActive", x)} />
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function Trigger({ edit, label }: { edit: boolean; label: string }) {
  return edit ? (
    <Button size="icon-xs" variant="ghost" aria-label={label}>
      <Pencil className="size-3.5" />
    </Button>
  ) : (
    <Button size="sm">
      <Plus className="size-4" /> {label}
    </Button>
  );
}

export function TaxDialog({ tax, accounts }: { tax?: TaxCode; accounts: PostableAccount[] }) {
  const title = tax ? s.editTax : s.newTax;
  return (
    <FormDialog title={title} size="lg" trigger={<Trigger edit={Boolean(tax)} label={title} />}>
      {(close) => <TaxForm tax={tax} accounts={accounts} close={close} />}
    </FormDialog>
  );
}

export function TaxTable({ taxes, accounts, canApprove }: { taxes: TaxCode[]; accounts: PostableAccount[]; canApprove: boolean }) {
  const columns: ColumnDef<TaxCode, unknown>[] = [
    {
      accessorKey: "code",
      header: s.code,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-mono text-xs font-medium">{row.original.code}</div>
          {row.original.isActive ? null : <div className="text-xs text-muted-foreground">{s.inactive}</div>}
        </div>
      ),
    },
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="max-w-xs min-w-0">
          <div>{row.original.name}</div>
          {row.original.notes ? <div className="line-clamp-2 text-xs text-muted-foreground">{row.original.notes}</div> : null}
        </div>
      ),
    },
    { accessorKey: "kind", header: s.kind, cell: ({ row }) => s.kindShort[row.original.kind] },
    { accessorKey: "scope", header: s.scope, cell: ({ row }) => s.scopes[row.original.scope] },
    { id: "effective", header: s.effective, meta: { align: "right" }, cell: ({ row }) => percentText(effectiveRate(row.original)) },
    ...(canApprove
      ? [{ id: "actions", header: "", enableSorting: false, cell: ({ row }: { row: { original: TaxCode } }) => <TaxDialog tax={row.original} accounts={accounts} /> } satisfies ColumnDef<TaxCode, unknown>]
      : []),
  ];
  return <ErpDataTable columns={columns} data={taxes} getRowId={(t) => t.id} emptyMessage={s.empty} />;
}

const BLANK_CHARGE: ChargeType = { id: "", code: "", name: "", scope: "BOTH", computation: "FIXED", value: "", taxable: false, isDeduction: false, accountSaleId: "", accountPurchaseId: "", isActive: true, notes: "" };

function ChargeForm({ charge, accounts, close }: { charge?: ChargeType; accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState<ChargeType>(charge ?? BLANK_CHARGE);
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof ChargeType>(k: K, x: ChargeType[K]) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveChargeTypeAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.code} *`} htmlFor="ch-code" error={errors?.code}>
        <Input id="ch-code" value={v.code} onChange={(e) => set("code", e.target.value)} />
      </Fld>
      <Fld label={`${s.name} *`} htmlFor="ch-name" error={errors?.name}>
        <Input id="ch-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.scope} htmlFor="ch-scope">
        <SimpleSelect id="ch-scope" value={v.scope} onChange={(x) => set("scope", x as ChargeType["scope"])} options={options(s.scopes)} />
      </Fld>
      <Fld label={s.computation} htmlFor="ch-comp">
        <SimpleSelect id="ch-comp" value={v.computation} onChange={(x) => set("computation", x as ChargeType["computation"])} options={options(s.computations)} />
      </Fld>
      <Fld label={s.value} htmlFor="ch-value" error={errors?.value} hint={s.valueHint} className="sm:col-span-2">
        {v.computation === "PERCENT" ? (
          <Input id="ch-value" inputMode="decimal" placeholder="%" value={v.value} onChange={(e) => set("value", e.target.value.replace(",", "."))} />
        ) : (
          <MoneyInput id="ch-value" value={v.value} onChange={(x) => set("value", x)} />
        )}
      </Fld>
      <Fld label={s.accountSale} htmlFor="ch-sale" hint={s.chargeAccountHint} className="sm:col-span-2">
        <AccountPicker id="ch-sale" accounts={accounts} value={v.accountSaleId} onChange={(x) => set("accountSaleId", x)} clearable />
      </Fld>
      <Fld label={s.accountPurchase} htmlFor="ch-purchase" className="sm:col-span-2">
        <AccountPicker id="ch-purchase" accounts={accounts} value={v.accountPurchaseId} onChange={(x) => set("accountPurchaseId", x)} clearable />
      </Fld>
      <Fld label={s.notes} htmlFor="ch-notes" className="sm:col-span-2">
        <Input id="ch-notes" value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="flex flex-wrap gap-x-4 gap-y-2 sm:col-span-2">
        <Check label={s.taxable} checked={v.taxable} onChange={(x) => set("taxable", x)} />
        <Check label={s.isDeduction} checked={v.isDeduction} onChange={(x) => set("isDeduction", x)} />
        <Check label={s.active} checked={v.isActive} onChange={(x) => set("isActive", x)} />
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function ChargeDialog({ charge, accounts }: { charge?: ChargeType; accounts: PostableAccount[] }) {
  const title = charge ? s.editCharge : s.newCharge;
  return (
    <FormDialog title={title} size="lg" trigger={<Trigger edit={Boolean(charge)} label={title} />}>
      {(close) => <ChargeForm charge={charge} accounts={accounts} close={close} />}
    </FormDialog>
  );
}

export function ChargeTable({ charges, accounts, canApprove }: { charges: ChargeType[]; accounts: PostableAccount[]; canApprove: boolean }) {
  const columns: ColumnDef<ChargeType, unknown>[] = [
    {
      accessorKey: "code",
      header: s.code,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-mono text-xs font-medium">{row.original.code}</div>
          {row.original.isActive ? null : <div className="text-xs text-muted-foreground">{s.inactive}</div>}
        </div>
      ),
    },
    { accessorKey: "name", header: s.name },
    { accessorKey: "scope", header: s.scope, cell: ({ row }) => s.scopes[row.original.scope] },
    {
      accessorKey: "value",
      header: s.value,
      meta: { align: "right" },
      cell: ({ row }) => {
        const c = row.original;
        if (!Number(c.value)) return <span className="text-xs text-muted-foreground">{s.perDocument}</span>;
        return c.computation === "PERCENT" ? `${Number(c.value)}%` : <Money value={c.value} />;
      },
    },
    { accessorKey: "taxable", header: s.taxable, cell: ({ row }) => (row.original.taxable ? "Ya" : "Tidak") },
    { accessorKey: "isDeduction", header: s.isDeduction, cell: ({ row }) => (row.original.isDeduction ? "Ya" : "Tidak") },
    ...(canApprove
      ? [{ id: "actions", header: "", enableSorting: false, cell: ({ row }: { row: { original: ChargeType } }) => <ChargeDialog charge={row.original} accounts={accounts} /> } satisfies ColumnDef<ChargeType, unknown>]
      : []),
  ];
  return <ErpDataTable columns={columns} data={charges} getRowId={(c) => c.id} emptyMessage={s.empty} />;
}
