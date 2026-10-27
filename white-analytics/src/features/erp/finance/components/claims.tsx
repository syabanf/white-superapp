"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { AlertTriangle, Check, Loader2, Paperclip, Wallet, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Textarea } from "@/components/ui/textarea";
import { formatDate } from "@/lib/format";
import { toast } from "@/lib/toast";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction, ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { formatRupiah, sumMoney } from "@/features/erp/shared/money";
import { DatePicker, formatMonth, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { approveClaimAction, payClaimsAction, rejectClaimAction } from "../actions";
import type { Claim, PostableAccount } from "../types";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";

const rb = fs.reimburse;

export function ClaimStatusBadge({ status }: { status: string }) {
  return (
    <ErpStatusBadge
      status={status}
      labels={rb.statuses}
      kinds={{
        CLAIM_DRAFT: "neutral",
        SUBMITTED: "info",
        APPROVED_MANAGER: "warning",
        APPROVED_FINANCE: "good",
        PAID: "good",
        REJECTED: "critical",
        CANCELLED: "neutral",
      }}
    />
  );
}

export const categoryLabel = (c: string) => rb.categories[c] ?? c;

function OverLimit({ claim }: { claim: Claim }) {
  if (!claim.overLimit) return null;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant="outline" className="gap-1 border-amber-500/40 text-amber-700 dark:text-amber-400">
          <AlertTriangle className="size-3" /> {rb.overLimit}
        </Badge>
      </TooltipTrigger>
      {claim.limitNote ? <TooltipContent className="max-w-64 text-xs">{claim.limitNote}</TooltipContent> : null}
    </Tooltip>
  );
}

/** Approve and reject buttons for one claim (inbox rows and the detail page). */
export function ClaimDecision({ claim, size = "xs" }: { claim: Claim; size?: "xs" | "sm" }) {
  const [note, setNote] = React.useState("");
  const { run } = useErpAction();
  if (!claim.canApprove) return null;
  return (
    <span className="flex justify-end gap-1">
      <ReasonAction
        trigger={
          <Button size={size} variant="outline" className="text-destructive">
            <X className="size-3.5" /> {rb.reject}
          </Button>
        }
        title={rb.rejectTitle}
        description={`${claim.claimNo} · ${claim.employeeName} · ${formatRupiah(claim.amount)}`}
        reasonLabel={rb.rejectReason}
        confirmLabel={rb.reject}
        onConfirm={async (reason) => (await run(() => rejectClaimAction(claim.id, reason), { success: rb.rejected })).ok}
      />
      <ConfirmAction
        trigger={
          <Button size={size}>
            <Check className="size-3.5" /> {rb.approve}
          </Button>
        }
        title={`${rb.approve} ${claim.claimNo}?`}
        description={`${claim.employeeName} · ${categoryLabel(claim.category)} · ${formatRupiah(claim.amount)}`}
        confirmLabel={rb.approve}
        onConfirm={async () => (await run(() => approveClaimAction(claim.id, note), { success: rb.approved })).ok}
      >
        <FormField label={rb.approveNote} htmlFor={`note-${claim.id}`}>
          <Input id={`note-${claim.id}`} value={note} onChange={(e) => setNote(e.target.value)} />
        </FormField>
      </ConfirmAction>
    </span>
  );
}

