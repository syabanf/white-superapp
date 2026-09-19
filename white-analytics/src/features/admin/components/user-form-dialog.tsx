"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { createUserAction, updateUserAction, type UserFormInput } from "@/features/admin/actions";
import type { AdminUserRow } from "@/features/admin/queries";
import { t } from "@/i18n/id";
import { ta } from "@/features/admin/strings";

export type ClientOption = { id: string; name: string; slug: string };

type MembershipState = Record<string, "MANAGER" | "VIEWER">;

function FieldError({ errors, name }: { errors: Record<string, string[]> | undefined; name: string }) {
  const msg = errors?.[name]?.[0];
  if (!msg) return null;
  return <p className="text-xs text-destructive">{msg}</p>;
}

/**
 * Create / edit user dialog (ADMIN). Form state lives in the body component,
 * which unmounts when the dialog closes — it always opens fresh.
 */
export function UserFormDialog({
  mode,
  user,
  clients,
  trigger,
  open: openProp,
  onOpenChange,
}: {
  mode: "create" | "edit";
  user?: AdminUserRow;
  clients: ClientOption[];
  trigger?: React.ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}) {
  const [openState, setOpenState] = React.useState(false);
  const open = openProp ?? openState;
  const setOpen = (next: boolean) => {
    onOpenChange?.(next);
    if (openProp === undefined) setOpenState(next);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {trigger ? <DialogTrigger asChild>{trigger}</DialogTrigger> : null}
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{mode === "create" ? t.admin.newUser : t.admin.editUser}</DialogTitle>
          <DialogDescription>{t.admin.subtitle}</DialogDescription>
        </DialogHeader>
        <UserFormBody key={user?.id ?? "new"} mode={mode} user={user} clients={clients} onClose={() => setOpen(false)} />
      </DialogContent>
    </Dialog>
  );
}

function UserFormBody({
  mode,
  user,
  clients,
  onClose,
}: {
  mode: "create" | "edit";
  user?: AdminUserRow;
  clients: ClientOption[];
  onClose: () => void;
}) {
  const [name, setName] = React.useState(user?.name ?? "");
  const [email, setEmail] = React.useState(user?.email ?? "");
  const [password, setPassword] = React.useState("");
  const [role, setRole] = React.useState<"ADMIN" | "MEMBER">(user?.role ?? "MEMBER");
  const [isActive, setIsActive] = React.useState(user?.isActive ?? true);
  const [memberships, setMemberships] = React.useState<MembershipState>(() => {
    const out: MembershipState = {};
    for (const c of user?.clients ?? []) out[c.id] = c.role;
    return out;
  });
  const [errors, setErrors] = React.useState<Record<string, string[]> | undefined>(undefined);
  const [pending, startTransition] = React.useTransition();

  const toggleClient = (clientId: string, checked: boolean) => {
    setMemberships((prev) => {
      const next = { ...prev };
      if (checked) next[clientId] = next[clientId] ?? "VIEWER";
      else delete next[clientId];
      return next;
    });
  };

  const submit = () => {
    const payload: UserFormInput = {
      name,
      email,
      password,
      role,
      isActive,
      memberships: Object.entries(memberships).map(([clientId, r]) => ({ clientId, role: r })),
    };
    startTransition(async () => {
      const res = mode === "create" ? await createUserAction(payload) : await updateUserAction(user!.id, payload);
      if (!res.ok) {
        setErrors(res.fieldErrors);
        toast.error(res.error);
        return;
      }
      toast.success(mode === "create" ? t.admin.created : t.admin.updated);
      onClose();
    });
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        submit();
      }}
      className="space-y-4"
    >
      <div className="grid gap-5 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="user-name">{t.admin.name}</Label>
          <Input id="user-name" value={name} onChange={(e) => setName(e.target.value)} required placeholder="Tim WHITE" />
          <FieldError errors={errors} name="name" />
        </div>
        <div className="space-y-2">
          <Label htmlFor="user-email">{t.admin.email}</Label>
          <Input id="user-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required placeholder="nama@white.id" />
          <FieldError errors={errors} name="email" />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="user-password">{t.admin.password}</Label>
        <Input
          id="user-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required={mode === "create"}
          minLength={mode === "create" ? 8 : undefined}
          placeholder="••••••••"
          autoComplete="new-password"
        />
        <p className="text-xs text-muted-foreground">{mode === "edit" ? t.admin.passwordHint : ta.passwordMin}</p>
        <FieldError errors={errors} name="password" />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="user-role">{t.admin.role}</Label>
          <Select value={role} onValueChange={(v) => setRole(v as "ADMIN" | "MEMBER")}>
            <SelectTrigger id="user-role" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ADMIN">{t.admin.roleAdmin}</SelectItem>
              <SelectItem value="MEMBER">{t.admin.roleMember}</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-xs text-muted-foreground">{role === "ADMIN" ? t.admin.roleAdminDesc : t.admin.roleMemberDesc}</p>
        </div>
        <div className="space-y-2">
          <Label htmlFor="user-active">{t.admin.activeStatus}</Label>
          <div className="flex h-9 items-center">
            <Switch id="user-active" checked={isActive} onCheckedChange={setIsActive} />
          </div>
        </div>
      </div>

      <div className="space-y-2">
        <Label>{t.admin.clientsAccess}</Label>
        {role === "ADMIN" ? (
          <p className="rounded-md border border-dashed px-3 py-2 text-xs text-muted-foreground">{ta.adminAccessAll}</p>
        ) : (
          <>
            <p className="text-xs text-muted-foreground">{ta.accessHint}</p>
            <ScrollArea className="max-h-48 rounded-md border">
              <ul className="divide-y">
                {clients.map((c) => {
                  const checked = memberships[c.id] !== undefined;
                  return (
                    <li key={c.id} className="flex items-center justify-between gap-2 px-3 py-2">
                      <label className="flex min-w-0 flex-1 cursor-pointer items-center gap-2 text-sm">
                        <Checkbox checked={checked} onCheckedChange={(v) => toggleClient(c.id, v === true)} aria-label={c.name} />
                        <span className="truncate">{c.name}</span>
                      </label>
                      {checked ? (
                        <label className="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground">
                          {ta.makeManager}
                          <Switch
                            checked={memberships[c.id] === "MANAGER"}
                            onCheckedChange={(v) => setMemberships((prev) => ({ ...prev, [c.id]: v ? "MANAGER" : "VIEWER" }))}
                            aria-label={`${ta.makeManager} ${c.name}`}
                          />
                        </label>
                      ) : (
                        <Badge variant="outline" className="px-1.5 py-0 text-[10px] text-muted-foreground/60">
                          {t.clients.roleViewer}
                        </Badge>
                      )}
                    </li>
                  );
                })}
              </ul>
            </ScrollArea>
          </>
        )}
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" onClick={onClose} disabled={pending}>
          {t.common.cancel}
        </Button>
        <Button type="submit" disabled={pending || name.trim().length === 0 || email.trim().length === 0}>
          {pending ? <Loader2 className="size-4 animate-spin" /> : null}
          {mode === "create" ? t.common.create : t.common.save}
        </Button>
      </DialogFooter>
    </form>
  );
}
