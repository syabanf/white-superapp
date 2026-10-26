"use client";

import * as React from "react";
import { DrillSheet, useDrill, useDrillParams, type DrillAction, type DrillResult, type DrillTarget } from "@/features/erp/shared/drill";
import { erpClientFetch } from "@/features/erp/shared/client";
import { formatRupiah } from "@/features/erp/shared/money";
import { formatDate, formatDecimal, formatNumber } from "@/lib/format";
import { candidateHref, candidatesHref, type DrillContext } from "../../analytics";
import { CANDIDATE_STATUS, OFFER_STATUS, SOURCE, ra } from "../../strings";
import type { DashboardOffer, ListCandidatesResponse } from "../../types";

/**
 * Drill-down of the recruitment dashboard (`?drill=<kind>:<filters>`).
 * cand:  status job source from to (application days)   -> candidates list
 * offer: status bucket(0-3 response bucket) from to (sent days) -> offers already on the page
 * `l` carries the mark's label.
 */
type Filter = Record<string, string | number | undefined | null>;

const d = ra.drill;

function encode(f: Filter): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(f)) if (v != null && v !== "") p.set(k, String(v));
  return p.toString();
}

const read = (t: DrillTarget) => Object.fromEntries(new URLSearchParams(t.key));

/** Opens a recruitment drill; the dashboard's period and filters carry over. */
export function useRecruitmentDrill(ctx: DrillContext) {
  const { open } = useDrill();
  return React.useCallback(
    (kind: "cand" | "offer", f: Filter) => open(kind, encode({ from: ctx.from, to: ctx.to, job: ctx.job, source: ctx.source, status: kind === "cand" ? ctx.stage : undefined, ...f })),
    [open, ctx],
  );
}

/** Week starting at `start` as an inclusive day range. */
export function weekRange(start: string): { from: string; to: string } {
  const t = new Date(`${start.slice(0, 10)}T00:00:00Z`);
  t.setUTCDate(t.getUTCDate() + 6);
  return { from: start.slice(0, 10), to: t.toISOString().slice(0, 10) };
}

async function loadCandidates(f: Record<string, string>, signal: AbortSignal): Promise<DrillResult> {
  const res = await erpClientFetch<ListCandidatesResponse>(
    `/v1/recruitment/candidates?${encode({ status: f.status, jobOpeningId: f.job, source: f.source, dateFrom: f.from, dateTo: f.to, limit: 100 })}`,
    { signal },
  );
  return {
    total: res.total,
    summary: [
      { label: d.candidates, value: formatNumber(res.total) },
      { label: d.hired, value: formatNumber(res.candidates.filter((c) => c.status === "hired").length) },
    ],
    items: res.candidates.map((c) => ({
      id: c.id,
      title: c.fullName,
      subtitle: [c.jobOpeningTitle, SOURCE[c.source] ?? c.source].filter(Boolean).join(" · "),
      value: CANDIDATE_STATUS[c.status] ?? c.status,
      meta: formatDate(c.createdAt),
      href: candidateHref(c.id),
    })),
  };
}

const BUCKETS = [1, 3, 7, Infinity];
const bucketOf = (days: number) => BUCKETS.findIndex((b) => days <= b);

function offerRows(offers: DashboardOffer[], f: Record<string, string>): DashboardOffer[] {
  return offers.filter((o) => {
    if (f.status && o.status !== f.status) return false;
    if (f.bucket != null && (o.responseDays == null || (o.status !== "accepted" && o.status !== "declined") || bucketOf(o.responseDays) !== Number(f.bucket))) return false;
    const sent = o.sentAt.slice(0, 10);
    if (f.week && (sent < f.from! || sent > f.to!)) return false;
    return true;
  });
}

/** Drill sheets of the recruitment dashboard; offers come from the page data. */
export function RecruitmentDrillSheet({ ctx, offers }: { ctx: DrillContext; offers: DashboardOffer[] }) {
  const { hrefWith } = useDrillParams();
  const actions = (t: DrillTarget): DrillAction[] => {
    const f = read(t);
    const out: DrillAction[] = [];
    if (f.week) out.push({ label: d.focusPeriod, href: hrefWith({ preset: "custom", from: f.from, to: f.to }) });
    if (f.dim === "source" && f.source) out.push({ label: d.focusDim(SOURCE[f.source] ?? f.source), href: hrefWith({ source: f.source, tab: "ringkasan" }) });
    if (f.dim === "job" && f.job) out.push({ label: d.focusDim(f.l ?? ""), href: hrefWith({ job: f.job }) });
    return out;
  };
  const chips = (t: DrillTarget) => {
    const f = read(t);
    return [
      f.from && f.to ? `${formatDate(f.from)} – ${formatDate(f.to)}` : null,
      f.status ? ((t.kind === "offer" ? OFFER_STATUS : CANDIDATE_STATUS)[f.status] ?? f.status) : null,
      f.source ? (SOURCE[f.source] ?? f.source) : null,
    ].filter((c): c is string => c != null);
  };
  return (
    <>
      <DrillSheet
        kind="cand"
        title={(t) => read(t).l || d.candidates}
        description={() => d.candHint}
        chips={chips}
        actions={actions}
        load={(t, signal) => loadCandidates(read(t), signal)}
        fullListHref={(t) => {
          const f = read(t);
          return candidatesHref({ ...ctx, from: f.from ?? ctx.from, to: f.to ?? ctx.to, source: f.source, job: f.job }, { status: f.status, job: f.job });
        }}
      />
      <DrillSheet
        kind="offer"
        title={(t) => read(t).l || d.offers}
        chips={chips}
        actions={actions}
        load={(t) => {
          const rows = offerRows(offers, read(t));
          const resp = rows.filter((o) => o.responseDays != null);
          return {
            summary: [
              { label: d.offers, value: formatNumber(rows.length) },
              { label: d.avgResponse, value: resp.length ? ra.days(formatDecimal(resp.reduce((a, o) => a + o.responseDays!, 0) / resp.length, 1)) : "–" },
            ],
            items: rows.map((o) => ({
              id: `${o.candidateId}-${o.sentAt}`,
              title: o.candidateName,
              subtitle: [o.positionTitle, OFFER_STATUS[o.status] ?? o.status].filter(Boolean).join(" · "),
              value: formatRupiah(o.baseSalary),
              meta: o.responseDays != null ? ra.days(formatDecimal(o.responseDays, 1)) : formatDate(o.sentAt),
              href: candidateHref(o.candidateId, "offers"),
            })),
          };
        }}
        fullListHref={() => candidatesHref(ctx, { status: "offer" })}
      />
    </>
  );
}
