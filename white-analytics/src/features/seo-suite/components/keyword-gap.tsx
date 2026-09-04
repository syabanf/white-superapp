"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { EmptyState } from "@/components/dashboard/empty-state";
import { formatNumber, truncate } from "@/lib/format";
import type { GapRow, KeywordGap } from "@/lib/metrics/seo-suite";
import { t } from "@/i18n/id";
import { s } from "@/features/seo-suite/strings";

const right: ColMeta = { align: "right" };
type GapKey = keyof KeywordGap;
const KEYS: GapKey[] = ["missing", "weak", "strong", "shared"];
const LABELS: Record<GapKey, string> = {
  missing: s.gapMissing,
  weak: s.gapWeak,
  strong: s.gapStrong,
  shared: s.gapShared,
};
const DESCS: Record<GapKey, string> = {
  missing: s.gapMissingDesc,
  weak: s.gapWeakDesc,
  strong: s.gapStrongDesc,
  shared: s.gapSharedDesc,
};

function Pos({ v }: { v: number | null }) {
  return v == null ? <span className="text-muted-foreground">–</span> : <span className="tabular">{formatNumber(v)}</span>;
}

export function KeywordGapSection({
  competitors,
  selected,
  ownDomain,
  gap,
  competitorsHref,
}: {
  competitors: { id: string; domain: string }[];
  selected: string | null;
  ownDomain: string;
  gap: KeywordGap | null;
  competitorsHref: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  const [view, setView] = React.useState<GapKey>("missing");

  const pick = (domain: string) => {
    const q = new URLSearchParams(sp.toString());
    q.set("gap", domain);
    router.push(`${pathname}?${q.toString()}#gap`, { scroll: false });
  };

  if (competitors.length === 0) {
    return (
      <EmptyState
        compact
        title={s.gapNoCompetitors}
        action={
          <Button asChild size="sm" variant="outline">
            <Link href={competitorsHref}>{s.gapManageCompetitors}</Link>
          </Button>
        }
      />
    );
  }

  const columns: ColumnDef<GapRow, unknown>[] = [
    {
      accessorKey: "keyword",
      header: t.seo.query,
      cell: ({ row }) => <span title={row.original.keyword}>{truncate(row.original.keyword, 48)}</span>,
    },
    {
      accessorKey: "volume",
      header: s.volume,
      meta: right,
      cell: ({ row }) => <span className="tabular font-medium">{formatNumber(row.original.volume)}</span>,
    },
    {
      accessorKey: "ownPosition",
      header: `${s.ownPosition} (${ownDomain})`,
      meta: right,
      cell: ({ row }) => <Pos v={row.original.ownPosition} />,
    },
    {
      accessorKey: "competitorPosition",
      header: `${s.competitorPosition}${selected ? ` (${selected})` : ""}`,
      meta: right,
      cell: ({ row }) => <Pos v={row.original.competitorPosition} />,
    },
  ];

  return (
    <Card className="gap-0 py-0" id="gap">
      <CardContent className="space-y-4 p-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <span className="label-mono">{s.gapPickCompetitor}</span>
            <Select value={selected ?? ""} onValueChange={pick}>
              <SelectTrigger size="sm" className="min-w-44">
                <SelectValue placeholder={s.gapPickCompetitor} />
              </SelectTrigger>
              <SelectContent>
                {competitors.map((c) => (
                  <SelectItem key={c.id} value={c.domain}>
                    {c.domain}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {gap ? (
            <div className="max-w-full overflow-x-auto scrollbar-thin">
              <Tabs value={view} onValueChange={(v) => setView(v as GapKey)}>
                <TabsList>
                  {KEYS.map((k) => (
                    <TabsTrigger key={k} value={k}>
                      {LABELS[k]} <span className="tabular text-[10px] text-muted-foreground">{formatNumber(gap[k].length)}</span>
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
          ) : null}
        </div>
        {gap ? (
          <>
            <p className="text-sm text-muted-foreground">{DESCS[view]}</p>
            <DataTable
              data={gap[view]}
              columns={columns}
              searchKey="keyword"
              exportName={`gap-${view}-${selected ?? "x"}`}
              initialSorting={[{ id: "volume", desc: true }]}
              pageSize={10}
              emptyMessage={t.common.noResults}
            />
          </>
        ) : (
          <p className="py-6 text-center text-sm text-muted-foreground">{s.gapPickCompetitor}</p>
        )}
      </CardContent>
    </Card>
  );
}
