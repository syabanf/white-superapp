"use client";

import * as React from "react";
import { FileUp, Loader2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { importLeadsAction, saveLeadAction, type LeadValues } from "../actions";
import { LEAD_SOURCE, LEAD_STATUS, ORG_TYPE, TEMPERATURE, opts, ss } from "../strings";
import type { CustomField, ImportLeadsResponse, Lead } from "../types";
import { CustomFieldInputs } from "./custom-fields";
import { type FieldErrors, Fld, FormDialog, FormFooter, type Option, Picker, SimpleSelect } from "./ui";

const s = ss.leads;

function toValues(lead: Lead | null): LeadValues {
  return {
    orgName: lead?.orgName ?? "",
    orgType: lead?.orgType ?? "corporate",
    picName: lead?.picName ?? "",
    picTitle: lead?.picTitle ?? "",
    picPhone: lead?.picPhone ?? "",
    picEmail: lead?.picEmail ?? "",
    city: lead?.city ?? "",
    source: lead?.source ?? "wa",
    temperature: lead?.temperature ?? "hangat",
    status: lead?.status ?? "baru",
    notes: lead?.notes ?? "",
    ownerUserId: lead?.ownerUserId ?? "",
    accountId: lead?.accountId ?? "",
    custom: lead?.custom ?? {},
  };
}

/** Create or edit a lead. */
export function LeadFormDialog({
  lead,
  owners,
  accounts,
  customFields,
  trigger,
  open,
  onOpenChange,
}: {
  lead: Lead | null;
  owners: Option[];
  accounts: Option[];
  customFields: CustomField[];
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (o: boolean) => void;
}) {
  return (
    <FormDialog
      trigger={trigger}
      open={open}
      onOpenChange={onOpenChange}
      title={lead ? s.editTitle : s.addTitle}
      size="lg"
    >
      {(close) => <LeadForm lead={lead} owners={owners} accounts={accounts} customFields={customFields} close={close} />}
    </FormDialog>
  );
}

function LeadForm({
  lead,
  owners,
  accounts,
  customFields,
  close,
}: {
  lead: Lead | null;
  owners: Option[];
  accounts: Option[];
  customFields: CustomField[];
  close: () => void;
}) {
  const router = useRouter();
  const [v, setV] = React.useState<LeadValues>(() => toValues(lead));
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof LeadValues>(k: K, val: LeadValues[K]) => setV((p) => ({ ...p, [k]: val }));
  const text = (k: keyof LeadValues, label: string, props: React.ComponentProps<typeof Input> = {}) => (
    <Fld label={label} htmlFor={`lead-${k}`} error={errors?.[k]}>
      <Input
        id={`lead-${k}`}
        value={String(v[k] ?? "")}
        onChange={(e) => set(k, e.target.value as never)}
        aria-invalid={!!errors?.[k] || undefined}
        {...props}
      />
    </Fld>
  );
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveLeadAction(lead?.id ?? null, v), {
          success: ss.saved,
          onOk: (saved) => {
            close();
            if (!lead) router.push(`/erp/sales/leads/${saved.id}`);
          },
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      {text("orgName", `${s.orgName} *`, { className: "sm:col-span-2" })}
      <Fld label={s.orgType} htmlFor="lead-orgType">
        <SimpleSelect id="lead-orgType" value={v.orgType ?? ""} onChange={(x) => set("orgType", x)} options={opts(ORG_TYPE)} />
      </Fld>
      {text("city", s.city)}
      {text("picName", `${s.picName} *`)}
      {text("picTitle", s.picTitle)}
      {text("picPhone", `${s.picPhone} *`, { inputMode: "tel", placeholder: "0812…" })}
      {text("picEmail", s.picEmail, { type: "email" })}
      <Fld label={s.source} htmlFor="lead-source">
        <SimpleSelect id="lead-source" value={v.source ?? ""} onChange={(x) => set("source", x)} options={opts(LEAD_SOURCE)} />
      </Fld>
      <Fld label={s.temperature} htmlFor="lead-temp">
        <SimpleSelect id="lead-temp" value={v.temperature ?? ""} onChange={(x) => set("temperature", x)} options={opts(TEMPERATURE)} />
      </Fld>
      <Fld label={s.status} htmlFor="lead-status">
        <SimpleSelect id="lead-status" value={v.status ?? ""} onChange={(x) => set("status", x)} options={opts(LEAD_STATUS)} />
      </Fld>
      <Fld label={ss.owner} htmlFor="lead-owner">
        <Picker id="lead-owner" value={v.ownerUserId ?? ""} onChange={(x) => set("ownerUserId", x)} options={owners} allowEmpty={ss.noOwner} />
      </Fld>
      <Fld label={ss.accounts.name} htmlFor="lead-account" className="sm:col-span-2">
        <Picker id="lead-account" value={v.accountId ?? ""} onChange={(x) => set("accountId", x)} options={accounts} allowEmpty={ss.dash} />
      </Fld>
      <Fld label={ss.notes} htmlFor="lead-notes" className="sm:col-span-2">
        <Textarea id="lead-notes" rows={3} value={v.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <CustomFieldInputs fields={customFields} value={v.custom ?? {}} onChange={(c) => set("custom", c)} />
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

/** CSV/XLSX import with per-row errors. */
export function LeadImportDialog() {
  return (
    <FormDialog
      title={s.importTitle}
      description={s.importHint}
      trigger={
        <Button variant="outline">
          <FileUp className="size-4" /> {s.import}
        </Button>
      }
    >
      {(close) => <ImportForm close={close} />}
    </FormDialog>
  );
}

function ImportForm({ close }: { close: () => void }) {
  const [file, setFile] = React.useState<File | null>(null);
  const [result, setResult] = React.useState<ImportLeadsResponse | null>(null);
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!file) return;
        const fd = new FormData();
        fd.append("file", file);
        void run(() => importLeadsAction(fd), {
          onOk: (r) => setResult(r),
        });
      }}
    >
      <Input
        type="file"
        accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
        onChange={(e) => {
          setFile(e.target.files?.[0] ?? null);
          setResult(null);
        }}
        aria-label={s.importPick}
      />
      {result ? (
        <div className="rounded-lg border p-3 text-sm">
          <p className="font-medium">{s.importDone(result.imported, result.skipped)}</p>
          {result.errors.length ? (
            <>
              <p className="mt-2 text-xs text-muted-foreground">{s.importErrors}</p>
              <ul className="mt-1 max-h-40 space-y-0.5 overflow-y-auto text-xs">
                {result.errors.map((er, i) => (
                  <li key={i}>
                    <span className="tabular text-muted-foreground">{s.row(er.row)}</span> {er.message}
                  </li>
                ))}
              </ul>
            </>
          ) : null}
        </div>
      ) : null}
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={close} disabled={pending}>
          {ss.close}
        </Button>
        <Button type="submit" disabled={!file || pending}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {s.importRun}
        </Button>
      </div>
    </form>
  );
}
