"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { MoreHorizontal, Smartphone, Monitor, Tag, Trash2 } from "lucide-react";
import type { ColumnDef } from "@tanstack/react-table";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { Sparkline } from "@/components/dashboard/sparkline";
import { formatNumber, truncate } from "@/lib/format";
import { t } from "@/i18n/id";
import { deleteTrackedKeyword, updateKeywordTags } from "@/features/seo-suite/actions";
import type { RankKeywordRow } from "@/features/seo-suite/queries";
import { s } from "@/features/seo-suite/strings";
import { DifficultyBadge, SerpFeatureChips } from "./badges";

const right: ColMeta = { align: "right" };

function DeltaCell({ change }: { change: RankKeywordRow["change"] }) {
  if (change.kind === "new") return <span className="text-xs font-medium text-positive">{s.improved}</span>;
  if (change.kind === "lost") return <span className="text-xs font-medium text-negative">{s.declinedKw}</span>;
  if (change.delta == null || change.kind === "none") return <span className="text-muted-foreground">–</span>;
  if (change.delta === 0) return <span className="tabular text-muted-foreground">0</span>;
  const up = change.delta > 0;
  return <span className={`tabular text-xs font-medium ${up ? "text-positive" : "text-negative"}`}>{up ? `▲ ${change.delta}` : `▼ ${Math.abs(change.delta)}`}</span>;
}

function RowMenu({ row, clientId, canManage }: { row: RankKeywordRow; clientId: string; canManage: boolean }) {
  const router = useRouter();
  const [pending, startTransition] = React.useTransition();
  const [editing, setEditing] = React.useState(false);
  const [tags, setTags] = React.useState(row.tags.join(", "));
  if (!canManage) return null;

  const remove = () =>
    startTransition(async () => {
      const res = await deleteTrackedKeyword({ clientId, keywordId: row.id });
      if (res.ok) {
        toast.success(s.keywordDeleted, { description: row.keyword });
        router.refresh();
      } else toast.error(res.error);
    });
  const saveTags = () =>
    startTransition(async () => {
      const res = await updateKeywordTags({ clientId, keywordId: row.id, tags });
      if (res.ok) {
        toast.success(s.tagsSaved);
        setEditing(false);
        router.refresh();
      } else toast.error(res.error);
    });

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon-xs" aria-label={t.common.actions} disabled={pending}>
            {pending ? <Spinner className="size-3.5" /> : <MoreHorizontal className="size-3.5" />}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onSelect={() => setEditing(true)}>
            <Tag className="size-4" /> {s.editTags}
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={remove}>
            <Trash2 className="size-4" /> {s.deleteKeyword}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <Dialog open={editing} onOpenChange={setEditing}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{s.editTags}</DialogTitle>
            <DialogDescription>{row.keyword}</DialogDescription>
          </DialogHeader>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              saveTags();
            }}
            className="space-y-4"
          >
            <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder={s.tagsPlaceholder} autoFocus />
            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => setEditing(false)}>
                {t.common.cancel}
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? <Spinner className="size-4" /> : null}
                {t.common.save}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}

export function RankTable({ rows, tags, clientId, canManage, toolbar }: { rows: RankKeywordRow[]; tags: string[]; clientId: string; canManage: boolean; toolbar?: React.ReactNode }) {
  const [tag, setTag] = React.useState<string>("all");
  const filtered = React.useMemo(() => (tag === "all" ? rows : tag === "__none" ? rows.filter((r) => r.tags.length === 0) : rows.filter((r) => r.tags.includes(tag))), [rows, tag]);

  const columns: ColumnDef<RankKeywordRow, unknown>[] = [
    {
      accessorKey: "keyword",
      header: t.seo.query,
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col gap-0.5">
          <span title={row.original.keyword}>{truncate(row.original.keyword, 44)}</span>
          {row.original.tags.length ? (
            <span className="flex flex-wrap gap-1">
              {row.original.tags.map((tg) => (
                <Badge key={tg} variant="outline" className="px-1.5 py-0 text-[10px] font-normal text-muted-foreground">
                  {tg}
                </Badge>
              ))}
            </span>
          ) : null}
        </div>
      ),
    },
    {
      accessorKey: "device",
      header: s.device,
      cell: ({ row }) => (
        <span className="inline-flex items-center gap-1 text-xs text-muted-foreground" title={row.original.device === "MOBILE" ? t.seo.mobile : t.seo.desktop}>
          {row.original.device === "MOBILE" ? <Smartphone className="size-3.5" /> : <Monitor className="size-3.5" />}
        </span>
      ),
    },
    {
      accessorKey: "position",
      header: s.positionCol,
      meta: right,
      sortingFn: (a, b) => (a.original.position ?? 999) - (b.original.position ?? 999),
      cell: ({ row }) => (row.original.position == null ? <span className="text-muted-foreground">–</span> : <span className="tabular text-base font-semibold">{formatNumber(row.original.position)}</span>),
    },
    { id: "delta", header: s.deltaCol, meta: right, accessorFn: (r) => r.change.delta ?? -999, cell: ({ row }) => <DeltaCell change={row.original.change} /> },
    { accessorKey: "best", header: s.bestCol, meta: right, cell: ({ row }) => (row.original.best == null ? "–" : <span className="tabular">{formatNumber(row.original.best)}</span>) },
    {
      accessorKey: "url",
      header: s.urlCol,
      enableSorting: false,
      cell: ({ row }) =>
        row.original.url ? (
          <a href={row.original.url} target="_blank" rel="noreferrer" className="text-xs text-muted-foreground underline-offset-2 hover:underline" title={row.original.url}>
            {truncate(row.original.url.replace(/^https?:\/\/[^/]+/, "") || "/", 28)}
          </a>
        ) : (
          <span className="text-muted-foreground">–</span>
        ),
    },
    { accessorKey: "volume", header: s.volume, meta: right, cell: ({ row }) => (row.original.volume == null ? "–" : <span className="tabular">{formatNumber(row.original.volume)}</span>) },
    { accessorKey: "difficulty", header: s.kd, meta: right, cell: ({ row }) => <DifficultyBadge kd={row.original.difficulty} /> },
    { id: "serp", header: s.serpFeatures, enableSorting: false, cell: ({ row }) => <SerpFeatureChips features={row.original.serpFeatures} max={2} /> },
    { id: "spark", header: s.history30, enableSorting: false, cell: ({ row }) => <Sparkline values={row.original.spark} accent="var(--chart-1)" /> },
    ...(canManage ? [{ id: "menu", header: "", enableSorting: false, cell: ({ row }) => <RowMenu row={row.original} clientId={clientId} canManage={canManage} /> } satisfies ColumnDef<RankKeywordRow, unknown>] : []),
  ];

  return (
    <Card className="gap-0 py-0">
      <CardContent className="p-6">
        <DataTable
          data={filtered}
          columns={columns}
          searchKey="keyword"
          searchPlaceholder={t.common.search}
          exportName="peringkat"
          initialSorting={[{ id: "position", desc: false }]}
          pageSize={20}
          emptyMessage={t.common.noResults}
          getRowId={(r) => r.id}
          toolbar={
            <div className="flex flex-wrap items-center gap-2">
              <Select value={tag} onValueChange={setTag}>
                <SelectTrigger size="sm" className="min-w-36">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{s.allTags}</SelectItem>
                  <SelectItem value="__none">{s.noTag}</SelectItem>
                  {tags.map((tg) => (
                    <SelectItem key={tg} value={tg}>
                      {tg}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {toolbar}
            </div>
          }
        />
      </CardContent>
    </Card>
  );
}
