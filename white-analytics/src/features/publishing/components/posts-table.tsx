"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { PLATFORMS, type Platform } from "@/features/publishing/lib";
import type { PostRow } from "@/features/publishing/queries";
import type { PostStatusKey } from "@/lib/metrics/publishing";
import { PostStatusBadge } from "@/features/publishing/components/post-status";
import { clock, toZoned } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { formatDateShort } from "@/lib/format";
import { t } from "@/i18n/id";
import { POSTS_TABS, TAB_STATUSES, tabCount, type PostsTab } from "@/features/publishing/posts-tabs";

/* eslint-disable @next/next/no-img-element */

export function PostsTable({ rows, counts, tab, slug, timezone }: { rows: PostRow[]; counts: Record<PostStatusKey, number>; tab: PostsTab; slug: string; timezone: string }) {
  const router = useRouter();
  const pathname = usePathname();
  const sp = useSearchParams();
  const [platform, setPlatform] = React.useState<Platform | "ALL">("ALL");

  const hrefFor = (tb: PostsTab) => {
    const q = new URLSearchParams(sp.toString());
    if (tb === "ALL") q.delete("status");
    else q.set("status", tb);
    const qs = q.toString();
    return qs ? `${pathname}?${qs}` : pathname;
  };
  const statuses = TAB_STATUSES[tab];
  const data = rows.filter((r) => (!statuses || statuses.includes(r.status)) && (platform === "ALL" || r.platforms.includes(platform)));

  const columns = React.useMemo<ColumnDef<PostRow>[]>(
    () => [
      {
        id: "at",
        accessorKey: "at",
        header: p.colSchedule,
        size: 120,
        cell: ({ row }) => {
          const z = toZoned(new Date(row.original.at), timezone);
          const planned = !row.original.publishedAt && row.original.status !== "SCHEDULED";
          return (
            <span className="tabular whitespace-nowrap text-xs">
              <span className="font-medium">{formatDateShort(new Date(`${z.date}T00:00:00Z`))}</span>
              <span className="text-muted-foreground"> · {clock(row.original.at, timezone)}</span>
              {planned && row.original.scheduledAt ? <span className="label-mono block text-muted-foreground">{p.planned}</span> : null}
            </span>
          );
        },
      },
      {
        id: "search",
        accessorFn: (r) => `${r.title} ${r.body}`,
        header: p.colPost,
        cell: ({ row }) => (
          <span className="flex min-w-0 items-center gap-2.5">
            {row.original.thumbnailUrl ? (
              <img src={row.original.thumbnailUrl} alt="" loading="lazy" className="size-9 shrink-0 rounded-md bg-muted object-cover" />
            ) : (
              <span className="size-9 shrink-0 rounded-md bg-muted" />
            )}
            <span className="min-w-0">
              <span className="block max-w-[38ch] truncate text-[13px] font-medium">{row.original.title || p.untitled}</span>
              <span className="block max-w-[38ch] truncate text-xs text-muted-foreground">{row.original.bodyShort}</span>
            </span>
          </span>
        ),
      },
      {
        id: "platforms",
        accessorFn: (r) => r.platforms.join(","),
        header: p.colTargets,
        size: 90,
        enableSorting: false,
        cell: ({ row }) => (
          <span className="flex items-center gap-1">
            {row.original.platforms.map((pl) => (
              <PlatformIcon key={pl} platform={pl} className="size-4" />
            ))}
          </span>
        ),
      },
      {
        id: "status",
        accessorKey: "status",
        header: p.colStatus,
        size: 150,
        cell: ({ row }) => <PostStatusBadge status={row.original.status} />,
      },
      { id: "author", accessorKey: "author", header: p.colAuthor, size: 130, cell: ({ getValue }) => <span className="text-xs text-muted-foreground">{(getValue() as string | null) ?? "–"}</span> },
    ],
    [timezone],
  );

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto scrollbar-thin">
        <Tabs value={tab}>
          <TabsList aria-label={t.common.status}>
            {POSTS_TABS.map((tb) => (
              <TabsTrigger key={tb} value={tb} asChild>
                <Link href={hrefFor(tb)} scroll={false}>
                  {tb === "ALL" ? t.common.all : p.status[tb]}
                  <span className="tabular ml-1 text-muted-foreground">{tabCount(tb, counts)}</span>
                </Link>
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      <DataTable
        columns={columns}
        data={data}
        searchKey="search"
        searchPlaceholder={p.searchPosts}
        initialSorting={[{ id: "at", desc: true }]}
        pageSize={15}
        exportName={`posts-${slug}`}
        emptyMessage={p.noPosts}
        onRowClick={(row) => router.push(`/clients/${slug}/publish/posts/${row.original.id}`)}
        rowClassName={() => "cursor-pointer"}
        getRowId={(r) => r.id}
        toolbar={
          <Select value={platform} onValueChange={(v) => setPlatform(v as Platform | "ALL")}>
            <SelectTrigger size="sm" className="w-[150px]" aria-label={p.allPlatforms}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">{p.allPlatforms}</SelectItem>
              {PLATFORMS.map((pl) => (
                <SelectItem key={pl} value={pl}>
                  <PlatformIcon platform={pl} className="size-3.5" /> {platformLabel(pl)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        }
      />
    </div>
  );
}

export type { ColMeta };
