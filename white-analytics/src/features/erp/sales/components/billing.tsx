"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { BadgeCheck, Ban, Plus, Trash2, Wallet } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import {
  cancelInvoiceAction,
  deletePaymentAction,
  issueInvoiceAction,
  recordPaymentAction,
  submitInvoiceAction,
  type PaymentValues,
} from "../actions";
import type { SalesCaps } from "../lib";
import { INVOICE_STATUS, PAYMENT_METHOD, PAYMENT_STATUS, lbl, opts, ss } from "../strings";
import type { Invoice, Payment, Quotation, TermProgress } from "../types";
import { ActionButton, ConfirmAction, type FieldErrors, Fld, FormDialog, FormFooter, ReasonDialog, SalesBadge, SimpleSelect } from "./ui";

const s = ss.invoices;

/** Issue (finance) and cancel buttons plus the PDF link for one invoice. */
function InvoiceRowActions({ inv, caps }: { inv: Invoice; caps: SalesCaps }) {
  return (
    <div className="flex justify-end gap-1">
      <DownloadButton href={`/api/erp/api/sales/invoices/${inv.id}/pdf`} fileName={`${inv.invoiceNumber}.pdf`} size="xs" variant="ghost">
        PDF
      </DownloadButton>
      {caps.finance && inv.status === "diajukan" ? (
        <ActionButton size="xs" action={() => issueInvoiceAction(inv.id)} success={s.issued}>
          <BadgeCheck className="size-3.5" /> {s.issue}
        </ActionButton>
      ) : null}
      {caps.finance && inv.status !== "batal" ? (
        <ReasonDialog
          title={s.cancelTitle}
          label={s.cancelReason}
          submitLabel={s.cancel}
          success={s.cancelled}
          onSubmit={(r) => cancelInvoiceAction(inv.id, r)}
          trigger={
            <Button size="xs" variant="ghost">
              <Ban className="size-3.5" /> {s.cancel}
            </Button>
          }
        />
      ) : null}
    </div>
  );
}

export function InvoicesTable({ invoices, caps, showDeal = true }: { invoices: Invoice[]; caps: SalesCaps; showDeal?: boolean }) {
  const columns: ColumnDef<Invoice, unknown>[] = [
    {
      accessorKey: "invoiceNumber",
      header: s.number,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium tabular">{row.original.invoiceNumber}</div>
          <div className="truncate text-xs text-muted-foreground">{row.original.label}</div>
        </div>
      ),
    },
    ...(showDeal
      ? [
          {
            accessorKey: "clientName",
            header: ss.quotations.client,
            cell: ({ row }: { row: { original: Invoice } }) => (
              <Link href={`/erp/sales/pipeline/${row.original.dealId}`} className="min-w-0 hover:underline" onClick={(e) => e.stopPropagation()}>
                <div className="truncate">{row.original.clientName}</div>
                <div className="truncate text-xs text-muted-foreground">{row.original.dealTitle}</div>
              </Link>
            ),
          } satisfies ColumnDef<Invoice, unknown>,
        ]
      : []),
    { accessorKey: "amount", header: s.amount, accessorFn: (i) => Number(i.amount), cell: ({ row }) => <Money value={row.original.amount} />, meta: { align: "right" } },
    { accessorKey: "paidTotal", header: s.paid, accessorFn: (i) => Number(i.paidTotal), cell: ({ row }) => <Money value={row.original.paidTotal} muteZero />, meta: { align: "right" } },
    { accessorKey: "dueDate", header: s.due, cell: ({ row }) => formatDate(row.original.dueDate || null) },
    {
      accessorKey: "status",
      header: s.status,
      cell: ({ row }) => (
        <span title={row.original.cancelReason || undefined}>
          <SalesBadge value={row.original.status} map={INVOICE_STATUS} />
        </span>
      ),
    },
    { id: "actions", header: "", enableSorting: false, cell: ({ row }) => <InvoiceRowActions inv={row.original} caps={caps} /> },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={invoices}
      getRowId={(i) => i.id}
      exportName={showDeal ? "invoice" : undefined}
      dense={!showDeal}
      toolbar={showDeal ? <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(INVOICE_STATUS)} /> : undefined}
    />
  );
}

