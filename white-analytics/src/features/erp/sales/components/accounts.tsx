"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Star, Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { SearchFilter } from "@/features/erp/shared/url-filters";
import { useErpAction } from "@/features/erp/shared/client";
import {
  deleteContactAction,
  saveAccountAction,
  saveContactAction,
  type AccountValues,
  type ContactValues,
} from "../actions";
import { nameOf, type SalesCaps } from "../lib";
import { ORG_TYPE, opts, ss } from "../strings";
import type { Account, Contact } from "../types";
import { ConfirmAction, type FieldErrors, Fld, FormDialog, FormFooter, type Option, Picker, SimpleSelect } from "./ui";

const a = ss.accounts;
const c = ss.contacts;

export function AccountsTable({ accounts, owners, caps }: { accounts: Account[]; owners: Option[]; caps: SalesCaps }) {
  const router = useRouter();
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  const columns: ColumnDef<Account, unknown>[] = [
    {
      accessorKey: "name",
      header: a.name,
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    { accessorKey: "industry", header: a.industry },
    { accessorKey: "city", header: ss.leads.city },
    { accessorKey: "phone", header: a.phone },
    { accessorKey: "email", header: a.email },
    { id: "owner", header: ss.owner, accessorFn: (r) => nameOf(names, r.ownerUserId) },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={accounts}
      getRowId={(r) => r.id}
      onRowClick={(row) => router.push(`/erp/sales/leads/accounts/${row.original.id}`)}
      exportName="akun"
      toolbar={
        <>
          <SearchFilter param="aq" placeholder={a.searchPlaceholder} />
          {caps.write ? (
            <AccountFormDialog
              account={null}
              owners={owners}
              trigger={
                <Button>
                  <Plus className="size-4" /> {a.add}
                </Button>
              }
            />
          ) : null}
        </>
      }
    />
  );
}

export function AccountFormDialog({
  account,
  owners,
  trigger,
}: {
  account: Account | null;
  owners: Option[];
  trigger: React.ReactNode;
}) {
  return (
    <FormDialog trigger={trigger} title={account ? a.editTitle : a.addTitle} size="lg">
      {(close) => <AccountForm account={account} owners={owners} close={close} />}
    </FormDialog>
  );
}

function AccountForm({ account, owners, close }: { account: Account | null; owners: Option[]; close: () => void }) {
  const router = useRouter();
  const [v, setV] = React.useState<AccountValues>({
    name: account?.name ?? "",
    accountType: account?.accountType ?? "corporate",
    industry: account?.industry ?? "",
    address: account?.address ?? "",
    city: account?.city ?? "",
    phone: account?.phone ?? "",
    email: account?.email ?? "",
    website: account?.website ?? "",
    npwp: account?.npwp ?? "",
    notes: account?.notes ?? "",
    ownerUserId: account?.ownerUserId ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (k: keyof AccountValues, val: string) => setV((p) => ({ ...p, [k]: val }));
  const text = (k: keyof AccountValues, label: string, cls?: string) => (
    <Fld label={label} htmlFor={`acc-${k}`} error={errors?.[k]} className={cls}>
      <Input id={`acc-${k}`} value={String(v[k] ?? "")} onChange={(e) => set(k, e.target.value)} />
    </Fld>
  );
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveAccountAction(account?.id ?? null, v), {
          success: ss.saved,
          onOk: (saved) => {
            close();
            if (!account) router.push(`/erp/sales/leads/accounts/${saved.id}`);
          },
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      {text("name", `${a.name} *`, "sm:col-span-2")}
      <Fld label={a.type} htmlFor="acc-type">
        <SimpleSelect id="acc-type" value={v.accountType ?? ""} onChange={(x) => set("accountType", x)} options={opts(ORG_TYPE)} />
      </Fld>
      {text("industry", a.industry)}
      {text("phone", a.phone)}
      {text("email", a.email)}
      {text("website", a.website)}
      {text("npwp", a.npwp)}
      {text("city", ss.leads.city)}
      <Fld label={ss.owner} htmlFor="acc-owner">
        <Picker id="acc-owner" value={v.ownerUserId ?? ""} onChange={(x) => set("ownerUserId", x)} options={owners} allowEmpty={ss.noOwner} />
      </Fld>
      {text("address", a.address, "sm:col-span-2")}
      <Fld label={ss.notes} htmlFor="acc-notes" className="sm:col-span-2">
        <Textarea id="acc-notes" rows={2} value={v.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function ContactsTable({
  contacts,
  accounts,
  caps,
  accountId,
  compact,
}: {
  contacts: Contact[];
  accounts: Option[];
  caps: SalesCaps;
  /** fixes the account for new contacts (account detail page) */
  accountId?: string;
  /** account detail: no search or export, outline add button */
  compact?: boolean;
}) {
  const accountNames = Object.fromEntries(accounts.map((o) => [o.value, o.label]));
  const columns: ColumnDef<Contact, unknown>[] = [
    {
      accessorKey: "name",
      header: c.name,
      cell: ({ row }) => (
        <span className="flex items-center gap-1.5 font-medium">
          {row.original.name}
          {row.original.isPrimary ? <Star className="size-3 fill-amber-400 text-amber-400" aria-label={c.primary} /> : null}
        </span>
      ),
    },
    { accessorKey: "title", header: c.title },
    { accessorKey: "phone", header: a.phone },
    { accessorKey: "email", header: a.email },
    ...(accountId ? [] : [{ id: "account", header: c.account, accessorFn: (r: Contact) => nameOf(accountNames, r.accountId) }]),
    ...(caps.write
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: Contact } }) => (
              <div className="flex justify-end gap-1">
                <ContactFormDialog
                  contact={row.original}
                  accounts={accounts}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
                <ConfirmAction
                  title={c.deleteTitle}
                  action={() => deleteContactAction(row.original.id)}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </div>
            ),
          } satisfies ColumnDef<Contact, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={contacts}
      getRowId={(r) => r.id}
      exportName={compact ? undefined : "kontak"}
      toolbar={
        <>
          {compact ? null : <SearchFilter param="cq" placeholder={c.searchPlaceholder} />}
          {caps.write ? (
            <ContactFormDialog
              contact={null}
              accounts={accounts}
              accountId={accountId}
              trigger={
                <Button variant={compact ? "outline" : "default"}>
                  <Plus className="size-3.5" /> {c.add}
                </Button>
              }
            />
          ) : null}
        </>
      }
    />
  );
}

function ContactFormDialog({
  contact,
  accounts,
  accountId,
  trigger,
}: {
  contact: Contact | null;
  accounts: Option[];
  accountId?: string;
  trigger: React.ReactNode;
}) {
  return (
    <FormDialog trigger={trigger} title={contact ? c.editTitle : c.addTitle}>
      {(close) => <ContactForm contact={contact} accounts={accounts} accountId={accountId} close={close} />}
    </FormDialog>
  );
}

function ContactForm({
  contact,
  accounts,
  accountId,
  close,
}: {
  contact: Contact | null;
  accounts: Option[];
  accountId?: string;
  close: () => void;
}) {
  const [v, setV] = React.useState<ContactValues>({
    accountId: contact?.accountId ?? accountId ?? "",
    name: contact?.name ?? "",
    title: contact?.title ?? "",
    phone: contact?.phone ?? "",
    email: contact?.email ?? "",
    isPrimary: contact?.isPrimary ?? false,
    notes: contact?.notes ?? "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof ContactValues>(k: K, val: ContactValues[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveContactAction(contact?.id ?? null, v), {
          success: ss.saved,
          onOk: close,
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={`${c.name} *`} htmlFor="ct-name" error={errors?.name} className="sm:col-span-2">
        <Input id="ct-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={c.title} htmlFor="ct-title">
        <Input id="ct-title" value={v.title ?? ""} onChange={(e) => set("title", e.target.value)} />
      </Fld>
      <Fld label={a.phone} htmlFor="ct-phone">
        <Input id="ct-phone" value={v.phone ?? ""} onChange={(e) => set("phone", e.target.value)} />
      </Fld>
      <Fld label={a.email} htmlFor="ct-email" error={errors?.email}>
        <Input id="ct-email" type="email" value={v.email ?? ""} onChange={(e) => set("email", e.target.value)} />
      </Fld>
      <Fld label={c.account} htmlFor="ct-account">
        <Picker
          id="ct-account"
          value={v.accountId ?? ""}
          onChange={(x) => set("accountId", x)}
          options={accounts}
          allowEmpty={ss.dash}
          disabled={!!accountId}
        />
      </Fld>
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox checked={!!v.isPrimary} onCheckedChange={(x) => set("isPrimary", x === true)} /> {c.primary}
      </label>
      <Fld label={ss.notes} htmlFor="ct-notes" className="sm:col-span-2">
        <Textarea id="ct-notes" rows={2} value={v.notes ?? ""} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
