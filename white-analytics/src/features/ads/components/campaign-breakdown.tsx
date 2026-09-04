"use client";

import * as React from "react";
import { ArrowDown, ArrowUp, ArrowUpDown, ChevronRight, Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatCurrency, formatNumber, formatPercent } from "@/lib/format";
import { isFatigued } from "@/lib/metrics";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";
import { objectiveLabel, resultTypeLabel, s } from "@/features/ads/strings";
import type { AdsCampaignNode, AdsNodeKpis } from "@/features/ads/queries";

type SortKey = "spend" | "cpr";
type SortDir = "asc" | "desc";

function statusKind(status: string): "good" | "neutral" {
  return status === "ACTIVE" ? "good" : "neutral";
}

function statusLabel(status: string): string {
  if (status === "ACTIVE") return t.common.active;
  if (status === "PAUSED") return t.common.paused;
  if (status === "ARCHIVED") return t.common.archived;
  return status;
}

/** Nilai urut: CPR 0 karena tanpa hasil → paling akhir saat ascending. */
function sortValue(k: AdsNodeKpis, key: SortKey): number {
  if (key === "spend") return k.spend;
  return k.results > 0 ? k.cpr : Number.POSITIVE_INFINITY;
}

function csvEscape(v: string): string {
  return /[",\n]/.test(v) ? `"${v.replace(/"/g, '""')}"` : v;
}

/**
 * Tabel drill-down "Rincian kampanye": kampanye → set iklan → iklan.
 * Bisa diurutkan berdasar belanja / biaya per hasil; ekspor CSV (rata).
 */
export function CampaignBreakdown({ campaigns, currency }: { campaigns: AdsCampaignNode[]; currency: string }) {
  const [openCampaigns, setOpenCampaigns] = React.useState<ReadonlySet<string>>(new Set());
  const [openAdSets, setOpenAdSets] = React.useState<ReadonlySet<string>>(new Set());
  const [sortKey, setSortKey] = React.useState<SortKey>("spend");
  const [sortDir, setSortDir] = React.useState<SortDir>("desc");

  const toggle = (set: ReadonlySet<string>, id: string): Set<string> => {
    const next = new Set(set);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    return next;
  };

  const onSort = (key: SortKey) => {
    if (sortKey === key) setSortDir((d) => (d === "desc" ? "asc" : "desc"));
    else {
      setSortKey(key);
      setSortDir(key === "cpr" ? "asc" : "desc");
    }
  };

  const cmp = React.useCallback(
    (a: { kpis: AdsNodeKpis }, b: { kpis: AdsNodeKpis }) => {
      const va = sortValue(a.kpis, sortKey);
      const vb = sortValue(b.kpis, sortKey);
      return sortDir === "asc" ? va - vb : vb - va;
    },
    [sortKey, sortDir],
  );

  const sorted = React.useMemo(() => {
    return [...campaigns].sort(cmp).map((c) => ({
      ...c,
      adSets: [...c.adSets].sort(cmp).map((st) => ({ ...st, ads: [...st.ads].sort(cmp) })),
    }));
  }, [campaigns, cmp]);

  const exportCsv = () => {
    const header = [
      "level",
      t.ads.campaign,
      t.ads.adSet,
      t.ads.ad,
      t.common.status,
      t.ads.objective,
      t.ads.spend,
      t.ads.results,
      t.ads.resultType,
      t.ads.cpr,
      t.ads.impressions,
      t.ads.reach,
      t.ads.linkClicks,
      t.ads.ctr,
      t.ads.cpc,
      t.ads.frequency,
    ];
    const lines: string[] = [header.map(csvEscape).join(",")];
    const push = (level: string, names: [string, string, string], status: string, objective: string, k: AdsNodeKpis) => {
      lines.push(
        [
          level,
          ...names,
          status,
          objective,
          String(k.spend),
          String(k.results),
          k.resultType ?? "",
          k.cpr.toFixed(2),
          String(k.impressions),
          String(k.reach),
          String(k.linkClicks),
          k.ctr.toFixed(4),
          k.cpc.toFixed(2),
          k.frequency.toFixed(2),
        ]
          .map(csvEscape)
          .join(","),
      );
    };
    for (const c of sorted) {
      push("campaign", [c.name, "", ""], c.status, c.objective, c.kpis);
      for (const st of c.adSets) {
        push("adset", [c.name, st.name, ""], st.status, c.objective, st.kpis);
        for (const ad of st.ads) push("ad", [c.name, st.name, ad.name], ad.status, c.objective, ad.kpis);
      }
    }
    const blob = new Blob(["﻿" + lines.join("\n")], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "rincian-kampanye.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  if (campaigns.length === 0) {
    return <p className="rounded-lg border border-dashed py-10 text-center text-sm text-muted-foreground">{s.breakdownEmpty}</p>;
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-end">
        <Button variant="outline" size="sm" onClick={exportCsv}>
          <Download className="size-3.5" /> {t.common.exportCsv}
        </Button>
      </div>
      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="overflow-x-auto scrollbar-thin">
          <Table className="[&_td]:py-2 [&_th]:h-9">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="min-w-56 text-xs font-medium text-muted-foreground">{t.common.name}</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground">{t.common.status}</TableHead>
                <TableHead className="text-xs font-medium text-muted-foreground">{t.ads.objective}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">
                  <SortButton label={t.ads.spend} k="spend" sortKey={sortKey} sortDir={sortDir} onSort={onSort} />
                </TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.results}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">
                  <SortButton label={t.ads.cpr} k="cpr" sortKey={sortKey} sortDir={sortDir} onSort={onSort} />
                </TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.impressions}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.reach}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.linkClicks}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.ctr}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.cpc}</TableHead>
                <TableHead className="text-right text-xs font-medium text-muted-foreground">{t.ads.frequency}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sorted.map((c) => {
                const cOpen = openCampaigns.has(c.id);
                return (
                  <React.Fragment key={c.id}>
                    <TableRow className="cursor-pointer" onClick={() => setOpenCampaigns((prev) => toggle(prev, c.id))}>
                      <TableCell className="text-sm font-medium">
                        <span className="flex items-center gap-1.5">
                          <button
                            type="button"
                            aria-expanded={cOpen}
                            aria-label={`${cOpen ? s.collapseRow : s.expandRow}: ${c.name}`}
                            className="rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                            onClick={(e) => {
                              e.stopPropagation();
                              setOpenCampaigns((prev) => toggle(prev, c.id));
                            }}
                          >
                            <ChevronRight className={cn("size-4 transition-transform", cOpen && "rotate-90")} />
                          </button>
                          <span className="max-w-72 truncate" title={c.name}>
                            {c.name}
                          </span>
                        </span>
                      </TableCell>
                      <TableCell>
                        <StatusBadge kind={statusKind(c.status)}>{statusLabel(c.status)}</StatusBadge>
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">{objectiveLabel(c.objective)}</TableCell>
                      <MetricCells kpis={c.kpis} currency={currency} />
                    </TableRow>
                    {cOpen
                      ? c.adSets.map((st) => {
                          const sOpen = openAdSets.has(st.id);
                          return (
                            <React.Fragment key={st.id}>
                              <TableRow className="bg-muted/30">
                                <TableCell className="text-sm">
                                  <span className="flex items-center gap-1.5 pl-5">
                                    <button
                                      type="button"
                                      aria-expanded={sOpen}
                                      aria-label={`${sOpen ? s.collapseRow : s.expandRow}: ${st.name}`}
                                      className="rounded p-0.5 text-muted-foreground hover:bg-muted hover:text-foreground"
                                      onClick={() => setOpenAdSets((prev) => toggle(prev, st.id))}
                                    >
                                      <ChevronRight className={cn("size-4 transition-transform", sOpen && "rotate-90")} />
                                    </button>
                                    <span className="max-w-64 truncate" title={st.name}>
                                      {st.name}
                                    </span>
                                  </span>
                                </TableCell>
                                <TableCell>
                                  <StatusBadge kind={statusKind(st.status)}>{statusLabel(st.status)}</StatusBadge>
                                </TableCell>
                                <TableCell className="text-xs text-muted-foreground">{t.ads.adSet}</TableCell>
                                <MetricCells kpis={st.kpis} currency={currency} />
                              </TableRow>
                              {sOpen
                                ? st.ads.map((ad) => (
                                    <TableRow key={ad.id} className="bg-muted/50">
                                      <TableCell className="text-sm">
                                        <span className="block max-w-64 truncate pl-12" title={ad.name}>
                                          {ad.name}
                                        </span>
                                      </TableCell>
                                      <TableCell>
                                        <StatusBadge kind={statusKind(ad.status)}>{statusLabel(ad.status)}</StatusBadge>
                                      </TableCell>
                                      <TableCell className="text-xs text-muted-foreground">{t.ads.ad}</TableCell>
                                      <MetricCells kpis={ad.kpis} currency={currency} />
                                    </TableRow>
                                  ))
                                : null}
                            </React.Fragment>
                          );
                        })
                      : null}
                  </React.Fragment>
                );
              })}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  );
}

function SortButton({
  label,
  k,
  sortKey,
  sortDir,
  onSort,
}: {
  label: string;
  k: SortKey;
  sortKey: SortKey;
  sortDir: SortDir;
  onSort: (key: SortKey) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSort(k)}
      className="inline-flex flex-row-reverse items-center gap-1 hover:text-foreground"
      aria-label={k === "spend" ? s.sortBySpend : s.sortByCpr}
    >
      {label}
      {sortKey === k ? (
        sortDir === "asc" ? (
          <ArrowUp className="size-3" />
        ) : (
          <ArrowDown className="size-3" />
        )
      ) : (
        <ArrowUpDown className="size-3 opacity-40" />
      )}
    </button>
  );
}

function MetricCells({ kpis: k, currency }: { kpis: AdsNodeKpis; currency: string }) {
  const empty = k.spend === 0 && k.impressions === 0;
  const dash = <span className="text-muted-foreground">–</span>;
  return (
    <>
      <TableCell className="text-right text-sm tabular">{empty ? dash : formatCurrency(k.spend, currency)}</TableCell>
      <TableCell className="text-right text-sm tabular">
        {empty ? (
          dash
        ) : (
          <>
            {formatNumber(k.results)}
            {k.resultType ? <span className="ml-1 text-[11px] text-muted-foreground">{resultTypeLabel(k.resultType)}</span> : null}
          </>
        )}
      </TableCell>
      <TableCell className="text-right text-sm tabular">{empty || k.results === 0 ? dash : formatCurrency(k.cpr, currency)}</TableCell>
      <TableCell className="text-right text-sm tabular">{empty ? dash : formatNumber(k.impressions)}</TableCell>
      <TableCell className="text-right text-sm tabular">{empty ? dash : formatNumber(k.reach)}</TableCell>
      <TableCell className="text-right text-sm tabular">{empty ? dash : formatNumber(k.linkClicks)}</TableCell>
      <TableCell className="text-right text-sm tabular">{empty ? dash : formatPercent(k.ctr)}</TableCell>
      <TableCell className="text-right text-sm tabular">{empty || k.linkClicks === 0 ? dash : formatCurrency(k.cpc, currency)}</TableCell>
      <TableCell className="text-right text-sm tabular">
        {empty ? (
          dash
        ) : (
          <span className="inline-flex items-center gap-1.5">
            {formatNumber(k.frequency, 1)}
            {isFatigued(k.frequency) ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <StatusBadge kind="warning" className="px-1.5 py-0 text-[10px]">
                      {s.fatigueBadge}
                    </StatusBadge>
                  </span>
                </TooltipTrigger>
                <TooltipContent className="text-xs">{t.ads.fatigue}</TooltipContent>
              </Tooltip>
            ) : null}
          </span>
        )}
      </TableCell>
    </>
  );
}
