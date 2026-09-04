"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { WizardNav } from "@/features/clients/components/wizard/wizard-nav";
import { ClientFields, emptyClientForm, type ClientFormValues, type FieldErrors } from "@/features/clients/components/client-fields";
import { wizardCreateClientAction, type WizardProfileInput } from "@/features/clients/wizard-actions";
import { websiteRequiredFor, type ProjectType } from "@/features/clients/wizard";
import { slugify } from "@/lib/format";
import { tc } from "@/features/clients/strings";

/** Step 2 — creates the client with the chosen types, then jumps to sources. */
export function StepProfile({ types }: { types: ProjectType[] }) {
  const [values, setValues] = React.useState<ClientFormValues>(emptyClientForm);
  const [errors, setErrors] = React.useState<FieldErrors>();
  const [formError, setFormError] = React.useState<string>();
  const [pending, startTransition] = React.useTransition();
  const slugTouched = React.useRef(false);

  const setField = <K extends keyof ClientFormValues>(field: K, value: ClientFormValues[K]) => {
    setValues((v) => {
      const next = { ...v, [field]: value };
      if (field === "slug") slugTouched.current = true;
      // Keep the URL in step with the name until the operator edits it themselves.
      if (field === "name" && !slugTouched.current) next.slug = slugify(String(value));
      return next;
    });
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setErrors(undefined);
    setFormError(undefined);
    startTransition(async () => {
      const payload: WizardProfileInput = {
        name: values.name,
        slug: values.slug || slugify(values.name),
        description: values.description,
        industry: values.industry,
        websiteUrl: values.websiteUrl.trim(),
        currency: values.currency as WizardProfileInput["currency"],
        timezone: values.timezone as WizardProfileInput["timezone"],
      };
      const res = await wizardCreateClientAction(payload, types);
      if (res && !res.ok) {
        setErrors(res.fieldErrors);
        setFormError(res.fieldErrors ? undefined : res.error);
      }
    });
  };

  return (
    <form onSubmit={submit} className="space-y-6">
      <ClientFields values={values} onChange={setField} errors={errors} disabled={pending} idPrefix="wizard" />
      {websiteRequiredFor(types) ? (
        <p className="-mt-2 text-xs text-muted-foreground">{tc.wizard.websiteForSeo}</p>
      ) : null}
      {formError ? (
        <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
          {formError}
        </p>
      ) : null}
      <WizardNav
        back="jenis"
        types={types}
        submit={
          <Button
            type="submit"
            disabled={
              pending ||
              values.name.trim().length === 0 ||
              (websiteRequiredFor(types) && values.websiteUrl.trim().length === 0)
            }
          >
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {pending ? tc.wizard.saving : tc.wizard.next}
          </Button>
        }
      />
    </form>
  );
}
