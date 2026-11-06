"use client";

import * as React from "react";
import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Fld, type FieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDateTime } from "@/lib/format";
import { clearMailAction, saveCompanyProfileAction, saveMailAction } from "./actions";
import type { CompanyProfile, MailStatus } from "./data";
import { tc } from "./strings";

export function CompanyForm({ profile }: { profile: CompanyProfile }) {
  const [v, setV] = React.useState(profile);
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<CompanyProfile>) => setV((x) => ({ ...x, ...p }));
  const field = (key: keyof CompanyProfile, label: string, hint?: string) => (
    <Fld label={label} htmlFor={`co-${key}`} error={errors?.[key]} hint={hint}>
      <Input id={`co-${key}`} value={v[key]} onChange={(e) => set({ [key]: e.target.value })} />
    </Fld>
  );
  return (
    <Card className="gap-4 p-(--card-pad)">
      <div>
        <h2 className="font-semibold">{tc.company.title}</h2>
        <p className="text-sm text-muted-foreground">{tc.company.desc}</p>
      </div>
      <form
        className="grid gap-4 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault();
          void run(() => saveCompanyProfileAction(v), {
            success: tc.saved,
            onOk: (saved) => {
              setV(saved);
              setErrors(undefined);
            },
            onFail: (r) => setErrors(r.fieldErrors),
          });
        }}
      >
        {field("legalName", tc.company.legalName)}
        {field("npwp", tc.company.npwp)}
        <Fld label={tc.company.address} htmlFor="co-address" className="sm:col-span-2">
          <Textarea id="co-address" rows={2} value={v.address} onChange={(e) => set({ address: e.target.value })} />
        </Fld>
        {field("email", tc.company.email)}
        {field("phone", tc.company.phone)}
        <h3 className="pt-2 text-sm font-semibold sm:col-span-2">{tc.company.bank}</h3>
        {field("bankName", tc.company.bankName, tc.company.bankNameHint)}
        {field("bankAccountNo", tc.company.accountNo)}
        {field("bankAccountName", tc.company.accountName, tc.company.accountNameHint)}
        <div className="rounded-xl bg-muted/50 p-3 text-sm sm:col-span-2">
          <p className="text-xs text-muted-foreground">{tc.company.preview}</p>
          <p>{profile.bankLine}</p>
        </div>
        <div className="flex justify-end sm:col-span-2">
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null} {tc.save}
          </Button>
        </div>
      </form>
    </Card>
  );
}

export function MailForm({ status }: { status: MailStatus }) {
  const [from, setFrom] = React.useState(status.from);
  const [apiKey, setApiKey] = React.useState("");
  const [check, setCheck] = React.useState<{ ok: boolean; message: string } | null>(null);
  const { pending, run } = useErpAction();
  return (
    <Card className="gap-4 p-(--card-pad)">
      <div>
        <h2 className="font-semibold">{tc.mail.title}</h2>
        <p className="text-sm text-muted-foreground">{tc.mail.desc}</p>
      </div>
      <p className="text-sm">
        {status.configured ? tc.mail.sources[status.source] : tc.mail.notSet}
        {status.verifiedAt ? ` · ${tc.mail.verifiedAt} ${formatDateTime(status.verifiedAt)}` : ""}
      </p>
      <form
        className="grid gap-4"
        onSubmit={(e) => {
          e.preventDefault();
          void run(() => saveMailAction({ from, apiKey }), {
            onOk: (r) => {
              setCheck(r);
              setApiKey("");
            },
          });
        }}
      >
        <Fld label={tc.mail.from} htmlFor="mail-from" hint={tc.mail.fromHint}>
          <Input id="mail-from" value={from} onChange={(e) => setFrom(e.target.value)} placeholder="Mixist <billing@mixist.id>" required />
        </Fld>
        <Fld label={tc.mail.key} htmlFor="mail-key" hint={status.secretMasked ? tc.mail.keyKept(status.secretMasked) : tc.mail.keyHint}>
          <Input id="mail-key" type="password" autoComplete="off" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="re_…" />
        </Fld>
        {check ? (
          <p className={check.ok ? "flex items-center gap-1.5 text-sm" : "flex items-center gap-1.5 text-sm text-destructive"}>
            {check.ok ? <CheckCircle2 className="size-4 text-(--status-good)" /> : <CircleAlert className="size-4" />}
            {check.message}
          </p>
        ) : null}
        <div className="flex flex-wrap justify-end gap-2">
          {status.source === "DB" ? (
            <Button type="button" variant="outline" disabled={pending} onClick={() => void run(clearMailAction, { success: tc.mail.cleared })}>
              {tc.mail.clear}
            </Button>
          ) : null}
          <Button type="submit" disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null} {tc.mail.saveAndTest}
          </Button>
        </div>
      </form>
    </Card>
  );
}
