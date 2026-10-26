"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { DataTable } from "@/components/dashboard/data-table";
import { dayLabel } from "@/features/publishing/time";
import { DESIGN_STATUSES } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { DesignStatus, DesignTask, Person } from "@/features/content/types";
import { cn } from "@/lib/utils";
import { DesignStatusBadge } from "../status";
import { OptionSelect } from "../fields";
import { useSetParam } from "../use-set-param";

type Props = {
  slug: string;
  tasks: DesignTask[];
  people: Person[];
  status: DesignStatus | "";
  designerId: string;
  today: string;
};

export function DesignTaskList({ slug, tasks, people, status, designerId, today }: Props) {
  const router = useRouter();
  const names = React.useMemo(() => new Map(people.map((p) => [p.id, p.name])), [people]);

  const setParam = useSetParam();

  const columns = React.useMemo<ColumnDef<DesignTask>[]>(
    () => [
      {
        id: "search",
        accessorFn: (r) => `${r.title} ${r.brief}`,
        header: c.colTask,
        cell: ({ row }) => (
          <span className="min-w-0">
            <span className="block max-w-[44ch] truncate text-[13px] font-medium">{row.original.title}</span>
            <span className="block max-w-[44ch] truncate text-xs text-muted-foreground">{row.original.brief || c.none}</span>
          </span>
        ),
      },
      {
        id: "size",
        accessorFn: (r) => `${r.width}x${r.height}`,
        header: c.colSize,
        size: 110,
        enableSorting: false,
        cell: ({ row }) => (
          <span className="tabular text-xs text-muted-foreground">
            {row.original.width}×{row.original.height}
          </span>
        ),
      },
      {
        id: "designer",
        accessorFn: (r) => names.get(r.designerId ?? "") ?? "",
        header: c.colDesigner,
        size: 150,
        cell: ({ getValue }) => <span className="text-xs">{(getValue() as string) || c.unassigned}</span>,
      },
      {
        id: "due",
        accessorFn: (r) => r.dueDate ?? "",
        header: c.colDue,
        size: 100,
        cell: ({ row }) => {
          const d = row.original.dueDate;
          if (!d) return <span className="text-xs text-muted-foreground">{c.none}</span>;
          const late = d < today && row.original.status !== "DESIGN_APPROVED";
          return <span className={cn("tabular text-xs", late && "font-medium text-destructive")}>{dayLabel(d)}</span>;
        },
      },
      {
        id: "version",
        accessorKey: "currentVersion",
        header: c.colVersion,
        size: 80,
        cell: ({ row }) => <span className="tabular text-xs">{row.original.currentVersion ? c.version(row.original.currentVersion) : c.none}</span>,
      },
      { id: "status", accessorKey: "status", header: c.colStatus, size: 150, cell: ({ row }) => <DesignStatusBadge status={row.original.status} /> },
    ],
    [names, today],
  );

  return (
    <DataTable
      columns={columns}
      data={tasks}
      searchKey="search"
      searchPlaceholder={c.search}
      initialSorting={[{ id: "due", desc: true }]}
      pageSize={15}
      emptyMessage={c.noTasks}
      onRowClick={(row) => router.push(`/clients/${slug}/content/design/${row.original.id}`)}
      rowClassName={() => "cursor-pointer"}
      getRowId={(r) => r.id}
      toolbar={
        <>
          <OptionSelect
            label={c.allStatuses}
            value={status}
            onChange={(v) => setParam("status", v)}
            allLabel={c.allStatuses}
            options={DESIGN_STATUSES.map((s) => ({ value: s, label: c.designStatus[s] }))}
          />
          <OptionSelect
            label={c.allDesigners}
            value={designerId}
            onChange={(v) => setParam("designer", v)}
            allLabel={c.allDesigners}
            options={people.map((p) => ({ value: p.id, label: p.name }))}
          />
        </>
      }
    />
  );
}
