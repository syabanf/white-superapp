"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ps } from "../strings";
import type { FormField } from "../types";

/** One dynamic field of a public form or onboarding brief. */
export function FieldInput({
  field,
  value,
  onChange,
  error,
}: {
  field: FormField;
  value: string;
  onChange: (v: string) => void;
  error?: string;
}) {
  const id = `f-${field.key}`;
  const label = field.required ? `${field.label} *` : field.label;
  const invalid = error ? true : undefined;
  let control: React.ReactNode;
  switch (field.type) {
    case "textarea":
      control = <Textarea id={id} rows={4} value={value} onChange={(e) => onChange(e.target.value)} aria-invalid={invalid} required={field.required} />;
      break;
    case "select":
      control = (
        <Select value={value} onValueChange={onChange}>
          <SelectTrigger id={id} className="w-full" aria-invalid={invalid}>
            <SelectValue placeholder={ps.form.pick} />
          </SelectTrigger>
          <SelectContent>
            {field.options.map((o) => (
              <SelectItem key={o} value={o}>
                {o}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      );
      break;
    case "checkbox":
      return (
        <div className="flex flex-col gap-1">
          <label className="flex items-start gap-2 text-sm">
            <Checkbox id={id} checked={value === "true"} onCheckedChange={(c) => onChange(c === true ? "true" : "")} aria-invalid={invalid} />
            <span>{label}</span>
          </label>
          {error ? <p className="text-xs text-destructive">{error}</p> : field.help ? <p className="text-xs text-muted-foreground">{field.help}</p> : null}
        </div>
      );
    default: {
      const type = field.type === "email" ? "email" : field.type === "phone" ? "tel" : field.type === "number" ? "number" : field.type === "date" ? "date" : "text";
      control = (
        <Input
          id={id}
          type={type}
          inputMode={field.type === "phone" ? "tel" : undefined}
          autoComplete={field.type === "email" ? "email" : field.type === "phone" ? "tel" : undefined}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-invalid={invalid}
          required={field.required}
        />
      );
    }
  }
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      {control}
      {error ? <p className="text-xs text-destructive">{error}</p> : field.help ? <p className="text-xs text-muted-foreground">{field.help}</p> : null}
    </div>
  );
}
