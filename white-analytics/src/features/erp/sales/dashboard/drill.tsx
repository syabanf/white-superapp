"use client";

import * as React from "react";
import { DrillSheet, useDrill, useDrillParams, type DrillAction, type DrillItem, type DrillResult, type DrillTarget } from "@/features/erp/shared/drill";
import { bucketLabel, bucketRange } from "@/features/erp/shared/drill";
import { erpClientFetch } from "@/features/erp/shared/client";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatDate, formatNumber } from "@/lib/format";
import { LEAD_SOURCE, LEAD_STATUS, SERVICE_LINE } from "../strings";
import type { Conversation, Deal, Invoice, Lead, ListDealsResponse, ListLeadsResponse } from "../types";
import { rpCompact, sum } from "./format";
import { dd } from "./strings";
import type { DashboardQuery } from "./types";

/**
 * Drill-down of the sales dashboard. A mark opens `?drill=<kind>:<filters>`
 * where filters is a query string in the list pages' own param names, so
 * the sheet's "Buka daftar lengkap" link reuses it as is.
 *
 * deal:    pipeline owner line industry stage source status reason from to date(created|closed) age
 * lead:    owner source status from to scoreMin scoreMax
 * invoice: status from to age(receivable bucket)
 * chat:    owner channel from to
 * `l` carries the mark's label for the title.
 */
export type DrillFilter = Record<string, string | number | undefined | null>;

const KINDS = ["deal", "lead", "invoice", "chat"] as const;
type Kind = (typeof KINDS)[number];

const PAGES: Record<Kind, string> = {
  deal: "/erp/sales/pipeline",
  lead: "/erp/sales/leads",
  invoice: "/erp/sales/invoices",
  chat: "/erp/sales/chatbot",
};

const LIMIT = 200;
const t = dd.drill;

function encode(f: DrillFilter): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(f)) if (v != null && v !== "") p.set(k, String(v));
  return p.toString();
}

/** Dashboard filters every drill inherits (a mark's own filter wins). */
export function baseFilter(q: DashboardQuery): DrillFilter {
  return { pipeline: q.pipelineId, owner: q.ownerUserId, line: q.serviceLine, source: q.source };
}

/** Period of the dashboard as list params. */
export function periodFilter(q: DashboardQuery, date: "created" | "closed" = "closed"): DrillFilter {
  return { from: q.from, to: q.to, date };
}

/** Month "2026-03" as a from/to filter. */
export function monthFilter(ym: string): DrillFilter {
  return bucketRange(ym, "month");
}

/** href of a list page with the drill's filters (no label). */
export function listHref(kind: Kind, f: DrillFilter): string {
  const qs = encode(f);
  return qs ? `${PAGES[kind]}?${qs}` : PAGES[kind];
}

/** Opens a sales drill sheet; the dashboard filters are merged in. */
export function useSalesDrill(query: DashboardQuery) {
  const { open } = useDrill();
  return React.useCallback(
    (kind: Kind, f: DrillFilter) => {
      const own = kind === "deal" ? baseFilter(query) : kind === "lead" ? { owner: query.ownerUserId, source: query.source } : kind === "chat" ? { owner: query.ownerUserId } : {};
      open(kind, encode({ ...own, ...f }));
    },
    [open, query],
  );
}

function read(target: DrillTarget): Record<string, string> {
  return Object.fromEntries(new URLSearchParams(target.key));
}

const dealValue = (d: Deal) => Number(d.valueFinal || d.valueEstimate || 0);
const daysSince = (ts: string | null, now = Date.now()) => (ts ? Math.floor((now - Date.parse(ts)) / 86_400_000) : 0);

function inAgeBucket(days: number, bucket: string): boolean {
  const [lo, hi] = bucket.endsWith("+") ? [Number(bucket.slice(0, -1)), Infinity] : bucket.split("-").map(Number);
  return days >= (lo ?? 0) && days <= (hi ?? Infinity);
}

