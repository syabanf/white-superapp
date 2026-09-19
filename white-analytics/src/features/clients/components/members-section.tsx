"use client";

import * as React from "react";
import { Loader2, Trash2, UserPlus } from "lucide-react";
import { toast } from "@/lib/toast";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { addMemberAction, removeMemberAction, updateMemberRoleAction } from "@/features/clients/actions";
import type { SettingsData } from "@/features/clients/queries";
import type { ActionResult } from "@/lib/action-result";
import { formatDate, initials } from "@/lib/format";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

const ROLE_LABEL: Record<string, string> = {
  MANAGER: t.clients.roleManager,
  VIEWER: t.clients.roleViewer,
};

export function MembersSection({
  clientId,
  members,
  candidates,
  canManage,
}: {
  clientId: string;
  members: SettingsData["members"];
  candidates: { id: string; name: string; email: string }[];
  canManage: boolean;
}) {
  const [pending, startTransition] = React.useTransition();
  const [selectedUser, setSelectedUser] = React.useState("");
  const [selectedRole, setSelectedRole] = React.useState("VIEWER");

  const available = candidates.filter((u) => !members.some((m) => m.userId === u.id));

  const run = (fn: () => Promise<ActionResult<void>>, msg: string, after?: () => void) => {
    startTransition(async () => {
      const res = await fn();
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(msg);
      after?.();
    });
  };

  const changeRole = (userId: string, nextRole: string, previousRole: string) => {
    startTransition(async () => {
      const res = await updateMemberRoleAction(clientId, { userId, role: nextRole });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(tc.settings.memberRoleUpdated, {
        action: {
          label: "Urungkan",
          onClick: () =>
            startTransition(async () => {
              const undo = await updateMemberRoleAction(clientId, { userId, role: previousRole });
              if (undo.ok) toast.success("Peran anggota dikembalikan");
              else toast.error(undo.error);
            }),
        },
      });
    });
  };

  return (
    <Card>
      <CardContent className="space-y-4">
        {canManage ? (
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-56 flex-1 space-y-1.5">
              <span className="text-xs font-medium text-muted-foreground">{t.clients.addMember}</span>
              <Select value={selectedUser} onValueChange={setSelectedUser} disabled={available.length === 0}>
                <SelectTrigger className="w-full" aria-label={tc.settings.pickUser}>
                  <SelectValue
                    placeholder={available.length === 0 ? tc.settings.noCandidates : tc.settings.pickUser}
                  />
                </SelectTrigger>
                <SelectContent>
                  {available.map((u) => (
                    <SelectItem key={u.id} value={u.id}>
                      {u.name} <span className="text-muted-foreground">· {u.email}</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <Select value={selectedRole} onValueChange={setSelectedRole}>
              <SelectTrigger className="w-32" aria-label={t.admin.role}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="VIEWER">{t.clients.roleViewer}</SelectItem>
                <SelectItem value="MANAGER">{t.clients.roleManager}</SelectItem>
              </SelectContent>
            </Select>
            <Button
              size="sm"
              className="h-8"
              disabled={pending || selectedUser === ""}
              onClick={() =>
                run(
                  () => addMemberAction(clientId, { userId: selectedUser, role: selectedRole }),
                  tc.settings.memberAdded,
                  () => setSelectedUser(""),
                )
              }
            >
              {pending ? <Loader2 className="size-4 animate-spin" /> : <UserPlus className="size-4" />}
              Tambah anggota
            </Button>
          </div>
        ) : null}

        {members.length === 0 ? (
          <p className="rounded-md border border-dashed px-3 py-3 text-sm text-muted-foreground">
            {tc.settings.membersEmpty}
          </p>
        ) : (
          <>
            <div className="space-y-2 md:hidden">
              {members.map((m) => (
                <div key={m.userId} className="space-y-3 rounded-lg border p-3">
                  <div className="flex items-center gap-2.5">
                    <Avatar className="size-8">
                      <AvatarFallback className="text-xs font-semibold">{initials(m.name)}</AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{m.name}</p>
                      <p className="truncate text-xs text-muted-foreground">{m.email}</p>
                    </div>
                    {!m.isActive ? <Badge variant="outline">{t.common.inactive}</Badge> : null}
                  </div>
                  <div className="flex items-center justify-between gap-3 border-t pt-3">
                    <span className="text-xs text-muted-foreground">{tc.settings.memberSince} {formatDate(m.createdAt)}</span>
                    {canManage ? (
                      <div className="flex items-center gap-1">
                        <Select
                          value={m.role}
                          onValueChange={(role) => changeRole(m.userId, role, m.role)}
                          disabled={pending}
                        >
                          <SelectTrigger size="sm" className="w-28" aria-label={`${t.admin.role} ${m.name}`}><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="VIEWER">{t.clients.roleViewer}</SelectItem>
                            <SelectItem value="MANAGER">{t.clients.roleManager}</SelectItem>
                          </SelectContent>
                        </Select>
                        <Button variant="ghost" size="icon" className="size-8 text-muted-foreground hover:text-destructive" aria-label={`${t.common.delete} ${m.name}`} disabled={pending} onClick={() => run(() => removeMemberAction(clientId, m.userId), tc.settings.memberRemoved)}>
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    ) : <Badge variant="secondary">{ROLE_LABEL[m.role] ?? m.role}</Badge>}
                  </div>
                </div>
              ))}
            </div>
            <div className="hidden overflow-hidden rounded-lg border md:block">
              <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs">{t.common.name}</TableHead>
                  <TableHead className="text-xs">{t.admin.role}</TableHead>
                  <TableHead className="text-xs">{tc.settings.memberSince}</TableHead>
                  {canManage ? (
                    <TableHead className="w-12 text-right text-xs">{t.common.actions}</TableHead>
                  ) : null}
                </TableRow>
              </TableHeader>
              <TableBody>
                {members.map((m) => (
                  <TableRow key={m.userId}>
                    <TableCell>
                      <span className="flex items-center gap-2.5">
                        <Avatar className="size-8">
                          <AvatarFallback className="text-xs font-semibold">
                            {initials(m.name)}
                          </AvatarFallback>
                        </Avatar>
                        <span className="min-w-0">
                          <span className="block truncate text-sm font-medium">
                            {m.name}
                            {!m.isActive ? (
                              <Badge
                                variant="outline"
                                className="ml-2 px-1.5 py-0 text-xs text-muted-foreground"
                              >
                                {t.common.inactive}
                              </Badge>
                            ) : null}
                          </span>
                          <span className="block truncate text-xs text-muted-foreground">{m.email}</span>
                        </span>
                      </span>
                    </TableCell>
                    <TableCell>
                      {canManage ? (
                        <Select
                          value={m.role}
                          onValueChange={(role) => changeRole(m.userId, role, m.role)}
                          disabled={pending}
                        >
                          <SelectTrigger size="sm" className="w-28" aria-label={t.admin.role}>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="VIEWER">{t.clients.roleViewer}</SelectItem>
                            <SelectItem value="MANAGER">{t.clients.roleManager}</SelectItem>
                          </SelectContent>
                        </Select>
                      ) : (
                        <Badge variant="secondary" className="text-xs">
                          {ROLE_LABEL[m.role] ?? m.role}
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDate(m.createdAt)}</TableCell>
                    {canManage ? (
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-7 text-muted-foreground hover:text-destructive"
                          aria-label={t.common.delete}
                          disabled={pending}
                          onClick={() =>
                            run(() => removeMemberAction(clientId, m.userId), tc.settings.memberRemoved)
                          }
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </TableCell>
                    ) : null}
                  </TableRow>
                ))}
              </TableBody>
              </Table>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
