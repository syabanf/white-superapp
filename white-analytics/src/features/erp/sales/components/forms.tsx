"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowDown, ArrowUp, ExternalLink, Inbox, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { slugify, formatDateTime } from "@/lib/format";
import { deleteFormAction, saveFormAction } from "../actions";
import { FORM_FIELD_TYPE, LEAD_SOURCE, SUBMISSION_STATUS, TEMPERATURE, opts, ss } from "../strings";
import type { FormField, FormSubmission, LeadForm } from "../types";
import { ChipToggle, ConfirmAction, CopyLinkButton, Fld, FormDialog, FormFooter, type Option, SalesBadge, SimpleSelect } from "./ui";

const f = ss.forms;

const EMPTY: LeadForm = {
  id: "",
  slug: "",
  name: "",
  description: "",
  fields: [
    { key: "pic_name", label: "Nama lengkap", type: "text", required: true, options: [], help: "" },
    { key: "org_name", label: "Nama perusahaan", type: "text", required: true, options: [], help: "" },
    { key: "pic_phone", label: "Nomor WhatsApp", type: "phone", required: true, options: [], help: "" },
    { key: "pic_email", label: "Email", type: "email", required: false, options: [], help: "" },
  ],
  submitLabel: "Kirim",
  successMessage: "Terima kasih, tim kami akan segera menghubungi Anda.",
  redirectUrl: "",
  defaultSource: "form",
  leadTemperature: "hangat",
  notifyUserIds: [],
  isActive: true,
  submissionCount: 0,
};

