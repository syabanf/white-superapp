"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Plus } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { DataTable } from "@/components/dashboard/data-table";
import { monthLabel } from "@/features/publishing/time";
import { createPlan } from "@/features/content/actions";
import { c } from "@/features/content/strings";
import type { Plan } from "@/features/content/types";
import { formatRelative } from "@/lib/format";
import { PlanStatusBadge } from "../status";
import { Field } from "../fields";

export function PlansTable({ plans, slug }: { plans: Plan[]; slug: string }) {
  const router = useRouter();
  const columns = React.useMemo<ColumnDef<Plan>[]>(
    () => [
      {
        id: "month",
        accessorKey: "month",
        header: c.colMonth,
        size: 150,
        cell: ({ row }) => <span className="text-[13px] font-medium">{monthLabel(row.original.month)}</span>,
      },
      { id: "title", accessorKey: "title", header: c.colTitle, cell: ({ getValue }) => <span className="text-[13px]">{getValue() as string}</span> },
      { id: "status", accessorKey: "status", header: c.colStatus, size: 150, cell: ({ row }) => <PlanStatusBadge status={row.original.status} /> },
      {
        id: "items",
        accessorKey: "itemCount",
        header: c.colItems,
        size: 90,
        cell: ({ row }) => <span className="tabular text-[13px]">{row.original.itemCount}</span>,
      },
      {
        id: "published",
        accessorKey: "publishedCount",
        header: c.colPublished,
        size: 90,
        cell: ({ row }) => <span className="tabular text-[13px]">{row.original.publishedCount}</span>,
      },
      {
        id: "updated",
        accessorKey: "updatedAt",
        header: c.colUpdated,
        size: 130,
        cell: ({ row }) => <span className="text-xs text-muted-foreground">{formatRelative(row.original.updatedAt)}</span>,
      },
    ],
    [],
  );
  return (
    <DataTable
      columns={columns}
      data={plans}
      initialSorting={[{ id: "month", desc: true }]}
      pageSize={12}
      emptyMessage={c.noPlans}
      onRowClick={(row) => router.push(`/clients/${slug}/content/plans/${row.original.id}`)}
      rowClassName={() => "cursor-pointer"}
      getRowId={(r) => r.id}
    />
  );
}

export function CreatePlanButton({ slug, defaultMonth }: { slug: string; defaultMonth: string }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [month, setMonth] = React.useState(defaultMonth);
  const [title, setTitle] = React.useState("");
  const [pending, start] = React.useTransition();

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    start(async () => {
      const res = await createPlan({ slug, month, title });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(c.planCreated);
      setOpen(false);
      router.push(`/clients/${slug}/content/plans/${res.data.id}`);
    });
  };

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Plus /> {c.createPlan}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{c.newPlanTitle}</DialogTitle>
            <DialogDescription>{c.newPlanDesc}</DialogDescription>
          </DialogHeader>
          <form id="plan-form" onSubmit={submit} className="space-y-4">
            <Field id="plan-month" label={c.month}>
              <Input id="plan-month" type="month" value={month} onChange={(e) => setMonth(e.target.value)} required />
            </Field>
            <Field id="plan-title" label={c.titleLabel} hint={c.titleHint}>
              <Input id="plan-title" value={title} onChange={(e) => setTitle(e.target.value)} />
            </Field>
          </form>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              {c.cancel}
            </Button>
            <Button type="submit" form="plan-form" disabled={pending || !month}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {c.createPlan}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
