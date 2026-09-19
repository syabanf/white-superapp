"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Check, ListPlus } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { formatCompact, formatCurrency, formatDateTime, formatNumber, truncate } from "@/lib/format";
import type { KeywordIdea, KeywordIntentKey } from "@/lib/providers/dataforseo/types";
import { t } from "@/i18n/id";
import { trackKeywords } from "@/features/seo-suite/actions";
import { DEFAULT_IDEA_FILTERS, INTENT_LABELS, filterIdeas, summarizeIdeas, type IdeaFilters } from "@/features/seo-suite/lib";
import { s } from "@/features/seo-suite/strings";
import { DifficultyBadge, IntentBadge, SerpFeatureChips, TrendCell } from "./badges";

const right: ColMeta = { align: "right" };
const INTENTS: KeywordIntentKey[] = ["INFORMATIONAL", "NAVIGATIONAL", "COMMERCIAL", "TRANSACTIONAL"];

type Row = KeywordIdea & { tracked: boolean };

export function ResearchResults({
  clientId,
  propertyId,
  ideas,
  tracked,
  fetchedAt,
  fromCache,
  canManage,
  exportName,
}: {
  clientId: string;
  propertyId: string;
  ideas: KeywordIdea[];
  tracked: string[];
  fetchedAt: string;
  fromCache: boolean;
  canManage: boolean;
  exportName: string;
}) {
  const router = useRouter();
  const [filters, setFilters] = React.useState<IdeaFilters>(DEFAULT_IDEA_FILTERS);
  const [selected, setSelected] = React.useState<Set<string>>(() => new Set());
  const [pending, startTransition] = React.useTransition();
  const trackedSet = React.useMemo(() => new Set(tracked), [tracked]);

  const rows = React.useMemo<Row[]>(() => filterIdeas(ideas, filters).map((i) => ({ ...i, tracked: trackedSet.has(i.keyword.toLowerCase()) })), [ideas, filters, trackedSet]);
  const summary = summarizeIdeas(rows);
  const selectable = rows.filter((r) => !r.tracked);
  const allVisibleSelected = selectable.length > 0 && selectable.every((r) => selected.has(r.keyword));

  const toggle = (keyword: string, on: boolean) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (on) next.add(keyword);
      else next.delete(keyword);
      return next;
    });

  const track = () => {
    const picked = ideas.filter((i) => selected.has(i.keyword) && !trackedSet.has(i.keyword.toLowerCase()));
    if (picked.length === 0) {
      toast.info(s.trackedNone);
      return;
    }
    startTransition(async () => {
      const res = await trackKeywords({
        clientId,
        propertyId,
        device: "MOBILE",
        tags: [],
        ideas: picked.map((i) => ({ keyword: i.keyword, volume: i.volume, difficulty: i.difficulty, cpc: i.cpc, intent: i.intent })),
      });
      if (res.ok) {
        toast.success(`${res.data.added} ${s.trackedAdded}`);
        setSelected(new Set());
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const columns: ColumnDef<Row, unknown>[] = [
    ...(canManage
      ? [
          {
            id: "select",
            enableSorting: false,
            header: () => (
              <Checkbox
                aria-label={s.selectAllVisible}
                checked={allVisibleSelected}
                onCheckedChange={(v) =>
                  setSelected((prev) => {
                    const next = new Set(prev);
                    for (const r of selectable) if (v) next.add(r.keyword);
                    else next.delete(r.keyword);
                    return next;
                  })
                }
              />
            ),
            cell: ({ row }) =>
              row.original.tracked ? (
                <span className="inline-flex items-center gap-1 text-[10px] text-muted-foreground" title={s.tracked}>
                  <Check className="size-3" /> {s.tracked}
                </span>
              ) : (
                <Checkbox aria-label={row.original.keyword} checked={selected.has(row.original.keyword)} onCheckedChange={(v) => toggle(row.original.keyword, Boolean(v))} />
              ),
          } satisfies ColumnDef<Row, unknown>,
        ]
      : []),
    { accessorKey: "keyword", header: t.seo.query, cell: ({ row }) => <span title={row.original.keyword}>{truncate(row.original.keyword, 48)}</span> },
    { accessorKey: "volume", header: s.volume, meta: right, cell: ({ row }) => <span className="tabular font-medium">{formatNumber(row.original.volume)}</span> },
    { id: "trend", header: s.trend12, enableSorting: false, cell: ({ row }) => <TrendCell values={row.original.trend} /> },
    { accessorKey: "difficulty", header: s.kd, meta: right, cell: ({ row }) => <DifficultyBadge kd={row.original.difficulty} /> },
    { accessorKey: "cpc", header: s.cpc, meta: right, cell: ({ row }) => <span className="tabular">{formatCurrency(row.original.cpc, "IDR")}</span> },
    { accessorKey: "intent", header: s.intent, cell: ({ row }) => <IntentBadge intent={row.original.intent} /> },
    { id: "serp", header: s.serpFeatures, enableSorting: false, cell: ({ row }) => <SerpFeatureChips features={row.original.serpFeatures} /> },
  ];

  return (
    <div className="flex flex-col gap-4">
      <StatStrip
        items={[
          { label: s.ideas, value: formatNumber(summary.count) },
          { label: s.totalVolume, value: formatCompact(summary.totalVolume), hint: s.volumeHint },
          { label: s.avgKd, value: formatNumber(summary.avgKd), hint: s.kdHint },
        ]}
      />
      <Card className="gap-0 py-0">
        <CardContent className="space-y-4 p-6">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
            <div className="space-y-1.5">
              <Label htmlFor="f-vol" className="label-mono">
                {s.minVolume}
              </Label>
              <Input id="f-vol" type="number" min={0} inputMode="numeric" value={filters.minVolume || ""} onChange={(e) => setFilters({ ...filters, minVolume: Number(e.target.value) || 0 })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="f-kd" className="label-mono">
                {s.maxKd}
              </Label>
              <Input id="f-kd" type="number" min={0} max={100} inputMode="numeric" value={filters.maxKd} onChange={(e) => setFilters({ ...filters, maxKd: Math.min(100, Math.max(0, Number(e.target.value) || 0)) })} />
            </div>
            <div className="space-y-1.5">
              <Label className="label-mono">{s.intent}</Label>
              <Select value={filters.intent} onValueChange={(v) => setFilters({ ...filters, intent: v as IdeaFilters["intent"] })}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{s.anyIntent}</SelectItem>
                  {INTENTS.map((i) => (
                    <SelectItem key={i} value={i}>
                      {INTENT_LABELS[i]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="f-inc" className="label-mono">
                {s.includeWords}
              </Label>
              <Input id="f-inc" value={filters.include} placeholder={s.filterPlaceholderInclude} onChange={(e) => setFilters({ ...filters, include: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="f-exc" className="label-mono">
                {s.excludeWords}
              </Label>
              <Input id="f-exc" value={filters.exclude} placeholder={s.filterPlaceholderExclude} onChange={(e) => setFilters({ ...filters, exclude: e.target.value })} />
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
            <span>
              {fromCache ? `${s.cachedAt} · ` : ""}
              {s.lastUpdated} {formatDateTime(fetchedAt)} · {s.cacheNote}
            </span>
            {canManage ? (
              <div className="flex items-center gap-2">
                {selected.size > 0 ? (
                  <>
                    <span className="tabular">
                      {formatNumber(selected.size)} {s.selected}
                    </span>
                    <Button variant="ghost" size="xs" onClick={() => setSelected(new Set())}>
                      {s.clearSelection}
                    </Button>
                  </>
                ) : null}
                <Button size="sm" onClick={track} disabled={pending || selected.size === 0}>
                  {pending ? <Spinner className="size-4" /> : <ListPlus className="size-4" />}
                  {s.trackSelected}
                </Button>
              </div>
            ) : null}
          </div>

          <DataTable
            data={rows}
            columns={columns}
            searchKey="keyword"
            searchPlaceholder={t.common.search}
            exportName={exportName}
            initialSorting={[{ id: "volume", desc: true }]}
            pageSize={20}
            emptyMessage={s.researchNoResults}
            getRowId={(r) => r.keyword}
          />
        </CardContent>
      </Card>
    </div>
  );
}
