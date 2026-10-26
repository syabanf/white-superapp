"use client";

import * as React from "react";
import Link from "next/link";
import { CheckCircle2, FileUp, Loader2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { FormField } from "@/features/erp/shared/form-field";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { uploadWithProgress } from "@/features/erp/shared/file-upload";
import { Progress } from "@/components/ui/progress";
import { toRupiahInt } from "../lib";
import { AVAILABILITY, SOURCE, cp } from "../strings";
import { errorMessage, portalPost } from "./client";

type ApplyResponse = { candidateId: string; message: string };

const c = cp.career;
const CV_TYPES = [".pdf", ".doc", ".docx"];
const MAX_CV = 10 * 1024 * 1024;
const SOURCES = ["instagram", "jobstreet", "referral", "jobfair", "other"];
const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const EMPTY = {
  fullName: "",
  email: "",
  phone: "",
  domicile: "",
  lastExperience: "",
  lastEducation: "",
  availability: "",
  expectedSalary: "",
  source: "",
  notes: "",
};

/** Career apply form; the CV is optional and goes as multipart when present. */
export function ApplyForm({ slug }: { slug: string }) {
  const [v, setV] = React.useState(EMPTY);
  const [cv, setCv] = React.useState<File | null>(null);
  const [errors, setErrors] = React.useState<Record<string, string>>({});
  const [formError, setFormError] = React.useState<string | null>(null);
  const [progress, setProgress] = React.useState<number | null>(null);
  const [pending, setPending] = React.useState(false);
  const [done, setDone] = React.useState<string | null>(null);
  const fileRef = React.useRef<HTMLInputElement>(null);
  const set = (k: keyof typeof EMPTY, val: string) => setV((p) => ({ ...p, [k]: val }));

  if (done)
    return (
      <div className="flex flex-col items-center gap-3 py-6 text-center" role="status">
        <CheckCircle2 className="size-10 text-positive" />
        <h2 className="text-lg font-semibold">{c.successTitle}</h2>
        <p className="max-w-md text-sm text-muted-foreground">{done}</p>
        <Button variant="outline" asChild>
          <Link href="/karier">{c.another}</Link>
        </Button>
      </div>
    );

  const validate = () => {
    const e: Record<string, string> = {};
    for (const k of ["fullName", "email", "phone", "domicile"] as const) if (!v[k].trim()) e[k] = c.required;
    if (v.email.trim() && !EMAIL.test(v.email.trim())) e.email = c.emailInvalid;
    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const pickCv = (file: File | undefined) => {
    if (!file) return;
    if (!CV_TYPES.some((t) => file.name.toLowerCase().endsWith(t))) return setErrors((p) => ({ ...p, cv: c.cvWrongType }));
    if (file.size > MAX_CV) return setErrors((p) => ({ ...p, cv: c.cvTooLarge }));
    setErrors((p) => {
      const next = { ...p };
      delete next.cv;
      return next;
    });
    setCv(file);
  };

  const submit = async () => {
    setFormError(null);
    if (!validate()) return;
    setPending(true);
    const fields = { ...v, expectedSalary: toRupiahInt(v.expectedSalary) ?? "" };
    try {
      const res = cv
        ? await uploadWithProgress<ApplyResponse>(`/api/public-erp/api/recruitment/public/jobs/${encodeURIComponent(slug)}/apply`, cv, fields, setProgress)
        : await portalPost<ApplyResponse>(`/v1/recruitment/public/jobs/${encodeURIComponent(slug)}/apply`, {
            ...fields,
            expectedSalary: fields.expectedSalary || undefined,
          });
      setDone(res.message || c.successDefault);
    } catch (e) {
      setFormError(errorMessage(e));
    } finally {
      setPending(false);
      setProgress(null);
    }
  };

  const field = (k: keyof typeof EMPTY, label: string, props: React.ComponentProps<typeof Input> = {}, required = false) => (
    <FormField label={label} htmlFor={`apply-${k}`} error={errors[k]} required={required}>
      <Input id={`apply-${k}`} value={v[k]} onChange={(e) => set(k, e.target.value)} aria-invalid={Boolean(errors[k]) || undefined} {...props} />
    </FormField>
  );

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        void submit();
      }}
    >
      <div className="sm:col-span-2">{field("fullName", c.fullName, { autoComplete: "name" }, true)}</div>
      {field("email", c.email, { type: "email", autoComplete: "email", inputMode: "email" }, true)}
      {field("phone", c.phone, { type: "tel", autoComplete: "tel", inputMode: "tel" }, true)}
      {field("domicile", c.domicile, { autoComplete: "address-level2" }, true)}
      {field("lastEducation", c.lastEducation)}
      <div className="sm:col-span-2">{field("lastExperience", c.lastExperience)}</div>
      <FormField label={c.availability} htmlFor="apply-availability">
        <select
          id="apply-availability"
          value={v.availability}
          onChange={(e) => set("availability", e.target.value)}
          className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
        >
          <option value="">{c.pick}</option>
          {Object.entries(AVAILABILITY).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </select>
      </FormField>
      <FormField label={c.expectedSalary} htmlFor="apply-salary">
        <MoneyInput id="apply-salary" value={v.expectedSalary} onChange={(x) => set("expectedSalary", x)} />
      </FormField>
      <FormField label={c.source} htmlFor="apply-source" className="sm:col-span-2">
        <select
          id="apply-source"
          value={v.source}
          onChange={(e) => set("source", e.target.value)}
          className="h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
        >
          <option value="">{c.pick}</option>
          {SOURCES.map((s) => (
            <option key={s} value={s}>
              {SOURCE[s]}
            </option>
          ))}
        </select>
      </FormField>
      <FormField label={c.notes} htmlFor="apply-notes" className="sm:col-span-2">
        <Textarea id="apply-notes" rows={3} value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </FormField>
      <div className="grid gap-1.5 sm:col-span-2">
        <Label htmlFor="apply-cv">{c.cv}</Label>
        {cv ? (
          <div className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
            <FileUp className="size-4 text-muted-foreground" />
            <span className="min-w-0 flex-1 truncate">{cv.name}</span>
            <Button type="button" size="icon-sm" variant="ghost" onClick={() => setCv(null)} aria-label={c.removeCv} disabled={pending}>
              <X className="size-4" />
            </Button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => fileRef.current?.click()}
            className="flex flex-col items-center gap-1 rounded-xl border border-dashed px-4 py-5 text-sm hover:bg-accent/50"
          >
            <FileUp className="size-5 text-muted-foreground" />
            <span className="font-medium">{c.cvPick}</span>
            <span className="text-xs text-muted-foreground">{c.cvHint}</span>
          </button>
        )}
        <input
          ref={fileRef}
          id="apply-cv"
          type="file"
          hidden
          accept={CV_TYPES.join(",")}
          onChange={(e) => {
            pickCv(e.target.files?.[0]);
            e.target.value = "";
          }}
        />
        {errors.cv ? <p className="text-xs text-destructive">{errors.cv}</p> : null}
        {progress !== null ? <Progress value={progress} /> : null}
      </div>
      {formError ? (
        <p role="alert" className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive sm:col-span-2">
          {formError}
        </p>
      ) : null}
      <div className="flex flex-col gap-2 sm:col-span-2">
        <Button type="submit" size="lg" disabled={pending} className="w-full sm:w-auto sm:self-end">
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {pending ? c.sending : c.apply}
        </Button>
        <p className="text-xs text-muted-foreground sm:text-right">{c.consent}</p>
      </div>
    </form>
  );
}