/** Term progress, invoices and payments for one deal. */
export function DealBilling({
  dealId,
  caps,
  quotations,
  invoices,
  payments,
  termProgress,
  referenceTotal,
  paidTotal,
  paymentStatus,
}: {
  dealId: string;
  caps: SalesCaps;
  quotations: Quotation[];
  invoices: Invoice[];
  payments: Payment[];
  termProgress: TermProgress[];
  referenceTotal: string;
  paidTotal: string;
  paymentStatus: string;
}) {
  const accepted = quotations.find((q) => q.status === "diterima");
  const invoiceNo = Object.fromEntries(invoices.map((i) => [i.id, i.invoiceNumber]));
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
        <span>
          <span className="text-muted-foreground">{ss.deal.reference} </span>
          <Money value={referenceTotal} className="font-semibold" />
        </span>
        <span>
          <span className="text-muted-foreground">{ss.deal.paid} </span>
          <Money value={paidTotal} className="font-semibold" />
        </span>
        <SalesBadge value={paymentStatus} map={PAYMENT_STATUS} />
        {caps.write ? (
          <div className="ml-auto flex gap-2">
            <SubmitInvoiceDialog dealId={dealId} accepted={accepted} />
            <PaymentDialog dealId={dealId} invoices={invoices.filter((i) => i.status !== "batal")} />
          </div>
        ) : null}
      </div>
      {termProgress.length ? (
        <div className="overflow-x-auto rounded-lg border">
          <table className="w-full text-sm">
            <thead className="bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-3 py-2 text-left font-medium">{ss.deal.terms}</th>
                <th className="px-3 py-2 text-left font-medium">{s.due}</th>
                <th className="px-3 py-2 text-right font-medium">{s.amount}</th>
                <th className="px-3 py-2 text-right font-medium">{s.paid}</th>
                <th className="px-3 py-2 text-left font-medium">{s.status}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {termProgress.map((t) => (
                <tr key={t.termId}>
                  <td className="px-3 py-2">
                    {t.label} <span className="text-xs text-muted-foreground">{t.percent}%</span>
                  </td>
                  <td className="px-3 py-2 tabular">{formatDate(t.dueDate || null)}</td>
                  <td className="px-3 py-2 text-right">
                    <Money value={t.amount} />
                  </td>
                  <td className="px-3 py-2 text-right">
                    <Money value={t.paid} muteZero />
                  </td>
                  <td className="px-3 py-2">
                    <SalesBadge value={t.status} map={PAYMENT_STATUS} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {invoices.length ? <InvoicesTable invoices={invoices} caps={caps} showDeal={false} /> : null}
      <div>
        <h3 className="mb-2 text-sm font-semibold">{s.payments}</h3>
        {payments.length === 0 ? (
          <p className="text-sm text-muted-foreground">{s.noPayments}</p>
        ) : (
          <ul className="divide-y rounded-lg border text-sm">
            {payments.map((p) => (
              <li key={p.id} className="flex items-center gap-3 px-3 py-2">
                <Wallet className="size-4 text-muted-foreground" />
                <span className="min-w-0 flex-1">
                  <span className="tabular">{formatDate(p.paidOn)}</span>
                  <span className="text-muted-foreground">
                    {" "}
                    · {lbl(PAYMENT_METHOD, p.method)}
                    {p.invoiceId ? ` · ${invoiceNo[p.invoiceId] ?? s.invoice}` : ""}
                    {p.note ? ` · ${p.note}` : ""}
                  </span>
                </span>
                <Money value={p.amount} className="font-medium" />
                {caps.write ? (
                  <ConfirmAction
                    title={s.deletePayment}
                    action={() => deletePaymentAction(p.id)}
                    trigger={
                      <Button size="icon-xs" variant="ghost" aria-label={ss.delete}>
                        <Trash2 className="size-3.5" />
                      </Button>
                    }
                  />
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>
      {!caps.finance ? <p className="text-xs text-muted-foreground">{s.financeOnly}</p> : null}
    </div>
  );
}

function SubmitInvoiceDialog({ dealId, accepted }: { dealId: string; accepted?: Quotation }) {
  return (
    <FormDialog
      title={s.submitTitle}
      trigger={
        <Button size="sm" variant="outline">
          <Plus className="size-3.5" /> {s.submit}
        </Button>
      }
    >
      {(close) => <SubmitInvoiceForm dealId={dealId} accepted={accepted} close={close} />}
    </FormDialog>
  );
}

const FREE = "__free__";

function SubmitInvoiceForm({ dealId, accepted, close }: { dealId: string; accepted?: Quotation; close: () => void }) {
  const [term, setTerm] = React.useState(accepted?.terms[0]?.id ?? FREE);
  const [label, setLabel] = React.useState("");
  const [amount, setAmount] = React.useState("");
  const [dueDate, setDueDate] = React.useState("");
  const { pending, run } = useErpAction();
  const isFree = term === FREE;
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            submitInvoiceAction(dealId, {
              quotationId: isFree ? "" : (accepted?.id ?? ""),
              termId: isFree ? "" : term,
              label,
              amount: isFree ? amount : "",
              dueDate,
            }),
          { success: s.submitted, onOk: close },
        );
      }}
    >
      <Fld label={s.term} htmlFor="inv-term" className="sm:col-span-2">
        <SimpleSelect
          id="inv-term"
          value={term}
          onChange={setTerm}
          options={[
            ...(accepted?.terms ?? []).map((t) => ({ value: t.id, label: `${t.label} · ${t.percent}%` })),
            { value: FREE, label: s.freeAmount },
          ]}
        />
      </Fld>
      <Fld label={s.label} htmlFor="inv-label" className="sm:col-span-2">
        <Input id="inv-label" value={label} onChange={(e) => setLabel(e.target.value)} />
      </Fld>
      {isFree ? (
        <Fld label={`${s.amount} *`} htmlFor="inv-amount">
          <MoneyInput id="inv-amount" value={amount} onChange={setAmount} />
        </Fld>
      ) : null}
      <Fld label={s.due} htmlFor="inv-due">
        <DatePicker id="inv-due" value={dueDate} onChange={setDueDate} clearable />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} submitLabel={s.submit} />
      </div>
    </form>
  );
}

function PaymentDialog({ dealId, invoices }: { dealId: string; invoices: Invoice[] }) {
  return (
    <FormDialog
      title={s.paymentTitle}
      trigger={
        <Button size="sm">
          <Wallet className="size-3.5" /> {s.recordPayment}
        </Button>
      }
    >
      {(close) => <PaymentForm dealId={dealId} invoices={invoices} close={close} />}
    </FormDialog>
  );
}

function PaymentForm({ dealId, invoices, close }: { dealId: string; invoices: Invoice[]; close: () => void }) {
  const [v, setV] = React.useState<PaymentValues>({
    invoiceId: "",
    amount: "",
    method: "transfer",
    paidOn: toIsoDate(new Date()),
    note: "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (k: keyof PaymentValues, val: string) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => recordPaymentAction(dealId, v), {
          success: s.paymentSaved,
          onOk: close,
          onFail: (r) => setErrors(r.fieldErrors),
        });
      }}
    >
      <Fld label={s.invoice} htmlFor="pay-inv" className="sm:col-span-2">
        <SimpleSelect
          id="pay-inv"
          value={v.invoiceId ?? ""}
          onChange={(x) => {
            const inv = invoices.find((i) => i.id === x);
            setV((p) => ({ ...p, invoiceId: x, amount: inv && !p.amount ? inv.amount : p.amount }));
          }}
          options={invoices.map((i) => ({ value: i.id, label: `${i.invoiceNumber} · ${i.label}` }))}
          emptyLabel={s.noInvoice}
        />
      </Fld>
      <Fld label={`${s.amount} *`} htmlFor="pay-amount" error={errors?.amount}>
        <MoneyInput id="pay-amount" value={v.amount} onChange={(x) => set("amount", x)} />
      </Fld>
      <Fld label={s.method} htmlFor="pay-method">
        <SimpleSelect id="pay-method" value={v.method} onChange={(x) => set("method", x)} options={opts(PAYMENT_METHOD)} />
      </Fld>
      <Fld label={s.paidOn} htmlFor="pay-date" error={errors?.paidOn}>
        <DatePicker id="pay-date" value={v.paidOn} onChange={(x) => set("paidOn", x)} />
      </Fld>
      <Fld label={s.note} htmlFor="pay-note">
        <Input id="pay-note" value={v.note ?? ""} onChange={(e) => set("note", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
