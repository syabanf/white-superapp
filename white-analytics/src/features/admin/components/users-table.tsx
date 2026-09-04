"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Pencil, ShieldCheck, Trash2, User as UserIcon } from "lucide-react";
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
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { deleteUserAction, setUserActiveAction } from "@/features/admin/actions";
import type { AdminUserRow } from "@/features/admin/queries";
import { formatDate, initials } from "@/lib/format";
import { t } from "@/i18n/id";
import { ta } from "@/features/admin/strings";
import { UserFormDialog, type ClientOption } from "./user-form-dialog";

export function UsersTable({ users, clients, currentUserId }: { users: AdminUserRow[]; clients: ClientOption[]; currentUserId: string }) {
  const [editTarget, setEditTarget] = React.useState<AdminUserRow | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<AdminUserRow | null>(null);
  const [pending, startTransition] = React.useTransition();

  const toggleActive = (user: AdminUserRow, next: boolean) => {
    startTransition(async () => {
      const res = await setUserActiveAction(user.id, next);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(ta.statusUpdated);
    });
  };

  const confirmDelete = () => {
    if (!deleteTarget) return;
    startTransition(async () => {
      const res = await deleteUserAction(deleteTarget.id);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(t.admin.deleted);
      setDeleteTarget(null);
    });
  };

  const columns: ColumnDef<AdminUserRow, unknown>[] = [
    {
      id: "name",
      // combined so the search box matches name AND email
      accessorFn: (row) => `${row.name} ${row.email}`,
      header: t.admin.name,
      cell: ({ row }) => (
        <span className="flex items-center gap-2.5">
          <Avatar className="size-8">
            <AvatarFallback className="text-xs font-semibold">{initials(row.original.name)}</AvatarFallback>
          </Avatar>
          <span className="min-w-0">
            <span className="block truncate font-medium">
              {row.original.name}
              {row.original.id === currentUserId ? (
                <Badge variant="secondary" className="ml-2 px-1.5 py-0 text-[10px]">
                  {ta.you}
                </Badge>
              ) : null}
            </span>
            <span className="block truncate text-xs text-muted-foreground">{row.original.email}</span>
          </span>
        </span>
      ),
    },
    {
      accessorKey: "role",
      header: t.admin.role,
      cell: ({ getValue }) => {
        const role = getValue() as AdminUserRow["role"];
        return role === "ADMIN" ? (
          <Badge className="gap-1 text-[11px]">
            <ShieldCheck className="size-3" /> {t.admin.roleAdmin}
          </Badge>
        ) : (
          <Badge variant="secondary" className="gap-1 text-[11px]">
            <UserIcon className="size-3" /> {t.admin.roleMember}
          </Badge>
        );
      },
    },
    {
      accessorKey: "isActive",
      header: t.admin.activeStatus,
      cell: ({ row }) => (
        <Switch
          checked={row.original.isActive}
          onCheckedChange={(v) => toggleActive(row.original, v)}
          disabled={pending || row.original.id === currentUserId}
          aria-label={t.admin.activeStatus}
        />
      ),
    },
    {
      id: "clients",
      accessorFn: (row) => row.clients.length,
      header: t.admin.clientsAccess,
      meta: { align: "right" } satisfies ColMeta,
      cell: ({ row }) => {
        const u = row.original;
        if (u.role === "ADMIN") return <span className="text-xs text-muted-foreground">{t.common.all}</span>;
        if (u.clients.length === 0) return <span className="text-xs text-muted-foreground">{ta.noClients}</span>;
        return (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="cursor-default underline decoration-dotted underline-offset-4 tabular">{u.clients.length}</span>
            </TooltipTrigger>
            <TooltipContent className="max-w-64">
              <p className="mb-1 text-xs font-medium">{ta.clientsTooltip}</p>
              <ul className="space-y-0.5 text-xs">
                {u.clients.map((c) => (
                  <li key={c.id} className="flex items-center justify-between gap-3">
                    <span className="truncate">{c.name}</span>
                    <span className="shrink-0 opacity-70">{c.role === "MANAGER" ? t.clients.roleManager : t.clients.roleViewer}</span>
                  </li>
                ))}
              </ul>
            </TooltipContent>
          </Tooltip>
        );
      },
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
        const u = row.original;
        return (
          <span className="flex items-center justify-end gap-1">
            <Button variant="ghost" size="icon" className="size-7" aria-label={t.admin.editUser} onClick={() => setEditTarget(u)}>
              <Pencil className="size-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 text-destructive hover:text-destructive disabled:opacity-30"
              aria-label={t.common.delete}
              disabled={u.id === currentUserId}
              onClick={() => setDeleteTarget(u)}
            >
              <Trash2 className="size-4" />
            </Button>
          </span>
        );
      },
    },
  ];

  return (
    <>
      <DataTable
        columns={columns}
        data={users}
        searchKey="name"
        searchPlaceholder={t.common.search}
        exportName="users"
        emptyMessage={ta.emptyUsers}
        initialSorting={[{ id: "name", desc: false }]}
      />

      {editTarget ? (
        <UserFormDialog
          mode="edit"
          user={editTarget}
          clients={clients}
          open
          onOpenChange={(open) => {
            if (!open) setEditTarget(null);
          }}
        />
      ) : null}

      <AlertDialog open={deleteTarget !== null} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{ta.deleteTitle}</AlertDialogTitle>
            <AlertDialogDescription>{deleteTarget ? ta.deleteDesc(deleteTarget.name) : null}</AlertDialogDescription>
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
