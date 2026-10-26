"use client";

import * as React from "react";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ps } from "../strings";
import type { PublicForm, SubmitPublicFormResponse } from "../types";
import { publicClientFetch } from "./client";
import { FieldInput } from "./field-input";

const UTM = ["utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content"];

/** Public lead form: dynamic fields, UTM attribution, honeypot and timing for spam checks. */
export function PublicLeadForm({ form }: { form: PublicForm }) {
  const [values, setValues] = React.useState<Record<string, string>>({});
  const [honeypot, setHoneypot] = React.useState("");
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [formError, setFormError] = React.useState<string | null>(null);
  const [done, setDone] = React.useState<string | null>(null);
  const [pending, setPending] = React.useState(false);
  const startedAt = React.useRef(0);
  React.useEffect(() => {
    startedAt.current = Date.now();
  }, []);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const missing = Object.fromEntries(
      form.fields.filter((f) => f.required && !values[f.key]?.trim()).map((f) => [f.key, ps.required]),
    );
    setErrors(missing);
    setFormError(null);
    if (Object.keys(missing).length) return;

    const url = new URL(window.location.href);
    const payload: Record<string, unknown> = {};
    for (const f of form.fields) {
      const v = values[f.key] ?? "";
      payload[f.key] = f.type === "checkbox" ? v === "true" : f.type === "number" && v !== "" ? Number(v) : v;
    }
    for (const k of UTM) payload[k] = url.searchParams.get(k) ?? "";
    payload.landing_page = url.toString();
    payload.referrer = document.referrer;
    payload.form_started_at = startedAt.current;
    payload.website_url = honeypot;

    setPending(true);
    try {
      const res = await publicClientFetch<SubmitPublicFormResponse>(`/v1/public/sales/forms/${encodeURIComponent(form.slug)}`, {
        method: "POST",
        body: JSON.stringify({ payload }),
      });
      if (!res.ok) {
        setErrors(Object.fromEntries(res.errors.map((er) => [er.key, er.message])));
        setFormError(res.message || null);
        return;
      }
      const redirect = res.redirectUrl || form.redirectUrl;
      if (redirect && /^https?:\/\//.test(redirect)) {
        window.location.assign(redirect);
        return;
      }
      setDone(res.message || form.successMessage || ps.form.success);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : ps.offline);
    } finally {
      setPending(false);
    }
  };

  if (done)
    return (
      <div className="flex flex-col items-center gap-3 py-10 text-center" role="status">
        <CheckCircle2 className="size-10 text-positive" />
        <p className="max-w-sm text-base">{done}</p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setValues({});
            setDone(null);
            startedAt.current = Date.now();
          }}
        >
          {ps.form.again}
        </Button>
      </div>
    );

  return (
    <form className="flex flex-col gap-4" onSubmit={submit} noValidate>
      {form.fields.map((f) => (
        <FieldInput
          key={f.key}
          field={f}
          value={values[f.key] ?? ""}
          onChange={(v) => setValues((p) => ({ ...p, [f.key]: v }))}
          error={errors[f.key]}
        />
      ))}
      {/* Honeypot: hidden from people, filled by bots. */}
      <div aria-hidden className="absolute -left-[9999px] h-0 w-0 overflow-hidden">
        <label>
          Website
          <input tabIndex={-1} autoComplete="off" value={honeypot} onChange={(e) => setHoneypot(e.target.value)} name="website_url" />
        </label>
      </div>
      {formError ? (
        <p className="text-sm text-destructive" role="alert">
          {formError}
        </p>
      ) : null}
      <Button type="submit" size="lg" disabled={pending}>
        {pending ? <Loader2 className="size-4 animate-spin" /> : null}
        {pending ? ps.sending : form.submitLabel || ps.form.submit}
      </Button>
    </form>
  );
}
