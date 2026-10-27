"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { CalendarDays, Plus, Table2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Toolbar, ToolbarActions } from "@/components/dashboard/layout";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DataTable } from "@/components/dashboard/data-table";
import { EmptyState } from "@/components/dashboard/empty-state";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { dayLabel, monthGrid, weekdayLabel } from "@/features/publishing/time";
import { currentDue, isOverdue } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { Item, Person } from "@/features/content/types";
import { cn } from "@/lib/utils";
import { ItemStatusBadge, ItemStatusDot } from "../status";
import { ItemFormDialog } from "../item-form-dialog";

type Props = {
  slug: string;
  planId: string;
  month: string;
  items: Item[];
  people: Person[];
  pillars: string[];
  timezone: string;
  today: string;
  canWrite: boolean;
};

export function PlanItems({ slug, planId, month, items, people, pillars, timezone, today, canWrite }: Props) {
  const router = useRouter();
  const [view, setView] = React.useState<"calendar" | "table">("calendar");
  const [createDate, setCreateDate] = React.useState<string | null>(null);
  const names = React.useMemo(() => new Map(people.map((p) => [p.id, p.name])), [people]);
  const itemHref = (id: string) => `/clients/${slug}/content/board/${id}`;
  const firstDay = `${month}-01`;

  const byDay = React.useMemo(() => {
    const m = new Map<string, Item[]>();
    for (const it of items) m.set(it.plannedDate, [...(m.get(it.plannedDate) ?? []), it]);
    return m;
  }, [items]);

  const columns = React.useMemo<ColumnDef<Item>[]>(
    () => [
      {
        id: "date",
        accessorKey: "plannedDate",
        header: c.colDate,
        size: 90,
        cell: ({ row }) => <span className="tabular text-xs font-medium">{dayLabel(row.original.plannedDate)}</span>,
      },
      {
        id: "search",
        accessorFn: (r) => `${r.title} ${r.pillar}`,
        header: c.colContent,
        cell: ({ row }) => (
          <span className="min-w-0">
            <span className="block max-w-[40ch] truncate text-[13px] font-medium">{row.original.title}</span>
            <span className="block max-w-[40ch] truncate text-xs text-muted-foreground">{row.original.pillar || c.none}</span>
          </span>
        ),
      },
      {
        id: "platform",
        accessorKey: "platform",
        header: c.colPlatform,
        size: 140,
        cell: ({ row }) => (
          <span className="flex items-center gap-1.5 text-xs">
            <PlatformIcon platform={row.original.platform} className="size-3.5" />
            {c.format[row.original.format]}
          </span>
        ),
      },
      { id: "status", accessorKey: "status", header: c.colStatus, size: 130, cell: ({ row }) => <ItemStatusBadge status={row.original.status} /> },
      {
        id: "team",
        accessorFn: (r) => `${names.get(r.copywriterId ?? "") ?? ""} ${names.get(r.designerId ?? "") ?? ""}`,
        header: c.colAssignees,
        size: 170,
        enableSorting: false,
        cell: ({ row }) => (
          <span className="block text-xs leading-tight text-muted-foreground">
            <span className="block truncate">{names.get(row.original.copywriterId ?? "") ?? c.unassigned}</span>
            <span className="block truncate">{names.get(row.original.designerId ?? "") ?? c.unassigned}</span>
          </span>
        ),
      },
      {
        id: "due",
        accessorFn: (r) => currentDue(r)?.date ?? "",
        header: c.colDue,
        size: 120,
        cell: ({ row }) => {
          const due = currentDue(row.original);
          if (!due) return <span className="text-xs text-muted-foreground">{c.none}</span>;
          return (
            <span className={cn("text-xs tabular", isOverdue(row.original, today) && "font-medium text-destructive")}>
              {c.stageDue[due.stage]} · {dayLabel(due.date)}
            </span>
          );
        },
      },
    ],
    [names, today],
  );

  return (
    <div className="flex min-w-0 flex-col gap-(--gap-inner)">
      <Toolbar>
        <Tabs value={view} onValueChange={(v) => setView(v as "calendar" | "table")}>
          <TabsList>
            <TabsTrigger value="calendar">
              <CalendarDays className="size-3.5" /> {c.calendar}
            </TabsTrigger>
            <TabsTrigger value="table">
              <Table2 className="size-3.5" /> {c.table}
            </TabsTrigger>
          </TabsList>
        </Tabs>
        {canWrite ? (
          <ToolbarActions>
            <Button variant="outline" onClick={() => setCreateDate(firstDay)}>
              <Plus /> {c.addItem}
            </Button>
          </ToolbarActions>
        ) : null}
      </Toolbar>

      {items.length === 0 ? (
        <EmptyState compact icon={<CalendarDays />} title={c.noItems} description={c.noItemsDesc} />
      ) : view === "table" ? (
        <DataTable
          columns={columns}
          data={items}
          searchKey="search"
          searchPlaceholder={c.searchItems}
          initialSorting={[{ id: "date", desc: false }]}
          pageSize={20}
          emptyMessage={c.noItems}
          onRowClick={(row) => router.push(itemHref(row.original.id))}
          rowClassName={() => "cursor-pointer"}
          getRowId={(r) => r.id}
        />
      ) : (
        <div className="overflow-x-auto rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow) scrollbar-thin">
          <div className="grid min-w-[640px] grid-cols-7 gap-px overflow-hidden rounded-xl border bg-border">
            {Array.from({ length: 7 }, (_, i) => (
              <div key={i} className="label-mono bg-muted/50 px-2 py-1.5 text-muted-foreground">
                {weekdayLabel(i)}
              </div>
            ))}
            {monthGrid(firstDay)
              .flat()
              .map((day) => {
                const inMonth = day.startsWith(month);
                const list = byDay.get(day) ?? [];
                return (
                  <div key={day} className={cn("group/day min-h-24 bg-card p-1.5", !inMonth && "bg-muted/30 text-muted-foreground")}>
                    <div className="flex items-center justify-between">
                      <span className={cn("tabular text-xs", day === today && "rounded-full bg-primary px-1.5 text-primary-foreground")}>
                        {Number(day.slice(8))}
                      </span>
                      {canWrite && inMonth ? (
                        <button
                          type="button"
                          onClick={() => setCreateDate(day)}
                          className="rounded p-0.5 text-muted-foreground opacity-0 transition-opacity group-hover/day:opacity-100 hover:bg-muted focus-visible:opacity-100"
                          aria-label={`${c.addItem} ${dayLabel(day)}`}
                        >
                          <Plus className="size-3" />
                        </button>
                      ) : null}
                    </div>
                    <ul className="mt-1 space-y-1">
                      {list.map((it) => (
                        <li key={it.id}>
                          <Link
                            href={itemHref(it.id)}
                            className="flex items-center gap-1 rounded-md border bg-background px-1.5 py-1 text-[11px] leading-tight hover:bg-muted"
                            title={`${it.title} · ${c.itemStatus[it.status]}`}
                          >
                            <ItemStatusDot status={it.status} />
                            <PlatformIcon platform={it.platform} className="size-3" />
                            <span className="min-w-0 flex-1 truncate">{it.title}</span>
                          </Link>
                        </li>
                      ))}
                    </ul>
                  </div>
                );
              })}
          </div>
        </div>
      )}

      {createDate ? (
        <ItemFormDialog
          key={createDate}
          open
          onOpenChange={(o) => !o && setCreateDate(null)}
          slug={slug}
          timezone={timezone}
          people={people}
          pillars={pillars}
          mode={{ kind: "create", planId, date: createDate }}
        />
      ) : null}
    </div>
  );
}
