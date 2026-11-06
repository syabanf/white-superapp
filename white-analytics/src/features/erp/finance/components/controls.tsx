"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Ban, PackageCheck, Plus, Trash2, Undo2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import {
  createProjectAllocationAction,
  deleteProjectAllocationAction,
  disposeAssetAction,
  matchInvoiceToPoAction,
  receiveGoodsAction,
  reverseJournalAction,
  voidGoodsReceiptAction,
} from "../actions";
import { fs } from "../strings";
import type { FixedAsset, GoodsReceipt, Invoice, PoMatch, PostableAccount, ProjectAllocation, ProjectTag, SalesReversal } from "../types";
import { AccountPicker } from "./account-picker";
import { TagPicker } from "./tag-picker";

const s = fs.controls;
type Errors = Record<string, string[]> | undefined;

export function SalesReversalsTable({ rows }: { rows: SalesReversal[] }) {
  const columns: ColumnDef<SalesReversal>[] = [
    { accessorKey: "docNo", header: s.document, cell: ({ row: { original: r } }) => <span>{r.docNo}<span className="block text-xs text-muted-foreground">{s.reversalKinds[r.kind]}</span></span> },
    { accessorKey: "partyName", header: s.party },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "reason", header: s.reason },
    { accessorKey: "error", header: s.blocker, cell: ({ row }) => <span className="text-xs text-destructive">{row.original.error}</span> },
  ];
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold">{s.reversalsTitle}</h2>
      <p className="text-xs text-muted-foreground">{s.reversalsHint}</p>
      <ErpDataTable columns={columns} data={rows} getRowId={(r) => r.id} emptyMessage={s.reversalsEmpty} dense />
    </section>
  );
}

export function ReverseJournalButton({ entryId }: { entryId: string }) {
  const { run } = useErpAction();
  return (
    <ReasonAction
      trigger={
        <Button variant="outline">
          <Undo2 className="size-4" /> {s.reverse}
        </Button>
      }
      title={s.reverseTitle}
      description={s.reverseHint}
      confirmLabel={s.reverse}
      reasonLabel={s.reason}
      onConfirm={async (reason: string) => (await run(() => reverseJournalAction(entryId, reason), { success: s.reversed })).ok}
    />
  );
}