function baseColumns(detailHref: (c: Claim) => string): ColumnDef<Claim, unknown>[] {
  return [
    {
      accessorKey: "claimNo",
      header: rb.claimNo,
      cell: ({ row }) => (
        <Link href={detailHref(row.original)} className="font-mono text-xs font-medium hover:underline">
          {row.original.claimNo || "–"}
        </Link>
      ),
    },
    {
      accessorKey: "employeeName",
      header: rb.employee,
      cell: ({ row }) => (
        <span>
          <span className="block">{row.original.employeeName}</span>
          <span className="text-xs text-muted-foreground">{row.original.departmentName}</span>
        </span>
      ),
    },
    {
      accessorKey: "category",
      header: rb.category,
      cell: ({ row }) => categoryLabel(row.original.category),
    },
    { accessorKey: "transactionDate", header: rb.transactionDate, cell: ({ row }) => formatDate(row.original.transactionDate) },
    {
      accessorKey: "description",
      header: fs.description,
      cell: ({ row }) => (
        <span className="flex items-start gap-1.5">
          <span className="line-clamp-2 max-w-xs">{row.original.description}</span>
          {row.original.receipts.length ? (
            <span className="flex shrink-0 items-center gap-0.5 text-xs text-muted-foreground" title={rb.receiptCount(row.original.receipts.length)}>
              <Paperclip className="size-3" />
              {row.original.receipts.length}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      accessorKey: "amount",
      header: fs.amount,
      meta: { align: "right" },
      cell: ({ row }) => (
        <span className="flex flex-col items-end gap-1">
          <Money value={row.original.amount} />
          <OverLimit claim={row.original} />
        </span>
      ),
    },
    { accessorKey: "status", header: fs.status, cell: ({ row }) => <ClaimStatusBadge status={row.original.status} /> },
  ];
}

export function InboxList({ claims }: { claims: Claim[] }) {
  if (claims.length === 0) return <ErpEmpty title={rb.emptyInbox} />;
  const columns: ColumnDef<Claim, unknown>[] = [
    ...baseColumns((c) => `/erp/finance/reimbursements/${c.id}`),
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) => <ClaimDecision claim={row.original} />,
    },
  ];
  return <ErpDataTable columns={columns} data={claims} emptyMessage={rb.emptyInbox} getRowId={(c) => c.id} />;
}

function last12Months() {
  const out: { value: string; label: string }[] = [];
  const d = new Date();
  for (let i = 0; i < 12; i++) {
    const m = new Date(d.getFullYear(), d.getMonth() - i, 1);
    const v = `${m.getFullYear()}-${String(m.getMonth() + 1).padStart(2, "0")}`;
    out.push({ value: v, label: formatMonth(v) });
  }
  return out;
}

function PayDialog({ claims, accounts, onDone }: { claims: Claim[]; accounts: PostableAccount[]; onDone: () => void }) {
  const [open, setOpen] = React.useState(false);
  const [form, setForm] = React.useState({ cashAccountId: "", paidAt: toIsoDate(new Date()), notes: "" });
  const { pending, run } = useErpAction();
  const total = sumMoney(claims.map((c) => c.amount));
  return (
    <>
      <Button size="sm" disabled={claims.length === 0} onClick={() => setOpen(true)}>
        <Wallet className="size-4" /> {rb.pay}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{rb.payTitle(claims.length)}</DialogTitle>
            <DialogDescription>{rb.payDesc}</DialogDescription>
          </DialogHeader>
          <div className="max-h-48 overflow-auto rounded-lg border text-sm scrollbar-thin">
            {claims.map((c) => (
              <div key={c.id} className="flex justify-between gap-3 border-b px-3 py-1.5 last:border-b-0">
                <span className="truncate">
                  <span className="font-mono text-xs">{c.claimNo}</span> · {c.employeeName}
                </span>
                <Money value={c.amount} />
              </div>
            ))}
          </div>
          <div className="flex items-center justify-between rounded-lg bg-muted/40 px-4 py-2.5 text-sm">
            <span className="text-muted-foreground">{fs.total}</span>
            <span className="text-base font-semibold tabular">{formatRupiah(total)}</span>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={rb.paidAt} required>
              <DatePicker value={form.paidAt} onChange={(v) => setForm({ ...form, paidAt: v })} />
            </FormField>
            <FormField label={rb.cashAccount} hint={fs.mappingDefault}>
              <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={form.cashAccountId} onChange={(v) => setForm({ ...form, cashAccountId: v })} clearable />
            </FormField>
          </div>
          <FormField label={rb.notes} htmlFor="pay-notes">
            <Textarea id="pay-notes" rows={2} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </FormField>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setOpen(false)}>
              {fs.cancel}
            </Button>
            <Button
              disabled={pending}
              onClick={() =>
                run(() => payClaimsAction({ claimIds: claims.map((c) => c.id), ...form }), {
                  onOk: (batch) => {
                    toast.success(rb.paid(batch.batchNo));
                    if (batch.journalStatus && batch.journalStatus !== "posted" && batch.journalReason) toast.warning(batch.journalReason);
                    setOpen(false);
                    onDone();
                  },
                })
              }
            >
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {rb.pay} {formatRupiah(total)}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function ClaimsTable({
  claims,
  page,
  totalAmount,
  canPay,
  accounts,
  departments,
}: {
  claims: Claim[];
  page: ServerPage;
  totalAmount: string;
  canPay: boolean;
  accounts: PostableAccount[];
  /** department names for the department filter */
  departments: string[];
}) {
  const [selected, setSelected] = React.useState<Set<string>>(new Set());
  const payable = claims.filter((c) => c.status === "APPROVED_FINANCE");
  const chosen = payable.filter((c) => selected.has(c.id));
  const toggle = (id: string, on: boolean) =>
    setSelected((s) => {
      const next = new Set(s);
      if (on) next.add(id);
      else next.delete(id);
      return next;
    });
  const selectColumn: ColumnDef<Claim, unknown> = {
    id: "select",
    enableSorting: false,
    header: () => (
      <Checkbox
        aria-label={rb.pay}
        checked={payable.length > 0 && chosen.length === payable.length}
        onCheckedChange={(v) => setSelected(v === true ? new Set(payable.map((c) => c.id)) : new Set())}
        disabled={payable.length === 0}
      />
    ),
    cell: ({ row }) =>
      row.original.status === "APPROVED_FINANCE" ? (
        <Checkbox aria-label={row.original.claimNo} checked={selected.has(row.original.id)} onCheckedChange={(v) => toggle(row.original.id, v === true)} />
      ) : null,
  };
  const columns = [...(canPay ? [selectColumn] : []), ...baseColumns((c) => `/erp/finance/reimbursements/${c.id}`)];
  return (
    <div className="page-stack">
      <ErpDataTable
        columns={columns}
        data={claims}
        pagination={page}
        emptyMessage={rb.empty}
        getRowId={(c) => c.id}
        toolbar={
          <>
            <SearchFilter placeholder={rb.searchPh} />
            <SelectFilter
              param="status"
              label={fs.status}
              allLabel={fs.status}
              options={Object.entries(rb.statuses).map(([value, label]) => ({ value, label }))}
            />
            <SelectFilter
              param="category"
              label={rb.category}
              allLabel={rb.allCategories}
              options={Object.entries(rb.categories).map(([value, label]) => ({ value, label }))}
            />
            {departments.length ? (
              <SelectFilter
                param="department"
                label={rb.department}
                allLabel={rb.allDepartments}
                options={departments.map((d) => ({ value: d, label: d }))}
              />
            ) : null}
            <SelectFilter param="month" label={rb.month} allLabel={rb.month} options={last12Months()} />
            <DateRangeFilter placeholder={rb.dateRange} />
            <ResetFilters params={["q", "status", "category", "department", "month", "from", "to"]} />
          </>
        }
      />
      <Card className="flex-row flex-wrap items-center justify-between gap-3 px-(--card-pad) text-sm">
        <span className="text-muted-foreground">
          {rb.totalAmount}: <span className="font-semibold text-foreground tabular">{formatRupiah(totalAmount)}</span>
        </span>
        {canPay ? (
          <span className="flex items-center gap-3">
            <span className="text-xs text-muted-foreground">
              {chosen.length ? rb.selected(chosen.length, formatRupiah(sumMoney(chosen.map((c) => c.amount)))) : rb.selectToPay}
            </span>
            <PayDialog claims={chosen} accounts={accounts} onDone={() => setSelected(new Set())} />
          </span>
        ) : null}
      </Card>
    </div>
  );
}

/** Claimant's own list (self-service). */
export function MyClaimsTable({ claims }: { claims: Claim[] }) {
  const router = useRouter();
  const columns = baseColumns((c) => `/erp/me/reimbursements/${c.id}`).filter(
    (c) => !("accessorKey" in c) || c.accessorKey !== "employeeName",
  );
  return (
    <ErpDataTable
      columns={columns}
      data={claims}
      emptyMessage={rb.emptyMine}
      onRowClick={(r) => router.push(`/erp/me/reimbursements/${r.original.id}`)}
      toolbar={
        <SelectFilter
          param="status"
          label={fs.status}
          allLabel={fs.status}
          options={Object.entries(rb.statuses).map(([value, label]) => ({ value, label }))}
        />
      }
    />
  );
}
