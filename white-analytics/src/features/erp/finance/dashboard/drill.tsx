"use client";

import * as React from "react";
import { CalendarRange, FileText, Filter } from "lucide-react";
import { DrillSheet, bucketLabel, useDrill, useDrillParams, type DrillAction, type DrillResult, type DrillTarget } from "@/features/erp/shared/drill";
import { erpClientFetch } from "@/features/erp/shared/client";
import { formatPeriod } from "@/features/erp/shared/dashboard-period";
import { formatDate, formatNumber } from "@/lib/format";
import type { CashMovement, ClaimsResponse, Invoice, JournalEntry } from "../types";
import { PaymentStatusBadge } from "../components/invoices";
import { claimsHref, compactIdr, invoicesHref, journalsHref, money, monthBounds, rangeQs } from "./lib";
import { ds } from "./strings";

/**
 * Drill sheets of the finance dashboard. A mark opens one with
 * `useFinanceDrill()(kind, params)`; the key is a query string of the
 * records behind the mark (period, party, department, category...).
 */

type Params = Record<string, string | undefined>;
export type FinanceDrillKind = "journals" | "invoices" | "claims" | "cash";

const PREVIEW = 20;
const t = ds.drill;

const encode = (p: Params) => {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(p)) if (v) sp.set(k, v);
  return sp.toString();
};
const decode = (key: string): Params => Object.fromEntries(new URLSearchParams(key));

export function useFinanceDrill() {
  const { open } = useDrill();
  return React.useCallback((kind: FinanceDrillKind, p: Params) => open(kind, encode(p)), [open]);
}

/** Opens the sheet for one month of a monthly chart; `extra` narrows it further. */
export function useMonthDrill(months: string[], kind: FinanceDrillKind = "journals", extra: Params = {}) {
  const drill = useFinanceDrill();
  const extraKey = encode(extra);
  return React.useCallback(
    (i: number) => {
      const m = months[i];
      if (m) drill(kind, { ...decode(extraKey), ...monthBounds(m) });
    },
    [months, kind, extraKey, drill],
  );
}

async function loadJournals(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const q = encode({ dateFrom: p.from, dateTo: p.to, accountId: p.account, status: "POSTED", pageSize: String(PREVIEW) });
  const res = await erpClientFetch<{ entries: JournalEntry[]; total: number }>(`/v1/finance/journal-entries?${q}`, { signal });
  return {
    total: res.total,
    summary: [
      { label: t.entries, value: formatNumber(res.total) },
      { label: t.shownDebit, value: compactIdr(res.entries.reduce((a, e) => a + money(e.totalDebit), 0)) },
    ],
    items: res.entries.map((e) => ({
      id: e.id,
      title: e.description || e.entryNo,
      subtitle: `${e.entryNo} · ${formatDate(e.entryDate)}`,
      value: compactIdr(money(e.totalDebit)),
      href: `/erp/finance/journals/${e.id}`,
    })),
  };
}

async function loadInvoices(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const kind = p.kind === "AP" ? "AP" : "AR";
  const q = encode({
    dateFrom: p.from,
    dateTo: p.to,
    partyName: p.party,
    paymentStatus: p.pay,
    status: "POSTED",
    pageSize: String(PREVIEW),
  });
  const res = await erpClientFetch<{ invoices: Invoice[]; total: number }>(`/v1/finance/${kind === "AP" ? "ap" : "ar"}/invoices?${q}`, { signal });
  return {
    total: res.total,
    summary: [
      { label: t.invoices, value: formatNumber(res.total) },
      { label: t.outstanding, value: compactIdr(res.invoices.reduce((a, i) => a + money(i.outstandingAmount), 0)) },
    ],
    items: res.invoices.map((i) => ({
      id: i.id,
      title: i.partyName,
      subtitle: `${i.invoiceNo} · ${formatDate(i.invoiceDate)}`,
      value: compactIdr(money(i.totalAmount)),
      meta: money(i.outstandingAmount) > 0 ? t.remaining(compactIdr(money(i.outstandingAmount))) : undefined,
      badge: <PaymentStatusBadge status={i.paymentStatus} />,
      href: invoicesHref(kind, { q: i.invoiceNo }),
    })),
  };
}

async function loadClaims(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const q = encode({
    dateFrom: p.from,
    dateTo: p.to,
    department: p.department,
    category: p.category,
    status: p.status,
    pageSize: String(PREVIEW),
  });
  const res = await erpClientFetch<ClaimsResponse>(`/v1/finance/reimbursements?${q}`, { signal });
  return {
    total: res.total,
    summary: [
      { label: t.claims, value: formatNumber(res.total) },
      { label: t.total, value: compactIdr(money(res.totalAmount)) },
    ],
    items: res.claims.map((c) => ({
      id: c.id,
      title: c.employeeName,
      subtitle: `${c.claimNo} · ${ds.spend.categoryNames[c.category] ?? c.category} · ${formatDate(c.transactionDate)}`,
      value: compactIdr(money(c.amount)),
      meta: ds.spend.statusNames[c.status] ?? c.status,
      href: `/erp/finance/reimbursements/${c.id}`,
    })),
  };
}