function DisposeForm({ asset, accounts, close }: { asset: FixedAsset; accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState({ disposalDate: toIsoDate(new Date()), assetAccountId: "", cashAccountId: "", gainLossAccountId: "", proceeds: "", reason: "" });
  const [errors, setErrors] = React.useState<Errors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  const gain = parseMoney(v.proceeds) - parseMoney(asset.bookValue);
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => disposeAssetAction(asset.id, v), { success: s.disposed, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.disposalDate} htmlFor="dp-date" error={errors?.disposal_date}>
        <DatePicker id="dp-date" value={v.disposalDate} onChange={(x) => set({ disposalDate: x })} />
      </Fld>
      <Fld label={s.proceeds} htmlFor="dp-proceeds" error={errors?.proceeds}>
        <MoneyInput id="dp-proceeds" value={v.proceeds} onChange={(x) => set({ proceeds: x })} />
      </Fld>
      <Fld label={s.assetAccount} htmlFor="dp-asset" error={errors?.asset_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={v.assetAccountId} onChange={(x) => set({ assetAccountId: x })} />
      </Fld>
      {parseMoney(v.proceeds) > 0 ? (
        <Fld label={s.cashAccount} htmlFor="dp-cash" error={errors?.cash_account_id}>
          <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={v.cashAccountId} onChange={(x) => set({ cashAccountId: x })} />
        </Fld>
      ) : null}
      <Fld label={s.gainLossAccount} htmlFor="dp-gl" error={errors?.gain_loss_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={v.gainLossAccountId} onChange={(x) => set({ gainLossAccountId: x })} />
      </Fld>
      <Fld label={s.reason} htmlFor="dp-reason" error={errors?.reason}>
        <Input id="dp-reason" value={v.reason} onChange={(e) => set({ reason: e.target.value })} />
      </Fld>
      <p className="text-sm sm:col-span-2">
        {s.bookValue} {formatRupiah(asset.bookValue)}. {gain >= 0 ? s.gain : s.loss} {formatRupiah(String(Math.abs(gain)))}.
      </p>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function DisposeAssetDialog({ asset, accounts }: { asset: FixedAsset; accounts: PostableAccount[] }) {
  return (
    <FormDialog
      title={s.disposeTitle}
      description={s.disposeHint}
      size="lg"
      trigger={
        <Button size="xs" variant="ghost">
          {s.dispose}
        </Button>
      }
    >
      {(close) => <DisposeForm asset={asset} accounts={accounts} close={close} />}
    </FormDialog>
  );
}

function ReceiveForm({ match, close }: { match: PoMatch; close: () => void }) {
  const left = parseMoney(match.ordered) - parseMoney(match.received);
  const [v, setV] = React.useState({ documentId: match.documentId, receiptDate: toIsoDate(new Date()), amount: left.toFixed(2), notes: "" });
  const [errors, setErrors] = React.useState<Errors>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => receiveGoodsAction(v), { success: s.received, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.receiptAmount} htmlFor="gr-amount" error={errors?.amount}>
        <MoneyInput id="gr-amount" value={v.amount} onChange={(x) => setV((p) => ({ ...p, amount: x }))} />
      </Fld>
      <Fld label={s.receiptDate} htmlFor="gr-date" error={errors?.receipt_date}>
        <DatePicker id="gr-date" value={v.receiptDate} onChange={(x) => setV((p) => ({ ...p, receiptDate: x }))} />
      </Fld>
      <Fld label={s.notes} htmlFor="gr-notes">
        <Input id="gr-notes" value={v.notes} onChange={(e) => setV((p) => ({ ...p, notes: e.target.value }))} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function MatchForm({ match, drafts, close }: { match: PoMatch; drafts: Invoice[]; close: () => void }) {
  const [invoiceId, setInvoiceId] = React.useState(drafts[0]?.id ?? "");
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => matchInvoiceToPoAction(invoiceId, match.documentId), { success: s.matched, onOk: close });
      }}
    >
      <Fld label={s.draftInvoice} htmlFor="gr-invoice">
        <SimpleSelect id="gr-invoice" value={invoiceId} onChange={setInvoiceId} options={drafts.map((i) => ({ value: i.id, label: `${i.invoiceNo} · ${i.partyName} · ${formatRupiah(i.subtotal)}` }))} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function ReceiptLine({ receipt, canApprove }: { receipt: GoodsReceipt; canApprove: boolean }) {
  const { run } = useErpAction();
  return (
    <span className="flex items-center gap-1 text-xs text-muted-foreground">
      <span className={receipt.status === "VOID" ? "line-through" : undefined}>{receipt.receiptNo} · {formatRupiah(receipt.amount)}</span>
      {canApprove && receipt.status === "POSTED" ? (
        <ReasonAction
          trigger={
            <Button size="icon-xs" variant="ghost" aria-label={s.voidReceipt} className="text-destructive">
              <Ban className="size-3" />
            </Button>
          }
          title={s.voidReceipt}
          description={s.voidReceiptHint}
          confirmLabel={s.voidReceipt}
          reasonLabel={s.reason}
          onConfirm={async (reason: string) => (await run(() => voidGoodsReceiptAction(receipt.id, reason), { success: s.receiptVoided })).ok}
        />
      ) : null}
    </span>
  );
}

export function PoMatchTable({ rows, drafts, canWrite, canApprove }: { rows: PoMatch[]; drafts: Invoice[]; canWrite: boolean; canApprove: boolean }) {
  const columns: ColumnDef<PoMatch>[] = [
    { accessorKey: "number", header: s.po, cell: ({ row: { original: m } }) => <span>{m.number}<span className="block text-xs text-muted-foreground">{m.partyName} · {m.title}</span></span> },
    { accessorKey: "ordered", header: s.ordered, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.ordered} /> },
    {
      accessorKey: "received",
      header: s.receivedCol,
      meta: { align: "right" },
      cell: ({ row: { original: m } }) => (
        <span className="flex flex-col items-end">
          <Money value={m.received} muteZero />
          {m.receipts?.map((r) => <ReceiptLine key={r.id} receipt={r} canApprove={canApprove} />)}
        </span>
      ),
    },
    {
      accessorKey: "billed",
      header: s.billed,
      meta: { align: "right" },
      cell: ({ row: { original: m } }) => (
        <span className="flex flex-col items-end">
          <Money value={m.billed} muteZero />
          {m.invoices?.map((i) => <span key={i.invoiceId} className="text-xs text-muted-foreground">{i.invoiceNo} · {i.status}</span>)}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: m } }) =>
        canWrite ? (
          <span className="flex justify-end gap-0.5">
            {parseMoney(m.received) < parseMoney(m.ordered) ? (
              <FormDialog title={s.receiveTitle} description={s.receiveHint} trigger={<Button size="xs" variant="ghost"><PackageCheck className="size-3.5" /> {s.receive}</Button>}>
                {(close) => <ReceiveForm match={m} close={close} />}
              </FormDialog>
            ) : null}
            {drafts.length ? (
              <FormDialog title={s.matchTitle} description={s.matchHint} trigger={<Button size="xs" variant="ghost">{s.match}</Button>}>
                {(close) => <MatchForm match={m} drafts={drafts} close={close} />}
              </FormDialog>
            ) : null}
          </span>
        ) : null,
    },
  ];
  return <ErpDataTable columns={columns} data={rows} getRowId={(m) => m.documentId} emptyMessage={s.poEmpty} exportName="pencocokan-po" />;
}

