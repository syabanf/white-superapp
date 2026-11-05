"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { deleteFiscalPositionAction, deletePartyProfileAction, saveFiscalPositionAction, savePartyProfileAction } from "../actions";
import { fs } from "../strings";
import type { FiscalPosition, PartyTaxProfile, TaxCode } from "../types";
import { Check, Trigger } from "./tax-setup";

const s = fs.taxSetup;
const options = (map: Record<string, string>) => Object.entries(map).map(([value, label]) => ({ value, label }));

function FiscalForm({ position, taxes, close }: { position?: FiscalPosition; taxes: TaxCode[]; close: () => void }) {
  const [v, setV] = React.useState<FiscalPosition>(position ?? { id: "", name: "", rules: [], isActive: true, notes: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const rules = v.rules ?? [];
  const taxOptions = taxes.map((t) => ({ value: t.id, label: `${t.code} · ${t.name}` }));
  const setRule = (i: number, patch: Partial<FiscalPosition["rules"][number]>) => setV((p) => ({ ...p, rules: rules.map((r, n) => (n === i ? { ...r, ...patch } : r)) }));
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveFiscalPositionAction({ ...v, rules }), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.name} *`} htmlFor="fp-name" error={errors?.name}>
        <Input id="fp-name" value={v.name} onChange={(e) => setV((p) => ({ ...p, name: e.target.value }))} />
      </Fld>
      <div className="flex flex-col gap-2">
        <span className="text-sm font-medium">{s.fiscalRules}</span>
        <p className="text-xs text-muted-foreground">{s.fiscalRulesHint}</p>
        {errors?.rules ? <p className="text-xs text-negative">{errors.rules[0]}</p> : null}
        {rules.map((r, i) => (
          <div key={i} className="grid items-center gap-2 sm:grid-cols-[1fr_1fr_auto]">
            <SimpleSelect value={r.fromTaxId} onChange={(x) => setRule(i, { fromTaxId: x })} placeholder={s.fromTax} options={taxOptions} />
            <SimpleSelect value={r.toTaxId || "-"} onChange={(x) => setRule(i, { toTaxId: x === "-" ? "" : x })} options={[{ value: "-", label: s.removeTax }, ...taxOptions]} />
            <Button type="button" size="icon-xs" variant="ghost" aria-label={s.removeRule} onClick={() => setV((p) => ({ ...p, rules: rules.filter((_, n) => n !== i) }))}>
              <X className="size-3.5" />
            </Button>
          </div>
        ))}
        <Button type="button" size="xs" variant="outline" className="self-start" onClick={() => setV((p) => ({ ...p, rules: [...rules, { fromTaxId: "", toTaxId: "" }] }))}>
          <Plus className="size-3.5" /> {s.addRule}
        </Button>
      </div>
      <Fld label={s.notes} htmlFor="fp-notes">
        <Input id="fp-notes" value={v.notes} onChange={(e) => setV((p) => ({ ...p, notes: e.target.value }))} />
      </Fld>
      <Check label={s.active} checked={v.isActive} onChange={(x) => setV((p) => ({ ...p, isActive: x }))} />
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

export function FiscalDialog({ position, taxes }: { position?: FiscalPosition; taxes: TaxCode[] }) {
  const title = position ? s.editFiscal : s.newFiscal;
  return (
    <FormDialog title={title} size="lg" trigger={<Trigger edit={Boolean(position)} label={title} />}>
      {(close) => <FiscalForm position={position} taxes={taxes} close={close} />}
    </FormDialog>
  );
}

const DeleteTrigger = () => (
  <Button size="icon-xs" variant="ghost" aria-label={s.remove} className="text-destructive">
    <Trash2 className="size-3.5" />
  </Button>
);

export function FiscalTable({ rows, taxes, canApprove }: { rows: FiscalPosition[]; taxes: TaxCode[]; canApprove: boolean }) {
  const code = (id: string) => taxes.find((t) => t.id === id)?.code ?? id;
  const columns: ColumnDef<FiscalPosition, unknown>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          {row.original.isActive ? null : <div className="text-xs text-muted-foreground">{s.inactive}</div>}
        </div>
      ),
    },
    {
      id: "rules",
      header: s.fiscalRules,
      enableSorting: false,
      cell: ({ row }) => (row.original.rules ?? []).map((r) => `${code(r.fromTaxId)} → ${r.toTaxId ? code(r.toTaxId) : s.removeTax}`).join(", ") || "–",
    },
    { accessorKey: "notes", header: s.notes },
    ...(canApprove
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: FiscalPosition } }) => (
              <span className="flex justify-end gap-0.5">
                <FiscalDialog position={row.original} taxes={taxes} />
                <ConfirmAction title={s.remove} description={s.removeHint} action={() => deleteFiscalPositionAction(row.original.id)} trigger={<DeleteTrigger />} />
              </span>
            ),
          } satisfies ColumnDef<FiscalPosition, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(f) => f.id}
      emptyMessage={s.empty}
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={options(s.statuses)} />
          <ResetFilters params={["q", "status"]} />
        </>
      }
    />
  );
}

function ProfileForm({ profile, positions, close }: { profile?: PartyTaxProfile; positions: FiscalPosition[]; close: () => void }) {
  const [v, setV] = React.useState<PartyTaxProfile>(profile ?? { id: "", side: "SUPPLIER", name: "", npwp: "", address: "", isPkp: true, fiscalPositionId: "", notes: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof PartyTaxProfile>(k: K, x: PartyTaxProfile[K]) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => savePartyProfileAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.side} htmlFor="pp-side">
        <SimpleSelect id="pp-side" value={v.side} onChange={(x) => set("side", x as PartyTaxProfile["side"])} options={options(s.sides)} />
      </Fld>
      <Fld label={`${s.partyName} *`} htmlFor="pp-name" error={errors?.name} hint={s.partyNameHint}>
        <Input id="pp-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.npwp} htmlFor="pp-npwp" error={errors?.npwp}>
        <Input id="pp-npwp" inputMode="numeric" value={v.npwp} onChange={(e) => set("npwp", e.target.value)} />
      </Fld>
      <Fld label={s.fiscalPosition} htmlFor="pp-fp">
        <SimpleSelect
          id="pp-fp"
          value={v.fiscalPositionId || "-"}
          onChange={(x) => set("fiscalPositionId", x === "-" ? "" : x)}
          options={[{ value: "-", label: s.noFiscal }, ...positions.filter((f) => f.isActive || f.id === v.fiscalPositionId).map((f) => ({ value: f.id, label: f.name }))]}
        />
      </Fld>
      <Fld label={s.address} htmlFor="pp-address" className="sm:col-span-2">
        <Input id="pp-address" value={v.address} onChange={(e) => set("address", e.target.value)} />
      </Fld>
      <Fld label={s.notes} htmlFor="pp-notes" className="sm:col-span-2">
        <Input id="pp-notes" value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <Check label={s.isPkp} checked={v.isPkp} onChange={(x) => set("isPkp", x)} />
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function ProfileDialog({ profile, positions }: { profile?: PartyTaxProfile; positions: FiscalPosition[] }) {
  const title = profile ? s.editProfile : s.newProfile;
  return (
    <FormDialog title={title} size="lg" trigger={<Trigger edit={Boolean(profile)} label={title} />}>
      {(close) => <ProfileForm profile={profile} positions={positions} close={close} />}
    </FormDialog>
  );
}

export function ProfilesTable({ rows, positions, canApprove }: { rows: PartyTaxProfile[]; positions: FiscalPosition[]; canApprove: boolean }) {
  const columns: ColumnDef<PartyTaxProfile, unknown>[] = [
    { accessorKey: "name", header: s.partyName, cell: ({ row }) => <span className="font-medium">{row.original.name}</span> },
    { accessorKey: "side", header: s.side, cell: ({ row }) => s.sides[row.original.side] },
    { accessorKey: "npwp", header: s.npwp, cell: ({ row }) => row.original.npwp || "–" },
    { accessorKey: "isPkp", header: s.isPkp, cell: ({ row }) => (row.original.isPkp ? s.pkp.yes : s.pkp.no) },
    { id: "fiscal", header: s.fiscalPosition, cell: ({ row }) => positions.find((f) => f.id === row.original.fiscalPositionId)?.name ?? "–" },
    ...(canApprove
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: PartyTaxProfile } }) => (
              <span className="flex justify-end gap-0.5">
                <ProfileDialog profile={row.original} positions={positions} />
                <ConfirmAction title={s.remove} description={s.removeHint} action={() => deletePartyProfileAction(row.original.id)} trigger={<DeleteTrigger />} />
              </span>
            ),
          } satisfies ColumnDef<PartyTaxProfile, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(p) => p.id}
      emptyMessage={s.empty}
      exportName="profil-pajak"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="side" label={s.side} allLabel={s.allSides} options={options(s.sides)} />
          <SelectFilter param="pkp" label={s.isPkp} allLabel={s.allPkp} options={options(s.pkp)} />
          <ResetFilters params={["q", "side", "pkp"]} />
        </>
      }
    />
  );
}
