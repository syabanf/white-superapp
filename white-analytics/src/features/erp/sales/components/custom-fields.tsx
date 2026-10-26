"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ss } from "../strings";
import type { Custom, CustomField } from "../types";
import { ChipToggle, Fld, SimpleSelect } from "./ui";

/** Inputs for the active custom fields of one record type. */
export function CustomFieldInputs({
  fields,
  value,
  onChange,
}: {
  fields: CustomField[];
  value: Custom;
  onChange: (v: Custom) => void;
}) {
  const active = fields.filter((f) => f.isActive).sort((a, b) => a.sortOrder - b.sortOrder);
  if (!active.length) return null;
  const set = (key: string, v: unknown) => onChange({ ...value, [key]: v });
  return (
    <div className="grid gap-3 sm:col-span-2 sm:grid-cols-2">
      {active.map((f) => {
        const id = `cf-${f.key}`;
        const raw = value[f.key];
        const label = f.isRequired ? `${f.label} *` : f.label;
        if (f.fieldType === "boolean")
          return (
            <label key={f.key} className="flex items-center gap-2 text-sm">
              <Checkbox checked={raw === true} onCheckedChange={(c) => set(f.key, c === true)} />
              {f.label}
            </label>
          );
        if (f.fieldType === "picklist")
          return (
            <Fld key={f.key} label={label} htmlFor={id}>
              <SimpleSelect
                id={id}
                value={typeof raw === "string" ? raw : ""}
                onChange={(v) => set(f.key, v)}
                options={f.options.map((o) => ({ value: o, label: o }))}
                emptyLabel={ss.dash}
              />
            </Fld>
          );
        if (f.fieldType === "multipicklist")
          return (
            <Fld key={f.key} label={label} className="sm:col-span-2">
              <ChipToggle
                value={Array.isArray(raw) ? (raw as string[]) : []}
                onChange={(v) => set(f.key, v)}
                options={f.options.map((o) => ({ value: o, label: o }))}
              />
            </Fld>
          );
        if (f.fieldType === "textarea")
          return (
            <Fld key={f.key} label={label} htmlFor={id} className="sm:col-span-2">
              <Textarea id={id} rows={2} value={String(raw ?? "")} onChange={(e) => set(f.key, e.target.value)} />
            </Fld>
          );
        const type =
          f.fieldType === "number" ? "number" : f.fieldType === "date" ? "date" : f.fieldType === "email" ? "email" : "text";
        return (
          <Fld key={f.key} label={label} htmlFor={id}>
            <Input
              id={id}
              type={type}
              value={raw == null ? "" : String(raw)}
              onChange={(e) =>
                set(f.key, f.fieldType === "number" && e.target.value !== "" ? Number(e.target.value) : e.target.value)
              }
            />
          </Fld>
        );
      })}
    </div>
  );
}

/** Read-only list of custom values. */
export function CustomFieldValues({ fields, value }: { fields: CustomField[]; value: Custom }) {
  const shown = fields.filter((f) => f.isActive && value[f.key] != null && value[f.key] !== "");
  if (!shown.length) return null;
  return (
    <>
      {shown.map((f) => {
        const v = value[f.key];
        const text = Array.isArray(v) ? v.join(", ") : typeof v === "boolean" ? (v ? ss.yes : ss.no) : String(v);
        return (
          <div key={f.key} className="flex min-w-0 flex-col gap-0.5">
            <dt className="text-xs text-muted-foreground">{f.label}</dt>
            <dd className="text-sm break-words">{text}</dd>
          </div>
        );
      })}
    </>
  );
}
