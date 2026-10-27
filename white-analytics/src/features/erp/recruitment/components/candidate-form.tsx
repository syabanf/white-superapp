"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import type { z } from "zod";
import { Pencil, UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { SearchableSelect, type SearchOption } from "@/features/erp/shared/searchable-select";
import { saveCandidateAction } from "../actions";
import { toRupiahInt } from "../lib";
import type { candidateSchema } from "../schemas";
import { AVAILABILITY, CANDIDATE_STATUS, SOURCE, rs } from "../strings";
import type { Candidate } from "../types";

type Values = z.output<typeof candidateSchema>;

const c = rs.candidates;
const opts = (m: Record<string, string>) => Object.entries(m).map(([value, label]) => ({ value, label }));

export function NewCandidateButton({ jobs }: { jobs: SearchOption[] }) {
  return (
    <FormDialog
      title={c.new}
      wide
      trigger={
        <Button>
          <UserPlus className="size-4" /> {c.new}
        </Button>
      }
    >
      {(close) => <CandidateForm jobs={jobs} close={close} />}
    </FormDialog>
  );
}

export function EditCandidateButton({ candidate, jobs }: { candidate: Candidate; jobs: SearchOption[] }) {
  return (
    <FormDialog
      title={c.editTitle}
      wide
      trigger={
        <Button variant="outline">
          <Pencil className="size-4" /> {rs.edit}
        </Button>
      }
    >
      {(close) => <CandidateForm jobs={jobs} close={close} candidate={candidate} />}
    </FormDialog>
  );
}

function CandidateForm({ jobs, close, candidate }: { jobs: SearchOption[]; close: () => void; candidate?: Candidate }) {
  const router = useRouter();
  const [v, setV] = React.useState<Values>({
    fullName: candidate?.fullName ?? "",
    email: candidate?.email ?? "",
    phone: candidate?.phone ?? "",
    domicile: candidate?.domicile ?? "",
    source: candidate?.source ?? "walk_in",
    jobOpeningId: candidate?.jobOpeningId ?? "",
    status: "applied",
    notes: candidate?.notes ?? "",
    lastExperience: candidate?.lastExperience ?? "",
    lastEducation: candidate?.lastEducation ?? "",
    availability: candidate?.availability ?? "",
    expectedSalary: candidate?.expectedSalary ?? "",
  });
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  const set = <K extends keyof Values>(k: K, val: Values[K]) => setV((p) => ({ ...p, [k]: val }));

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveCandidateAction(candidate?.id ?? null, v), {
          success: candidate ? c.updated : c.created,
          onOk: (data) => {
            close();
            if (!candidate) router.push(`/erp/hr/recruitment/candidates/${data.id}`);
          },
          onFail,
        });
      }}
    >
      <Fld label={c.name} htmlFor="cand-name" error={errors?.fullName} className="sm:col-span-2">
        <Input id="cand-name" value={v.fullName} onChange={(e) => set("fullName", e.target.value)} autoFocus />
      </Fld>
      <Fld label={c.email} htmlFor="cand-email" error={errors?.email}>
        <Input id="cand-email" type="email" value={v.email} onChange={(e) => set("email", e.target.value)} />
      </Fld>
      <Fld label={c.phone} htmlFor="cand-phone" error={errors?.phone}>
        <Input id="cand-phone" type="tel" value={v.phone} onChange={(e) => set("phone", e.target.value)} />
      </Fld>
      <Fld label={c.domicile} htmlFor="cand-dom" error={errors?.domicile}>
        <Input id="cand-dom" value={v.domicile} onChange={(e) => set("domicile", e.target.value)} />
      </Fld>
      <Fld label={c.job} htmlFor="cand-job" error={errors?.jobOpeningId}>
        <SearchableSelect id="cand-job" options={jobs} value={v.jobOpeningId} onChange={(x) => set("jobOpeningId", x)} placeholder={c.noJob} clearable />
      </Fld>
      <Fld label={c.source} htmlFor="cand-src" error={errors?.source}>
        <SimpleSelect id="cand-src" value={v.source} onChange={(x) => set("source", x)} options={opts(SOURCE)} />
      </Fld>
      {candidate ? null : (
        <Fld label={c.initialStatus} htmlFor="cand-status">
          <SimpleSelect
            id="cand-status"
            value={v.status}
            onChange={(x) => set("status", x as Values["status"])}
            options={["applied", "screening"].map((s) => ({ value: s, label: CANDIDATE_STATUS[s] }))}
          />
        </Fld>
      )}
      <Fld label={c.lastExperience} htmlFor="cand-exp" error={errors?.lastExperience}>
        <Input id="cand-exp" value={v.lastExperience} onChange={(e) => set("lastExperience", e.target.value)} />
      </Fld>
      <Fld label={c.lastEducation} htmlFor="cand-edu" error={errors?.lastEducation}>
        <Input id="cand-edu" value={v.lastEducation} onChange={(e) => set("lastEducation", e.target.value)} />
      </Fld>
      <Fld label={c.availability} htmlFor="cand-av" error={errors?.availability}>
        <SimpleSelect id="cand-av" value={v.availability} onChange={(x) => set("availability", x)} options={opts(AVAILABILITY)} emptyLabel={rs.unknown} />
      </Fld>
      <Fld label={c.expectedSalary} htmlFor="cand-sal" error={errors?.expectedSalary}>
        <MoneyInput id="cand-sal" value={v.expectedSalary} onChange={(x) => set("expectedSalary", toRupiahInt(x) ?? "")} />
      </Fld>
      <Fld label={c.notes} htmlFor="cand-notes" error={errors?.notes} className="sm:col-span-2">
        <Textarea id="cand-notes" rows={3} value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
