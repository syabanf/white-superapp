"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { ArrowRightLeft, CheckCheck, FileOutput, Pencil, Plus, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ActionButton, ConfirmAction, Fld, FormDialog, FormFooter, ReasonDialog, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { actOnDocumentAction, convertDocumentAction, deleteDocumentAction, issueDocumentAction, saveDocumentAction, type DocumentValues } from "../actions";
import { fs } from "../strings";
import { previewDocument } from "../pricing";
import type { ChargeType, DocumentItem, DocumentKind, FinanceDocument, PostableAccount, TaxCode, Vendor } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.documents;

export function DocumentsTable({ documents, kind, page }: { documents: FinanceDocument[]; kind: DocumentKind; page: ServerPage }) {
  const router = useRouter();
  const columns: ColumnDef<FinanceDocument, unknown>[] = [
    {
      accessorKey: "number",
      header: s.number,
      cell: ({ row }) => (
        <Link href={`/erp/finance/documents/${row.original.id}`} className="font-mono text-xs font-medium hover:underline" onClick={(e) => e.stopPropagation()}>
          {row.original.number}
        </Link>
      ),
    },
    { accessorKey: "title", header: s.subject, cell: ({ row }) => <span className="line-clamp-2 max-w-sm">{row.original.title}</span> },
    { accessorKey: "partyName", header: s.party, cell: ({ row }) => row.original.partyName || "–" },
    { accessorKey: "issueDate", header: s.date, cell: ({ row }) => formatDate(row.original.issueDate) },
    { accessorKey: "totalAmount", header: s.total, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.totalAmount} /> },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} /> },
  ];
  const statuses = kind === "QUOTATION" ? ["DRAFT", "SENT", "ACCEPTED", "REJECTED", "CONVERTED"] : kind === "INVOICE" ? ["DRAFT", "ISSUED"] : ["DRAFT", "SUBMITTED", "APPROVED", "REJECTED", "CONVERTED"];
  return (
    <ErpDataTable
      columns={columns}
      data={documents}
      pagination={page}
      getRowId={(d) => d.id}
      emptyMessage={s.empty}
      onRowClick={(r) => router.push(`/erp/finance/documents/${r.original.id}`)}
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={statuses.map((value) => ({ value, label: s.statuses[value] }))} />
          <ResetFilters params={["q", "status"]} />
        </>
      }
    />
  );
}

const blankItem = (): DocumentItem => ({ description: "", unit: "", qty: "1", unitPrice: "" });

/** The taxes and charges a document can offer, from the master data. */
export type TaxOptions = { taxes: TaxCode[]; charges: ChargeType[] };

const fitsSide = (scope: string, side: string) => scope === "BOTH" || (scope === "SALE") === (side === "B2B");

function initial(kind: DocumentKind, options: TaxOptions, d?: FinanceDocument): DocumentValues {
  if (d) {
    const { id, kind: k, side, vendorId, partyName, picName, phone, email, address, title, issueDate, dueDate, externalNumber, notes, items, discountPercent } = d;
    return {
      id, kind: k, side, vendorId, partyName, picName, phone, email, address, title, issueDate, dueDate, externalNumber, notes, items, discountPercent,
      taxIds: (d.taxes ?? []).map((t) => t.taxId),
      charges: (d.charges ?? []).map((c) => ({ chargeId: c.chargeId, value: c.value })),
    };
  }
  const supplier = kind === "PR" || kind === "PO";
  return { kind, side: supplier ? "SUPPLIER" : "B2B", vendorId: "", partyName: "", picName: "", phone: "", email: "", address: "", title: "", issueDate: toIsoDate(new Date()), dueDate: "", externalNumber: "", notes: "", items: [blankItem()], discountPercent: "0", taxIds: options.taxes.filter((t) => (supplier ? t.isDefaultPurchase : t.isDefaultSale)).map((t) => t.id), charges: [] };
}

