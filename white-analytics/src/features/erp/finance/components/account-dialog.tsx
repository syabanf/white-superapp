"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { FormField } from "@/features/erp/shared/form-field";
import { SearchableSelect } from "@/features/erp/shared/searchable-select";
import { saveAccountAction } from "../actions";
import type { Account, AccountType, CashFlowCategory } from "../types";
import { fs } from "../strings";

const a = fs.accounts;
const NONE = "__none__";

/** Candidate parents: not a level-4 leaf, not the account itself or its descendants. */
function parentCandidates(all: Account[], editingId?: string): Account[] {
  const excluded = new Set<string>();
  if (editingId) {
    const queue = [editingId];
    excluded.add(editingId);
    while (queue.length) {
      const cur = queue.shift()!;
      for (const row of all)
        if (row.parentId === cur && !excluded.has(row.id)) {
          excluded.add(row.id);
          queue.push(row.id);
        }
    }
  }
  return all.filter((p) => !excluded.has(p.id) && p.level < 4).sort((x, y) => x.code.localeCompare(y.code));
}

function Toggle({ id, label, hint, checked, onChange }: { id: string; label: string; hint?: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="flex items-start justify-between gap-3 rounded-lg border px-3 py-2.5">
      <div>
        <Label htmlFor={id}>{label}</Label>
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </div>
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
    </div>
  );
}

export function AccountDialog({
  account,
  parentId,
  accounts,
  accountTypes,
  onClose,
}: {
  account?: Account;
  parentId?: string;
  accounts: Account[];
  accountTypes: AccountType[];
  onClose: () => void;
}) {
  const parent = accounts.find((x) => x.id === (account?.parentId ?? parentId));
  const [form, setForm] = React.useState({
    code: account ? account.codeDisplay || account.code : "",
    name: account?.name ?? "",
    parentId: account?.parentId ?? parentId ?? "",
    accountTypeId: account?.accountTypeId ?? parent?.accountTypeId ?? accountTypes[0]?.id ?? "",
    isContra: account?.isContra ?? false,
    isCashBank: account?.isCashBank ?? false,
    cashFlowCategory: (account?.cashFlowCategory ?? "") as CashFlowCategory,
    description: account?.description ?? "",
    isActive: account?.isActive ?? true,
  });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const set = <K extends keyof typeof form>(k: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [k]: v }));
  const candidates = parentCandidates(accounts, account?.id);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    void run(() => saveAccountAction(account?.id ?? null, { ...form, parentId: form.parentId || null }), {
      success: fs.saved,
      onOk: onClose,
      onFail: (r) => setErrors(r.fieldErrors ?? {}),
    });
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-xl">
        <form onSubmit={submit} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{account ? a.editTitle : a.newTitle}</DialogTitle>
            {parent ? <DialogDescription>{`${a.parent}: ${parent.codeDisplay} ${parent.name}`}</DialogDescription> : null}
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-[180px_1fr]">
            <FormField label={fs.code} htmlFor="acc-code" error={errors.code} hint={a.codeHint} required>
              <Input id="acc-code" value={form.code} onChange={(e) => set("code", e.target.value)} className="font-mono" />
            </FormField>
            <FormField label={fs.name} htmlFor="acc-name" error={errors.name} required>
              <Input id="acc-name" value={form.name} onChange={(e) => set("name", e.target.value)} />
            </FormField>
          </div>
          <FormField label={a.parent} error={errors.parentId}>
            <SearchableSelect
              options={[
                { value: NONE, label: a.noParent },
                ...candidates.map((p) => ({ value: p.id, label: p.name, hint: p.codeDisplay || p.code })),
              ]}
              value={form.parentId || NONE}
              onChange={(v) => {
                const nextParent = accounts.find((x) => x.id === v);
                setForm((f) => ({
                  ...f,
                  parentId: v === NONE ? "" : v,
                  accountTypeId: nextParent?.accountTypeId ?? f.accountTypeId,
                }));
              }}
              searchPlaceholder={fs.searchAccount}
            />
          </FormField>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={a.accountType} error={errors.accountTypeId} required>
              <Select value={form.accountTypeId} onValueChange={(v) => set("accountTypeId", v)}>
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={a.accountType} />
                </SelectTrigger>
                <SelectContent>
                  {accountTypes.map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      {t.code} · {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label={a.cashFlowCategory}>
              <Select value={form.cashFlowCategory || NONE} onValueChange={(v) => set("cashFlowCategory", (v === NONE ? "" : v) as CashFlowCategory)}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>{a.noCategory}</SelectItem>
                  {Object.entries(fs.cashFlowCategory).map(([v, label]) => (
                    <SelectItem key={v} value={v}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            <Toggle id="acc-contra" label={a.contra} hint={a.contraHint} checked={form.isContra} onChange={(v) => set("isContra", v)} />
            <Toggle id="acc-cash" label={a.cashBank} hint={a.cashBankHint} checked={form.isCashBank} onChange={(v) => set("isCashBank", v)} />
            <Toggle id="acc-active" label={fs.active} checked={form.isActive} onChange={(v) => set("isActive", v)} />
          </div>
          <FormField label={fs.description} htmlFor="acc-desc">
            <Textarea id="acc-desc" rows={2} value={form.description} onChange={(e) => set("description", e.target.value)} />
          </FormField>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose} disabled={pending}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