function AllocationForm({ tags, close }: { tags: ProjectTag[]; close: () => void }) {
  const [v, setV] = React.useState({ tagId: "", month: toIsoDate(new Date()).slice(0, 7), kind: "PAYROLL", amount: "", notes: "" });
  const [errors, setErrors] = React.useState<Errors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createProjectAllocationAction(v), { success: s.allocated, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.project} htmlFor="al-tag" error={errors?.tag_id}>
        <TagPicker id="al-tag" tags={tags} value={v.tagId} onChange={(x) => set({ tagId: x })} />
      </Fld>
      <Fld label={s.month} htmlFor="al-month" error={errors?.month}>
        <Input id="al-month" value={v.month} onChange={(e) => set({ month: e.target.value })} />
      </Fld>
      <Fld label={s.costKind} htmlFor="al-kind">
        <SimpleSelect id="al-kind" value={v.kind} onChange={(x) => set({ kind: x })} options={Object.entries(s.allocKinds).map(([value, label]) => ({ value, label }))} />
      </Fld>
      <Fld label={s.amount} htmlFor="al-amount" error={errors?.amount}>
        <MoneyInput id="al-amount" value={v.amount} onChange={(x) => set({ amount: x })} />
      </Fld>
      <Fld label={s.notes} htmlFor="al-notes">
        <Input id="al-notes" value={v.notes} onChange={(e) => set({ notes: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

export function AllocationDialog({ tags }: { tags: ProjectTag[] }) {
  return (
    <FormDialog title={s.allocTitle} description={s.allocHint} trigger={<Button size="sm"><Plus className="size-4" /> {s.allocNew}</Button>}>
      {(close) => <AllocationForm tags={tags} close={close} />}
    </FormDialog>
  );
}

export function AllocationsTable({ rows, canApprove }: { rows: ProjectAllocation[]; canApprove: boolean }) {
  const { run } = useErpAction();
  const columns: ColumnDef<ProjectAllocation>[] = [
    { accessorKey: "month", header: s.month },
    { accessorKey: "tagName", header: s.project, cell: ({ row: { original: a } }) => <span>{a.tagName}<span className="block text-xs text-muted-foreground">{a.clientName}</span></span> },
    { accessorKey: "kind", header: s.costKind, cell: ({ row }) => s.allocKinds[row.original.kind] },
    { accessorKey: "notes", header: s.notes },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: a } }) =>
        canApprove ? (
          <Button size="icon-xs" variant="ghost" className="text-destructive" aria-label={s.allocDelete} onClick={() => void run(() => deleteProjectAllocationAction(a.id), { success: s.allocDeleted })}>
            <Trash2 className="size-3.5" />
          </Button>
        ) : null,
    },
  ];
  return <ErpDataTable columns={columns} data={rows} getRowId={(a) => a.id} emptyMessage={s.allocEmpty} exportName="alokasi-biaya" />;
}