function DocumentForm({ kind, document, vendors, options, close }: { kind: DocumentKind; document?: FinanceDocument; vendors: Vendor[]; options: TaxOptions; close: () => void }) {
  const router = useRouter();
  const [v, setV] = React.useState<DocumentValues>(() => initial(kind, options, document));
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof DocumentValues>(k: K, x: DocumentValues[K]) => setV((p) => ({ ...p, [k]: x }));
  const setItem = (i: number, patch: Partial<DocumentItem>) => set("items", v.items.map((it, n) => (n === i ? { ...it, ...patch } : it)));
  const text = (k: "picName" | "phone" | "email" | "address" | "externalNumber", label: string, cls?: string) => (
    <Fld label={label} htmlFor={`d-${k}`} className={cls}>
      <Input id={`d-${k}`} value={v[k]} onChange={(e) => set(k, e.target.value)} />
    </Fld>
  );
  const usesVendor = v.side === "SUPPLIER" && kind !== "PR";
  const manualParty = kind !== "PR" && !v.vendorId;
  const taxChoices = options.taxes.filter((t) => fitsSide(t.scope, v.side) || v.taxIds.includes(t.id));
  const chargeChoices = options.charges.filter((c) => fitsSide(c.scope, v.side) && !v.charges.some((x) => x.chargeId === c.id));
  const preview = previewDocument(v, options);
  const toggleTax = (id: string) => set("taxIds", v.taxIds.includes(id) ? v.taxIds.filter((x) => x !== id) : [...v.taxIds, id]);
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveDocumentAction(v), {
          success: s.saved,
          onOk: (saved) => {
            close();
            if (!document) router.push(`/erp/finance/documents/${saved.id}`);
          },
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={`${s.subject} *`} htmlFor="d-title" error={errors?.title} className="sm:col-span-2">
        <Input id="d-title" value={v.title} onChange={(e) => set("title", e.target.value)} />
      </Fld>
      {kind === "INVOICE" || kind === "QUOTATION" ? (
        <Fld label={s.side} htmlFor="d-side">
          <SimpleSelect id="d-side" value={v.side} onChange={(x) => setV((p) => ({ ...p, side: x, vendorId: x === "B2B" ? "" : p.vendorId }))} options={Object.entries(s.sides).map(([value, label]) => ({ value, label }))} />
        </Fld>
      ) : null}
      {usesVendor ? (
        <Fld label={s.vendor} htmlFor="d-vendor" error={errors?.vendor_id}>
          <SimpleSelect id="d-vendor" value={v.vendorId} onChange={(x) => set("vendorId", x)} emptyLabel={s.noVendor} options={vendors.filter((x) => x.isActive).map((x) => ({ value: x.id, label: x.name }))} />
        </Fld>
      ) : null}
      {manualParty ? (
        <>
          <Fld label={`${s.party} *`} htmlFor="d-party" error={errors?.party_name}>
            <Input id="d-party" value={v.partyName} onChange={(e) => set("partyName", e.target.value)} />
          </Fld>
          {text("picName", s.pic)}
          {text("phone", s.phone)}
          {text("email", s.email)}
          {text("address", s.address, "sm:col-span-2")}
        </>
      ) : null}
      <Fld label={s.date} htmlFor="d-date" error={errors?.issue_date}>
        <DatePicker id="d-date" value={v.issueDate} onChange={(x) => set("issueDate", x)} />
      </Fld>
      <Fld label={s.due} htmlFor="d-due" error={errors?.due_date}>
        <DatePicker id="d-due" value={v.dueDate} onChange={(x) => set("dueDate", x)} clearable />
      </Fld>
      {kind === "INVOICE" ? text("externalNumber", s.externalNumber, "sm:col-span-2") : null}

      <div className="flex flex-col gap-2 sm:col-span-2">
        <span className="text-sm font-medium">{s.items}</span>
        {errors?.items ? <p className="text-xs text-negative">{errors.items[0]}</p> : null}
        {v.items.map((it, i) => (
          <div key={i} className="grid grid-cols-[1fr_auto] items-start gap-2 rounded-lg border p-2 sm:grid-cols-[minmax(0,1fr)_4.5rem_5rem_8rem_auto]">
            <Input aria-label={s.description} placeholder={s.description} value={it.description} onChange={(e) => setItem(i, { description: e.target.value })} className="col-span-2 sm:col-span-1" />
            <Input aria-label={s.qty} placeholder={s.qty} inputMode="decimal" value={it.qty} onChange={(e) => setItem(i, { qty: e.target.value.replace(",", ".") })} />
            <Input aria-label={s.unit} placeholder={s.unit} value={it.unit} onChange={(e) => setItem(i, { unit: e.target.value })} />
            <MoneyInput aria-label={s.price} placeholder={s.price} value={it.unitPrice} onChange={(x) => setItem(i, { unitPrice: x })} />
            <Button type="button" size="icon-xs" variant="ghost" aria-label={s.removeItem} disabled={v.items.length === 1} onClick={() => set("items", v.items.filter((_, n) => n !== i))}>
              <X className="size-3.5" />
            </Button>
          </div>
        ))}
        <Button type="button" size="xs" variant="outline" className="w-fit" onClick={() => set("items", [...v.items, blankItem()])}>
          <Plus className="size-3.5" /> {s.addItem}
        </Button>
      </div>
      <Fld label={s.discount} htmlFor="d-disc" error={errors?.discount_percent}>
        <Input id="d-disc" inputMode="decimal" value={v.discountPercent} onChange={(e) => set("discountPercent", e.target.value.replace(",", "."))} />
      </Fld>
      <fieldset className="flex flex-col gap-2 sm:col-span-2">
        <legend className="mb-1 text-sm font-medium">{s.taxes}</legend>
        {errors?.tax_ids ? <p className="text-xs text-negative">{errors.tax_ids[0]}</p> : null}
        <div className="flex flex-wrap gap-x-4 gap-y-1.5">
          {taxChoices.map((t) => (
            <label key={t.id} className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={v.taxIds.includes(t.id)} onChange={() => toggleTax(t.id)} />
              {t.name}
            </label>
          ))}
        </div>
      </fieldset>
      <div className="flex flex-col gap-2 sm:col-span-2">
        <span className="text-sm font-medium">{s.charges}</span>
        {errors?.charges ? <p className="text-xs text-negative">{errors.charges[0]}</p> : null}
        {v.charges.map((c, i) => {
          const master = options.charges.find((x) => x.id === c.chargeId);
          return (
            <div key={c.chargeId} className="flex items-center gap-2">
              <span className="min-w-0 flex-1 truncate text-sm">{master?.name ?? c.chargeId}</span>
              {master?.computation === "PERCENT" ? (
                <Input aria-label={s.chargeValue} className="w-24" inputMode="decimal" placeholder="%" value={c.value} onChange={(e) => set("charges", v.charges.map((x, n) => (n === i ? { ...x, value: e.target.value.replace(",", ".") } : x)))} />
              ) : (
                <MoneyInput aria-label={s.chargeValue} className="w-40" value={c.value} onChange={(x) => set("charges", v.charges.map((y, n) => (n === i ? { ...y, value: x } : y)))} />
              )}
              <Button type="button" size="icon-xs" variant="ghost" aria-label={s.removeItem} onClick={() => set("charges", v.charges.filter((_, n) => n !== i))}>
                <X className="size-3.5" />
              </Button>
            </div>
          );
        })}
        {chargeChoices.length ? (
          <SimpleSelect
            id="d-add-charge"
            value=""
            placeholder={s.addCharge}
            onChange={(id) => {
              const master = options.charges.find((x) => x.id === id);
              if (master) set("charges", [...v.charges, { chargeId: id, value: parseMoney(master.value) > 0 ? master.value : "" }]);
            }}
            options={chargeChoices.map((c) => ({ value: c.id, label: c.name }))}
          />
        ) : null}
      </div>
      <dl className="flex flex-col gap-0.5 text-sm sm:col-span-2">
        <div className="flex justify-end gap-3 font-semibold"><dt>{s.total}</dt><dd>{formatRupiah(preview.total)}</dd></div>
        {preview.withholding > 0 ? (
          <div className="flex justify-end gap-3"><dt>{s.net}</dt><dd>{formatRupiah(preview.total - preview.withholding)}</dd></div>
        ) : null}
        <p className="text-right text-xs text-muted-foreground">{s.previewNote}</p>
      </dl>
      <Fld label={s.notes} htmlFor="d-notes" className="sm:col-span-2">
        <Textarea id="d-notes" rows={2} value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function DocumentDialog({ kind, document, vendors, options }: { kind: DocumentKind; document?: FinanceDocument; vendors: Vendor[]; options: TaxOptions }) {
  return (
    <FormDialog
      title={document ? `${s.edit} ${document.number}` : s.newKind[kind]}
      size="xl"
      trigger={
        document ? (
          <Button size="xs" variant="outline">
            <Pencil className="size-3.5" /> {s.edit}
          </Button>
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {s.newKind[kind]}
          </Button>
        )
      }
    >
      {(close) => <DocumentForm kind={kind} document={document} vendors={vendors} options={options} close={close} />}
    </FormDialog>
  );
}

function ConvertDialog({ doc, vendors }: { doc: FinanceDocument; vendors: Vendor[] }) {
  const router = useRouter();
  const [vendorId, setVendorId] = React.useState("");
  const { pending, run } = useErpAction();
  const go = () => void run(() => convertDocumentAction(doc.id, vendorId), { success: s.converted, onOk: (d) => router.push(`/erp/finance/documents/${d.id}`) });
  if (doc.kind === "QUOTATION") {
    return (
      <Button size="xs" disabled={pending} onClick={go}>
        <ArrowRightLeft className="size-3.5" /> {s.convertInvoice}
      </Button>
    );
  }
  return (
    <FormDialog
      title={s.convertPo}
      trigger={
        <Button size="xs">
          <ArrowRightLeft className="size-3.5" /> {s.convertPo}
        </Button>
      }
    >
      {(close) => (
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            go();
            close();
          }}
        >
          <Fld label={`${s.vendor} *`} htmlFor="cv-vendor">
            <SimpleSelect id="cv-vendor" value={vendorId} onChange={setVendorId} options={vendors.filter((x) => x.isActive).map((x) => ({ value: x.id, label: x.name }))} />
          </Fld>
          <FormFooter onCancel={close} pending={pending} submitLabel={s.convertPo} />
        </form>
      )}
    </FormDialog>
  );
}

