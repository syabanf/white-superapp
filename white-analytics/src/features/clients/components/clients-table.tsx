"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { ColumnDef } from "@tanstack/react-table";
import { ArrowUpRight, Globe, Loader2, Pencil, Trash2, Users } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { deleteClientAction } from "@/features/clients/actions";
import type { ClientRow } from "@/features/clients/queries";
import { formatDate, initials, truncate } from "@/lib/format";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";
import { ClientFormDialog, type EditableClient } from "./client-form-dialog";

export function ClientsTable({ clients, isAdmin }: { clients: ClientRow[]; isAdmin: boolean }) {
  const router = useRouter();
  const [editTarget, setEditTarget] = React.useState<EditableClient | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<ClientRow | null>(null);
  const [pending, startTransition] = React.useTransition();

  const confirmDelete = () => {
    if (!deleteTarget) return;
    startTransition(async () => {
      const res = await deleteClientAction(deleteTarget.id);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(t.clients.deleted);
      setDeleteTarget(null);
      router.refresh();
    });
  };

  const columns: ColumnDef<ClientRow, unknown>[] = [
    {
      id: "name",
      accessorKey: "name",
      header: t.common.name,
      cell: ({ row }) => (
        <span className="flex items-center gap-2.5">
          <Avatar className="size-8 rounded-md">
            {row.original.logoUrl ? <AvatarImage src={row.original.logoUrl} alt={row.original.name} /> : null}
            <AvatarFallback className="rounded-md bg-primary/10 text-xs font-semibold text-primary">{initials(row.original.name)}</AvatarFallback>
          </Avatar>
          <span className="min-w-0">
            <span className="block truncate font-medium">{row.original.name}</span>
            <span className="block truncate font-mono text-[11px] text-muted-foreground">/{row.original.slug}</span>
          </span>
        </span>
      ),
    },
    {
      accessorKey: "industry",
      header: t.clients.industry,
      cell: ({ getValue }) => <span className="text-muted-foreground">{(getValue() as string | null) ?? "–"}</span>,
    },
    {
      accessorKey: "websiteUrl",
      header: t.clients.website,
      cell: ({ getValue }) => {
        const url = getValue() as string | null;
        if (!url) return <span className="text-muted-foreground">–</span>;
        return (
          <a
            href={url}
            target="_blank"
            rel="noreferrer noopener"
            className="inline-flex items-center gap-1 text-muted-foreground hover:text-foreground hover:underline"
            onClick={(e) => e.stopPropagation()}
          >
            <Globe className="size-3.5" /> {truncate(url.replace(/^https?:\/\//, ""), 30)}
          </a>
        );
      },
    },
    {
      accessorKey: "membersCount",
      header: t.clients.members,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ getValue }) => (
        <span className="inline-flex items-center gap-1 tabular">
          <Users className="size-3.5 text-muted-foreground" /> {getValue() as number}
        </span>
      ),
    },
    {
      accessorKey: "createdAt",
      header: t.clients.createdAt,
      cell: ({ getValue }) => <span className="text-muted-foreground">{formatDate(getValue() as string)}</span>,
    },
    {
      id: "actions",
      header: t.common.actions,
      enableSorting: false,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => {
        const c = row.original;
        const canEdit = c.myRole === "ADMIN" || c.myRole === "MANAGER";
        return (
          <span className="flex items-center justify-end gap-1" onClick={(e) => e.stopPropagation()}>
            <Button asChild variant="ghost" size="icon" className="size-7" aria-label={t.clients.open}>
              <Link href={`/clients/${c.slug}`}>
                <ArrowUpRight className="size-4" />
              </Link>
            </Button>
            {canEdit ? (
              <Button
                variant="ghost"
                size="icon"
                className="size-7"
                aria-label={t.clients.edit}
                onClick={() =>
                  setEditTarget({
                    id: c.id,
                    name: c.name,
                    slug: c.slug,
                    description: c.description,
                    industry: c.industry ?? "",
                    websiteUrl: c.websiteUrl ?? "",
                    currency: c.currency,
                    timezone: c.timezone,
                  })
                }
              >
                <Pencil className="size-4" />
              </Button>
            ) : null}
            {isAdmin ? (
              <Button
                variant="ghost"
                size="icon"
                className="size-7 text-destructive hover:text-destructive"
                aria-label={t.common.delete}
                onClick={() => setDeleteTarget(c)}
              >
                <Trash2 className="size-4" />
              </Button>
            ) : null}
          </span>
        );
      },
    },
  ];

  return (
    <>
      <DataTable
        columns={columns}
        data={clients}
        searchKey="name"
        searchPlaceholder={tc.portfolio.search}
        exportName="klien"
        emptyMessage={t.portfolio.emptyTitle}
        onRowClick={(row) => router.push(`/clients/${row.original.slug}`)}
        initialSorting={[{ id: "name", desc: false }]}
      />

      {editTarget ? (
        <ClientFormDialog
          mode="edit"
          client={editTarget}
          open
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
        />
      ) : null}

      <AlertDialog open={deleteTarget !== null} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{tc.clients.deleteTitle}</AlertDialogTitle>
            <AlertDialogDescription>
              {t.clients.deleteConfirm} {t.common.confirmDeleteDesc}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={pending}>{t.common.cancel}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={pending}
              onClick={(e) => {
                e.preventDefault();
                confirmDelete();
              }}
            >
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {t.common.delete}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