async function loadCash(p: Params, signal: AbortSignal): Promise<DrillResult> {
  const q = encode({ dateFrom: p.from, dateTo: p.to, pageSize: String(PREVIEW) });
  const res = await erpClientFetch<{ movements: CashMovement[]; total: number }>(`/v1/finance/cash-bank/movements?${q}`, { signal });
  const signed = (m: CashMovement) => (m.kind === "CASH_OUT" ? -money(m.amount) : money(m.amount));
  return {
    total: res.total,
    summary: [
      { label: t.movements, value: formatNumber(res.total) },
      { label: t.shownNet, value: compactIdr(res.movements.reduce((a, m) => a + signed(m), 0)) },
    ],
    items: res.movements.map((m) => ({
      id: m.id,
      title: m.description || m.entryNo,
      subtitle: `${m.cashAccount?.name ?? "–"} · ${formatDate(m.entryDate)}`,
      value: (
        <span className={signed(m) < 0 ? "text-negative" : "text-positive"}>
          {signed(m) < 0 ? "−" : "+"}
          {compactIdr(Math.abs(signed(m)))}
        </span>
      ),
      href: m.cashAccount ? `/erp/finance/cash-bank/${m.cashAccount.id}?${rangeQs(p.from ?? "", p.to ?? "")}` : undefined,
    })),
  };
}

function periodChip(p: Params): string | undefined {
  if (!p.from || !p.to) return undefined;
  const month = monthBounds(p.from.slice(0, 7));
  return month.from === p.from && month.to === p.to ? bucketLabel(p.from, "month") : formatPeriod(p.from, p.to);
}

/** Mount once on the finance dashboard. `period` is the dashboard range. */
export function FinanceDrill({ period }: { period: { from: string; to: string } }) {
  const { hrefWith } = useDrillParams();

  const chips = ({ key }: DrillTarget) => {
    const p = decode(key);
    return [
      periodChip(p),
      p.kind ? (p.kind === "AP" ? ds.arap.payable : ds.arap.receivable) : undefined,
      p.party,
      p.department,
      p.category ? (ds.spend.categoryNames[p.category] ?? p.category) : undefined,
      p.status ? (ds.spend.statusNames[p.status] ?? p.status) : undefined,
      p.label,
    ].filter((c): c is string => Boolean(c));
  };

  const actions = ({ kind, key }: DrillTarget): DrillAction[] => {
    const p = decode(key);
    const out: DrillAction[] = [];
    if (p.from && p.to && (p.from !== period.from || p.to !== period.to)) {
      out.push({ label: t.zoom, icon: <CalendarRange aria-hidden />, href: hrefWith({ from: p.from, to: p.to, preset: null }) });
    }
    if (kind === "journals" && p.from && p.to) {
      out.push({ label: t.incomeStatement, icon: <FileText aria-hidden />, href: `/erp/finance/reports?r=income-statement&${rangeQs(p.from, p.to)}` });
    }
    if (kind === "cash" && p.from && p.to) {
      out.push({ label: t.cashFlow, icon: <FileText aria-hidden />, href: `/erp/finance/reports?r=cash-flow&${rangeQs(p.from, p.to)}` });
    }
    if (p.department) out.push({ label: t.filterDept, icon: <Filter aria-hidden />, href: hrefWith({ department: p.department }) });
    return out;
  };

  return (
    <>
      <DrillSheet
        kind="journals"
        title={({ key }) => decode(key).label ?? t.journalsTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadJournals(decode(key), signal)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return journalsHref(p.from ?? period.from, p.to ?? period.to, undefined, p.account);
        }}
      />
      <DrillSheet
        kind="invoices"
        title={({ key }) => (decode(key).kind === "AP" ? t.payablesTitle : t.receivablesTitle)}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadInvoices(decode(key), signal)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return invoicesHref(p.kind === "AP" ? "AP" : "AR", { from: p.from, to: p.to, party: p.party, pay: p.pay });
        }}
      />
      <DrillSheet
        kind="claims"
        title={() => t.claimsTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadClaims(decode(key), signal)}
        fullListHref={({ key }) => {
          const p = decode(key);
          return claimsHref({ from: p.from, to: p.to, department: p.department, category: p.category, status: p.status });
        }}
      />
      <DrillSheet
        kind="cash"
        title={() => t.cashTitle}
        chips={chips}
        actions={actions}
        load={({ key }, signal) => loadCash(decode(key), signal)}
        fullListHref={() => "/erp/finance/cash-bank"}
      />
    </>
  );
}