async function loadDeals(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const qs = encode({
    pipelineId: f.pipeline,
    ownerUserId: f.owner,
    serviceLine: f.line,
    industry: f.industry,
    stageId: f.stage,
    source: f.source,
    status: f.status,
    lostReasonId: f.reason,
    dateFrom: f.from,
    dateTo: f.to,
    dateField: f.from || f.to ? f.date || "created" : undefined,
    pageSize: f.age ? 500 : LIMIT,
  });
  const res = await erpClientFetch<ListDealsResponse>(`/v1/sales/deals?${qs}`, { signal });
  let deals = res.deals;
  let total = res.total;
  if (f.age) {
    deals = deals.filter((d) => inAgeBucket(daysSince(d.enteredStageAt), f.age!));
    total = deals.length;
  }
  const closed = f.status === "won" || f.status === "lost" || f.date === "closed";
  const items: DrillItem[] = deals.map((d) => ({
    id: d.id,
    title: d.title,
    subtitle: [d.clientName, d.stageName].filter(Boolean).join(" · "),
    value: rpCompact(dealValue(d)),
    meta: closed && d.closedAt ? formatDate(d.closedAt) : f.age ? dd.days(String(daysSince(d.enteredStageAt))) : formatDate(d.createdAt),
    href: `/erp/sales/pipeline/${d.id}`,
  }));
  const value = sum(deals.map(dealValue));
  return {
    items,
    total,
    summary: [
      { label: t.deals, value: formatNumber(total) },
      { label: total > deals.length ? t.valueShown : t.value, value: formatRupiah(value) },
    ],
  };
}

async function loadLeads(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const qs = encode({
    ownerUserId: f.owner,
    source: f.source,
    status: f.status,
    dateFrom: f.from,
    dateTo: f.to,
    scoreMin: f.scoreMin,
    scoreMax: f.scoreMax,
    pageSize: 100,
  });
  const res = await erpClientFetch<ListLeadsResponse>(`/v1/sales/leads?${qs}`, { signal });
  const scores = res.leads.map((l) => l.score);
  return {
    total: res.total,
    summary: [
      { label: t.leads, value: formatNumber(res.total) },
      { label: t.avgScore, value: scores.length ? formatNumber(sum(scores) / scores.length, 0) : "–" },
    ],
    items: res.leads.map((l: Lead) => ({
      id: l.id,
      title: l.orgName,
      subtitle: [l.picName, LEAD_SOURCE[l.source] ?? l.source, LEAD_STATUS[l.status] ?? l.status].filter(Boolean).join(" · "),
      value: t.score(l.score),
      meta: formatDate(l.createdAt),
      href: `/erp/sales/leads/${l.id}`,
    })),
  };
}

const todayIso = () => new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(new Date());

async function loadInvoices(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const qs = encode({ status: f.age ? "terkirim" : f.status, dateFrom: f.age ? undefined : f.from, dateTo: f.to, limit: 500 });
  const res = await erpClientFetch<{ invoices: Invoice[] }>(`/v1/sales/invoices?${qs}`, { signal });
  const asOf = f.to && f.to < todayIso() ? f.to : todayIso();
  const left = (i: Invoice) => Number(i.amount) - Number(i.paidTotal);
  const late = (i: Invoice) => (i.dueDate ? Math.round((Date.parse(asOf) - Date.parse(i.dueDate)) / 86_400_000) : 0);
  let list = res.invoices.filter((i) => i.status !== "batal");
  if (f.age) {
    list = list.filter((i) => left(i) > 0 && receivableBucket(late(i)) === f.age);
  }
  const amount = (i: Invoice) => (f.age ? left(i) : Number(i.amount));
  return {
    summary: [
      { label: t.invoices, value: formatNumber(list.length) },
      { label: f.age ? t.left : t.value, value: formatRupiah(sum(list.map(amount))) },
    ],
    items: list.map((i) => ({
      id: i.id,
      title: i.invoiceNumber || i.label,
      subtitle: [i.clientName, i.dealTitle].filter(Boolean).join(" · "),
      value: rpCompact(amount(i)),
      meta: f.age ? (late(i) > 0 ? t.late(late(i)) : formatDate(i.dueDate)) : formatDate(i.sentAt ?? i.createdAt),
      href: `/erp/sales/pipeline/${i.dealId}`,
    })),
  };
}

/** Mirrors domain.ReceivableBucket. */
export function receivableBucket(daysOverdue: number): string {
  if (daysOverdue <= 0) return "current";
  if (daysOverdue <= 30) return "1-30";
  if (daysOverdue <= 60) return "31-60";
  if (daysOverdue <= 90) return "61-90";
  return "90+";
}