export function FormsList({ forms, owners, canWrite }: { forms: LeadForm[]; owners: Option[]; canWrite: boolean }) {
  return (
    <div className="flex flex-col gap-3">
      {canWrite ? (
        <div>
          <FormBuilderDialog
            form={null}
            owners={owners}
            trigger={
              <Button size="sm">
                <Plus className="size-3.5" /> {f.add}
              </Button>
            }
          />
        </div>
      ) : null}
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {forms.map((form) => (
          <Card key={form.id} className="gap-3 px-5 py-4">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <h3 className="font-medium">{form.name}</h3>
                <p className="truncate font-mono text-xs text-muted-foreground">/f/{form.slug}</p>
              </div>
              {!form.isActive ? <SalesBadge value="ended" map={{ ended: ss.inactive }} /> : null}
            </div>
            {form.description ? <p className="line-clamp-2 text-sm text-muted-foreground">{form.description}</p> : null}
            <p className="text-xs text-muted-foreground">
              {form.fields.length} {f.fields.toLowerCase()} · {f.submissionCount(form.submissionCount)}
            </p>
            <div className="mt-auto flex flex-wrap items-center gap-1.5">
              <Button size="xs" variant="outline" asChild>
                <Link href={`/erp/sales/forms?form=${form.id}`}>
                  <Inbox className="size-3.5" /> {f.submissions}
                </Link>
              </Button>
              <CopyLinkButton path={`/f/${form.slug}`} size="xs" />
              <Button size="icon-xs" variant="ghost" asChild aria-label={f.publicLink}>
                <a href={`/f/${form.slug}`} target="_blank" rel="noreferrer">
                  <ExternalLink className="size-3.5" />
                </a>
              </Button>
              {canWrite ? (
                <span className="ml-auto flex">
                  <FormBuilderDialog
                    form={form}
                    owners={owners}
                    trigger={
                      <Button size="icon-xs" variant="ghost" aria-label={ss.edit}>
                        <Pencil className="size-3.5" />
                      </Button>
                    }
                  />
                  <ConfirmAction
                    title={f.deleteTitle}
                    action={() => deleteFormAction(form.id)}
                    trigger={
                      <Button size="icon-xs" variant="ghost" aria-label={ss.delete}>
                        <Trash2 className="size-3.5" />
                      </Button>
                    }
                  />
                </span>
              ) : null}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}

function FormBuilderDialog({ form, owners, trigger }: { form: LeadForm | null; owners: Option[]; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={form ? f.editTitle : f.addTitle} size="xl">
      {(close) => <FormBuilder form={form ?? EMPTY} owners={owners} close={close} />}
    </FormDialog>
  );
}

function FormBuilder({ form, owners, close }: { form: LeadForm; owners: Option[]; close: () => void }) {
  const [v, setV] = React.useState(form);
  const [slugTouched, setSlugTouched] = React.useState(!!form.id);
  const { pending, run } = useErpAction();
  const set = <K extends keyof LeadForm>(k: K, val: LeadForm[K]) => setV((p) => ({ ...p, [k]: val }));
  const patchField = (i: number, p: Partial<FormField>) => set("fields", v.fields.map((x, j) => (j === i ? { ...x, ...p } : x)));
  const moveField = (i: number, d: number) => {
    const next = [...v.fields];
    const [it] = next.splice(i, 1);
    next.splice(i + d, 0, it);
    set("fields", next);
  };
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveFormAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${f.name} *`} htmlFor="lf-name">
        <Input
          id="lf-name"
          value={v.name}
          onChange={(e) => setV((p) => ({ ...p, name: e.target.value, slug: slugTouched ? p.slug : slugify(e.target.value) }))}
        />
      </Fld>
      <Fld label={`${f.slug} *`} htmlFor="lf-slug" hint={`/f/${v.slug || "…"}`}>
        <Input
          id="lf-slug"
          value={v.slug}
          onChange={(e) => {
            setSlugTouched(true);
            set("slug", slugify(e.target.value));
          }}
        />
      </Fld>
      <Fld label={f.description} htmlFor="lf-desc" className="sm:col-span-2">
        <Textarea id="lf-desc" rows={2} value={v.description} onChange={(e) => set("description", e.target.value)} />
      </Fld>

      <fieldset className="flex flex-col gap-2 sm:col-span-2">
        <legend className="mb-1 flex w-full items-center justify-between text-sm font-medium">{f.fields}</legend>
        {v.fields.map((fld, i) => (
          <div key={i} className="grid gap-2 rounded-lg border p-3 md:grid-cols-[1fr_1fr_140px_auto]">
            <Input aria-label={f.label} placeholder={f.label} value={fld.label} onChange={(e) => patchField(i, { label: e.target.value })} />
            <Input
              aria-label={f.key}
              placeholder={f.key}
              value={fld.key}
              onChange={(e) => patchField(i, { key: e.target.value.replace(/[^a-z0-9_]/gi, "_").toLowerCase() })}
              className="font-mono text-xs"
            />
            <SimpleSelect value={fld.type} onChange={(t) => patchField(i, { type: t })} options={opts(FORM_FIELD_TYPE)} />
            <div className="flex items-center gap-1">
              <label className="flex items-center gap-1.5 px-1 text-xs">
                <Checkbox checked={fld.required} onCheckedChange={(c) => patchField(i, { required: c === true })} /> {f.required}
              </label>
              <Button type="button" size="icon-xs" variant="ghost" aria-label="Naik" disabled={i === 0} onClick={() => moveField(i, -1)}>
                <ArrowUp className="size-3.5" />
              </Button>
              <Button type="button" size="icon-xs" variant="ghost" aria-label="Turun" disabled={i === v.fields.length - 1} onClick={() => moveField(i, 1)}>
                <ArrowDown className="size-3.5" />
              </Button>
              <Button type="button" size="icon-xs" variant="ghost" aria-label={ss.delete} onClick={() => set("fields", v.fields.filter((_, j) => j !== i))}>
                <Trash2 className="size-3.5" />
              </Button>
            </div>
            {fld.type === "select" ? (
              <Input
                aria-label={f.options}
                placeholder={f.options}
                value={fld.options.join(", ")}
                onChange={(e) => patchField(i, { options: e.target.value.split(",").map((o) => o.trimStart()) })}
                onBlur={() => patchField(i, { options: fld.options.map((o) => o.trim()).filter(Boolean) })}
                className="md:col-span-2"
              />
            ) : null}
            <Input
              aria-label={f.help}
              placeholder={f.help}
              value={fld.help}
              onChange={(e) => patchField(i, { help: e.target.value })}
              className={fld.type === "select" ? "md:col-span-2" : "md:col-span-4"}
            />
          </div>
        ))}
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="self-start"
          onClick={() => set("fields", [...v.fields, { key: `field_${v.fields.length + 1}`, label: "", type: "text", required: false, options: [], help: "" }])}
        >
          <Plus className="size-3.5" /> {f.addField}
        </Button>
      </fieldset>

      <Fld label={f.submitLabel} htmlFor="lf-submit">
        <Input id="lf-submit" value={v.submitLabel} onChange={(e) => set("submitLabel", e.target.value)} />
      </Fld>
      <Fld label={f.redirectUrl} htmlFor="lf-redirect">
        <Input id="lf-redirect" type="url" value={v.redirectUrl} onChange={(e) => set("redirectUrl", e.target.value)} placeholder="https://" />
      </Fld>
      <Fld label={f.successMessage} htmlFor="lf-success" className="sm:col-span-2">
        <Input id="lf-success" value={v.successMessage} onChange={(e) => set("successMessage", e.target.value)} />
      </Fld>
      <Fld label={f.defaultSource} htmlFor="lf-source">
        <SimpleSelect id="lf-source" value={v.defaultSource} onChange={(x) => set("defaultSource", x)} options={opts(LEAD_SOURCE)} />
      </Fld>
      <Fld label={f.temperature} htmlFor="lf-temp">
        <SimpleSelect id="lf-temp" value={v.leadTemperature} onChange={(x) => set("leadTemperature", x)} options={opts(TEMPERATURE)} />
      </Fld>
      <Fld label={f.notify} className="sm:col-span-2">
        <ChipToggle value={v.notifyUserIds} onChange={(x) => set("notifyUserIds", x)} options={owners} />
      </Fld>
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox checked={v.isActive} onCheckedChange={(c) => set("isActive", c === true)} /> {ss.active}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

/** Submissions of one form, newest first. */
export function SubmissionsTable({ form, submissions }: { form: LeadForm; submissions: FormSubmission[] }) {
  const keys = form.fields.map((x) => x.key);
  if (!submissions.length) return <p className="rounded-xl border border-dashed py-10 text-center text-sm text-muted-foreground">{f.noSubmissions}</p>;
  return (
    <div className="overflow-x-auto rounded-lg border">
      <table className="w-full text-sm">
        <thead className="bg-muted/40 text-xs text-muted-foreground">
          <tr>
            <th className="px-3 py-2 text-left font-medium whitespace-nowrap">{ss.createdAt}</th>
            {form.fields.map((x) => (
              <th key={x.key} className="px-3 py-2 text-left font-medium whitespace-nowrap">
                {x.label}
              </th>
            ))}
            <th className="px-3 py-2 text-left font-medium">{ss.leads.status}</th>
            <th className="px-3 py-2 text-left font-medium">{f.lead}</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {submissions.map((sub) => (
            <tr key={sub.id}>
              <td className="px-3 py-2 text-xs whitespace-nowrap text-muted-foreground tabular">{formatDateTime(sub.createdAt)}</td>
              {keys.map((k) => (
                <td key={k} className="max-w-56 truncate px-3 py-2">
                  {sub.payload?.[k] == null ? "" : String(sub.payload[k])}
                </td>
              ))}
              <td className="px-3 py-2" title={sub.reason || undefined}>
                <SalesBadge value={sub.status} map={SUBMISSION_STATUS} />
              </td>
              <td className="px-3 py-2">
                {sub.leadId ? (
                  <Link href={`/erp/sales/leads/${sub.leadId}`} className="text-xs hover:underline">
                    {ss.open}
                  </Link>
                ) : null}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
