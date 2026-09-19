"use client";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { CLIENT_CURRENCIES, CLIENT_TIMEZONES } from "@/features/clients/constants";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export type ClientFormValues = {
  name: string;
  slug: string;
  description: string;
  industry: string;
  websiteUrl: string;
  currency: string;
  timezone: string;
};

export function emptyClientForm(): ClientFormValues {
  return { name: "", slug: "", description: "", industry: "", websiteUrl: "", currency: "IDR", timezone: "Asia/Jakarta" };
}

export type FieldErrors = Record<string, string[]> | undefined;

function FieldError({ errors, name }: { errors: FieldErrors; name: string }) {
  const msg = errors?.[name]?.[0];
  if (!msg) return null;
  return <p className="text-xs text-destructive">{msg}</p>;
}

/** Shared form fields for create/edit client (dialog + settings profile). */
export function ClientFields({
  values,
  onChange,
  errors,
  disabled,
  idPrefix = "client",
  websiteRequired = false,
  websiteHint,
}: {
  values: ClientFormValues;
  onChange: <K extends keyof ClientFormValues>(field: K, value: ClientFormValues[K]) => void;
  errors?: FieldErrors;
  disabled?: boolean;
  idPrefix?: string;
  websiteRequired?: boolean;
  websiteHint?: string;
}) {
  return (
    <div className="grid gap-5">
      <div className="grid gap-5 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-name`}>
            {t.clients.name} <span className="text-destructive">*</span>
          </Label>
          <Input
            id={`${idPrefix}-name`}
            value={values.name}
            onChange={(e) => onChange("name", e.target.value)}
            disabled={disabled}
            placeholder="Kopi Nusantara"
            required
          />
          <FieldError errors={errors} name="name" />
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-slug`}>{t.clients.slug}</Label>
          <Input
            id={`${idPrefix}-slug`}
            value={values.slug}
            onChange={(e) => onChange("slug", e.target.value)}
            disabled={disabled}
            placeholder="kopi-nusantara"
            className="font-mono text-sm"
          />
          <p className="text-xs text-muted-foreground">{tc.form.slugHint}</p>
          <FieldError errors={errors} name="slug" />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-description`}>
          {t.clients.description} <span className="text-muted-foreground">({t.common.optional})</span>
        </Label>
        <Textarea
          id={`${idPrefix}-description`}
          value={values.description}
          onChange={(e) => onChange("description", e.target.value)}
          disabled={disabled}
          placeholder={tc.form.descriptionPlaceholder}
          rows={2}
        />
        <FieldError errors={errors} name="description" />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-industry`}>{t.clients.industry}</Label>
          <Input
            id={`${idPrefix}-industry`}
            value={values.industry}
            onChange={(e) => onChange("industry", e.target.value)}
            disabled={disabled}
            placeholder={tc.form.industryPlaceholder}
          />
          <FieldError errors={errors} name="industry" />
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-website`}>
            {t.clients.website}
            {websiteRequired ? <span className="text-destructive"> *</span> : null}
          </Label>
          <Input
            id={`${idPrefix}-website`}
            type="url"
            value={values.websiteUrl}
            onChange={(e) => onChange("websiteUrl", e.target.value)}
            disabled={disabled}
            placeholder={tc.form.websitePlaceholder}
            required={websiteRequired}
            aria-required={websiteRequired}
            aria-describedby={websiteHint ? `${idPrefix}-website-hint` : undefined}
          />
          {websiteHint ? (
            <p id={`${idPrefix}-website-hint`} className="text-xs text-muted-foreground">
              {websiteHint}
            </p>
          ) : null}
          <FieldError errors={errors} name="websiteUrl" />
        </div>
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-currency`}>{t.clients.currency}</Label>
          <Select value={values.currency} onValueChange={(v) => onChange("currency", v)} disabled={disabled}>
            <SelectTrigger id={`${idPrefix}-currency`} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CLIENT_CURRENCIES.map((c) => (
                <SelectItem key={c} value={c}>
                  {c}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldError errors={errors} name="currency" />
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-timezone`}>{t.clients.timezone}</Label>
          <Select value={values.timezone} onValueChange={(v) => onChange("timezone", v)} disabled={disabled}>
            <SelectTrigger id={`${idPrefix}-timezone`} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {CLIENT_TIMEZONES.map((z) => (
                <SelectItem key={z} value={z}>
                  {z}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldError errors={errors} name="timezone" />
        </div>
      </div>
    </div>
  );
}