async function loadChats(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const res = await erpClientFetch<{ conversations: Conversation[] }>(`/v1/sales/conversations?${encode({ channel: f.channel, limit: 500 })}`, { signal });
  const from = f.from ? Date.parse(`${f.from}T00:00:00+07:00`) : -Infinity;
  const to = f.to ? Date.parse(`${f.to}T23:59:59+07:00`) : Infinity;
  const list = res.conversations.filter((c) => {
    const at = Date.parse(c.createdAt ?? "");
    return at >= from && at <= to && (!f.owner || c.assignedUserId === f.owner);
  });
  const responded = list.filter((c) => c.firstResponseSeconds > 0);
  return {
    summary: [
      { label: t.chats, value: formatNumber(list.length) },
      { label: t.avgResponse, value: responded.length ? dd.minutes(formatNumber(sum(responded.map((c) => c.firstResponseSeconds)) / responded.length / 60, 1)) : "–" },
    ],
    items: list.map((c) => ({
      id: c.id,
      title: c.displayName || c.phone || c.email || c.id,
      subtitle: dd.leads.channel[c.channel] ?? c.channel,
      value: c.firstResponseSeconds > 0 ? dd.minutes(formatNumber(c.firstResponseSeconds / 60, 1)) : "–",
      meta: c.slaResponseBreached ? t.breached : formatDate(c.createdAt),
      href: `/erp/sales/chatbot?c=${c.id}`,
    })),
  };
}

const LOADERS: Record<Kind, (f: Record<string, string>, s: AbortSignal) => Promise<DrillResult>> = {
  deal: loadDeals,
  lead: loadLeads,
  invoice: loadInvoices,
  chat: loadChats,
};

function chips(f: Record<string, string>): string[] {
  const out: string[] = [];
  if (f.from && f.to) out.push(f.from.slice(0, 7) === f.to.slice(0, 7) && f.from.endsWith("-01") ? bucketLabel(f.from, "month") : `${formatDate(f.from)} – ${formatDate(f.to)}`);
  if (f.line) out.push(SERVICE_LINE[f.line] ?? f.line);
  if (f.source) out.push(LEAD_SOURCE[f.source] ?? f.source);
  if (f.status && t.status[f.status]) out.push(t.status[f.status]!);
  if (f.owner) out.push(t.ownerFiltered);
  return out;
}

/**
 * The one drill sheet of the sales dashboard; mount it once per tab.
 * Actions offer the in-dashboard drill (filter by the mark's dimension or
 * period) next to the record list.
 */
export function SalesDrillSheet({ query }: { query: DashboardQuery }) {
  const { hrefWith } = useDrillParams();
  const { hrefFor } = useDrill();
  const actions = (target: DrillTarget): DrillAction[] => {
    const f = read(target);
    const out: DrillAction[] = [];
    const scope = (patch: Record<string, string>) => hrefWith({ ...patch, drill: null }, ["drill"]);
    const wholePeriod = f.from === query.from && f.to === query.to;
    if (f.from && f.to && !wholePeriod && f.dim !== "none") out.push({ label: t.focusPeriod, href: scope({ preset: "custom", from: f.from, to: f.to }) });
    if (f.dim === "line" && f.line) out.push({ label: t.focusDim(SERVICE_LINE[f.line] ?? f.line), href: scope({ line: f.line }) });
    if (f.dim === "source" && f.source) out.push({ label: t.focusDim(LEAD_SOURCE[f.source] ?? f.source), href: scope({ source: f.source }) });
    if (f.dim === "owner" && f.owner) out.push({ label: t.focusDim(f.l ?? ""), href: scope({ owner: f.owner }) });
    if (target.kind === "deal" && f.status === "won" && f.date === "closed") {
      out.push({ label: t.seeLost, href: hrefFor("deal", encode({ ...f, status: "lost", l: f.l ? `${f.l} · ${t.status.lost}` : undefined })) });
    }
    return out;
  };
  return (
    <DrillSheet
      kind={KINDS}
      title={(target) => read(target).l || t.title[target.kind as Kind]}
      description={(target) => t.title[target.kind as Kind]}
      chips={(target) => chips(read(target))}
      actions={actions}
      load={(target, signal) => LOADERS[target.kind as Kind](read(target), signal)}
      fullListHref={(target) => listHref(target.kind as Kind, cleanForList(target.kind as Kind, read(target)))}
    />
  );
}

/** Keeps only the params the list page reads. */
function cleanForList(kind: Kind, f: Record<string, string>): DrillFilter {
  if (kind === "invoice" && f.age) return { status: "terkirim" };
  return { ...f, l: undefined, dim: undefined, age: undefined };
}