function IssueDialog({ doc, accounts }: { doc: FinanceDocument; accounts: PostableAccount[] }) {
  const payable = doc.side === "SUPPLIER";
  return (
    <FormDialog
      title={s.issue}
      description={payable ? s.issueHintAP : s.issueHintAR}
      trigger={
        <Button size="xs">
          <FileOutput className="size-3.5" /> {s.issue}
        </Button>
      }
    >
      {(close) => <IssueForm doc={doc} accounts={accounts.filter((a) => (payable ? a.accountTypeCode === "EXPENSE" || a.accountTypeCode === "COGS" : a.accountTypeCode === "REVENUE"))} close={close} />}
    </FormDialog>
  );
}

function IssueForm({ doc, accounts, close }: { doc: FinanceDocument; accounts: PostableAccount[]; close: () => void }) {
  const [accountId, setAccountId] = React.useState(doc.accountId);
  const [dueDate, setDueDate] = React.useState(doc.dueDate);
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => issueDocumentAction(doc.id, accountId, dueDate), { success: s.issued, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.account} *`} htmlFor="is-account" error={errors?.account_id}>
        <AccountPicker id="is-account" accounts={accounts} value={accountId} onChange={setAccountId} />
      </Fld>
      <Fld label={s.due} htmlFor="is-due" error={errors?.due_date}>
        <DatePicker id="is-due" value={dueDate} onChange={setDueDate} clearable />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={s.issue} />
    </form>
  );
}

/** What a person can do now, from the document's kind and status. */
export function DocumentActions({ doc, userId, isAdmin, canWrite, canApprove, vendors, accounts, options }: { doc: FinanceDocument; userId: string; isAdmin: boolean; canWrite: boolean; canApprove: boolean; vendors: Vendor[]; accounts: PostableAccount[]; options: TaxOptions }) {
  const router = useRouter();
  const act = (action: string, variant?: "outline") => (
    <ActionButton key={action} size="xs" variant={variant} action={() => actOnDocumentAction(doc.id, action)} success={s.actionDone}>
      <CheckCheck className="size-3.5" /> {s.actions[action]}
    </ActionButton>
  );
  const reason = (action: string) => (
    <ReasonDialog
      key={action}
      title={s.actions[action]}
      label={s.rejectTitle}
      submitLabel={s.actions[action]}
      success={s.actionDone}
      onSubmit={(text) => actOnDocumentAction(doc.id, action, text)}
      trigger={
        <Button size="xs" variant="ghost">
          <X className="size-3.5" /> {s.actions[action]}
        </Button>
      }
    />
  );
  const purchase = doc.kind === "PR" || doc.kind === "PO";
  const own = doc.createdBy === userId && !isAdmin;
  const out: React.ReactNode[] = [];
  if (canWrite && doc.status === "DRAFT") {
    out.push(<DocumentDialog key="edit" kind={doc.kind} document={doc} vendors={vendors} options={options} />);
    if (purchase) out.push(act("submit"));
    if (doc.kind === "QUOTATION") out.push(act("send"));
    if (doc.kind === "INVOICE") out.push(<IssueDialog key="issue" doc={doc} accounts={accounts} />);
    out.push(
      <ConfirmAction
        key="delete"
        title={s.deleteTitle}
        success={s.deleted}
        action={() => deleteDocumentAction(doc.id)}
        onDone={() => router.back()}
        trigger={
          <Button size="icon-xs" variant="ghost" aria-label={s.deleteTitle}>
            <Trash2 className="size-3.5" />
          </Button>
        }
      />,
    );
  }
  if (purchase && doc.status === "SUBMITTED" && canApprove && !own) out.push(act("approve"), reason("reject"));
  if (canWrite && doc.kind === "QUOTATION" && doc.status === "SENT") out.push(act("accept"), reason("decline"));
  if (canWrite && doc.status === "REJECTED") out.push(act("reopen", "outline"));
  if (canWrite && doc.kind === "PR" && doc.status === "APPROVED") out.push(<ConvertDialog key="convert" doc={doc} vendors={vendors} />);
  if (canWrite && doc.kind === "QUOTATION" && doc.status === "ACCEPTED") out.push(<ConvertDialog key="convert" doc={doc} vendors={vendors} />);
  if (canWrite && doc.kind === "PO" && doc.status === "APPROVED") out.push(<IssueDialog key="issue" doc={doc} accounts={accounts} />);
  return (
    <div className="flex flex-wrap items-center gap-1">
      {out}
      <DownloadButton href={`/api/erp/api/finance/documents/${encodeURIComponent(doc.id)}/pdf`} fileName={`${doc.number}.pdf`} size="xs">
        {s.pdf}
      </DownloadButton>
    </div>
  );
}
