"use client";

import * as React from "react";
import { Globe, Loader2, Megaphone, Plus, Trash2 } from "lucide-react";
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
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import {
  addAdAccountAction,
  addSeoPropertyAction,
  addSocialAccountAction,
  removeAdAccountAction,
  removeSeoPropertyAction,
  removeSocialAccountAction,
} from "@/features/clients/actions";
import { SOCIAL_PLATFORMS } from "@/features/clients/constants";
import type { SettingsData } from "@/features/clients/queries";
import type { ActionResult } from "@/lib/action-result";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

// ── shared bits ──────────────────────────────────────────────

function useRun() {
  const [pending, startTransition] = React.useTransition();
  const run = (fn: () => Promise<ActionResult<void>>, successMsg: string, after?: () => void) => {
    startTransition(async () => {
      const res = await fn();
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(successMsg);
      after?.();
    });
  };
  return { pending, run };
}

function ListShell({ title, icon, count, addButton, children }: { title: string; icon: React.ReactNode; count: number; addButton: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <h4 className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground uppercase tracking-wide">
          {icon} {title} <span className="font-normal">({count})</span>
        </h4>
        {addButton}
      </div>
      {children}
    </div>
  );
}

function EmptyRow() {
  return <p className="rounded-md border border-dashed px-3 py-2.5 text-xs text-muted-foreground">{t.common.noData}</p>;
}

function RemoveButton({ onConfirm, pending, label }: { onConfirm: () => void; pending: boolean; label: string }) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="icon" className="size-6 text-muted-foreground hover:text-destructive" aria-label={label}>
          <Trash2 className="size-3.5" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t.common.confirmDelete}</AlertDialogTitle>
          <AlertDialogDescription>{tc.settings.removeConfirm}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" disabled={pending} onClick={onConfirm}>
            {t.common.delete}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function AddDialog({
  title,
  description,
  trigger,
  open,
  onOpenChange,
  onSubmit,
  pending,
  children,
}: {
  title: string;
  description?: string;
  trigger: React.ReactNode;
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onSubmit: () => void;
  pending: boolean;
  children: React.ReactNode;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            onSubmit();
          }}
          className="space-y-4"
        >
          {children}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
              {t.common.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {t.common.add}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function AddTrigger({ label, disabled }: { label: string; disabled?: boolean }) {
  return (
    <Button variant="outline" size="xs" disabled={disabled}>
      <Plus className="size-3.5" /> {label}
    </Button>
  );
}

// ── Akun sosial ──────────────────────────────────────────────

export function SocialAccountsList({ clientId, accounts, canManage }: { clientId: string; accounts: SettingsData["socialAccounts"]; canManage: boolean }) {
  const { pending, run } = useRun();
  const [open, setOpen] = React.useState(false);
  const [platform, setPlatform] = React.useState<string>("INSTAGRAM");
  const [username, setUsername] = React.useState("");
  const [displayName, setDisplayName] = React.useState("");

  const submit = () =>
    run(
      () => addSocialAccountAction(clientId, { platform, username, displayName }),
      tc.settings.resourceAdded,
      () => {
        setOpen(false);
        setUsername("");
        setDisplayName("");
      },
    );

  return (
    <ListShell
      title={t.settings.socialAccounts}
      icon={<PlatformIcon platform="INSTAGRAM" mono className="size-3.5" />}
      count={accounts.length}
      addButton={
        canManage ? (
          <AddDialog title={t.social.addAccount} trigger={<AddTrigger label={tc.settings.addSocial} />} open={open} onOpenChange={setOpen} onSubmit={submit} pending={pending}>
            <div className="space-y-2">
              <Label htmlFor="sa-platform">{t.social.platform}</Label>
              <Select value={platform} onValueChange={setPlatform}>
                <SelectTrigger id="sa-platform" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SOCIAL_PLATFORMS.map((p) => (
                    <SelectItem key={p} value={p}>
                      <PlatformIcon platform={p} className="size-4" /> {platformLabel(p)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="sa-username">{tc.settings.username}</Label>
              <Input id="sa-username" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="kopinusantara.id" required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="sa-display">
                {tc.settings.displayName} <span className="text-muted-foreground">({t.common.optional})</span>
              </Label>
              <Input id="sa-display" value={displayName} onChange={(e) => setDisplayName(e.target.value)} placeholder="Kopi Nusantara" />
            </div>
          </AddDialog>
        ) : null
      }
    >
      {accounts.length === 0 ? (
        <EmptyRow />
      ) : (
        <ul className="divide-y rounded-md border">
          {accounts.map((a) => (
            <li key={a.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
              <span className="flex min-w-0 items-center gap-2">
                <PlatformIcon platform={a.platform} className="size-4" />
                <span className="truncate">@{a.username}</span>
                <span className="truncate text-xs text-muted-foreground">{a.displayName}</span>
              </span>
              {canManage ? <RemoveButton pending={pending} label={t.common.delete} onConfirm={() => run(() => removeSocialAccountAction(clientId, a.id), tc.settings.resourceRemoved)} /> : null}
            </li>
          ))}
        </ul>
      )}
    </ListShell>
  );
}

// ── Properti SEO ─────────────────────────────────────────────

export function SeoPropertiesList({ clientId, properties, canManage }: { clientId: string; properties: SettingsData["seoProperties"]; canManage: boolean }) {
  const { pending, run } = useRun();
  const [open, setOpen] = React.useState(false);
  const [siteUrl, setSiteUrl] = React.useState("");
  const [ga4PropertyId, setGa4PropertyId] = React.useState("");

  const submit = () =>
    run(
      () => addSeoPropertyAction(clientId, { siteUrl, ga4PropertyId }),
      tc.settings.resourceAdded,
      () => {
        setOpen(false);
        setSiteUrl("");
        setGa4PropertyId("");
      },
    );

  return (
    <ListShell
      title={t.settings.seoProperties}
      icon={<Globe className="size-3.5" />}
      count={properties.length}
      addButton={
        canManage ? (
          <AddDialog title={t.seo.addProperty} trigger={<AddTrigger label={tc.settings.addSeo} />} open={open} onOpenChange={setOpen} onSubmit={submit} pending={pending}>
            <div className="space-y-2">
              <Label htmlFor="seo-url">{t.seo.siteUrl}</Label>
              <Input id="seo-url" value={siteUrl} onChange={(e) => setSiteUrl(e.target.value)} placeholder="https://klien.co.id" required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="seo-ga4">
                {tc.settings.ga4PropertyId} <span className="text-muted-foreground">({t.common.optional})</span>
              </Label>
              <Input id="seo-ga4" value={ga4PropertyId} onChange={(e) => setGa4PropertyId(e.target.value)} placeholder={tc.settings.ga4Placeholder} />
            </div>
          </AddDialog>
        ) : null
      }
    >
      {properties.length === 0 ? (
        <EmptyRow />
      ) : (
        <ul className="divide-y rounded-md border">
          {properties.map((p) => (
            <li key={p.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
              <span className="flex min-w-0 items-center gap-2">
                <Globe className="size-4 shrink-0 text-muted-foreground" />
                <span className="truncate">{p.siteUrl}</span>
                {p.ga4PropertyId ? (
                  <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">
                    GA4 · {p.ga4PropertyId}
                  </Badge>
                ) : null}
              </span>
              {canManage ? <RemoveButton pending={pending} label={t.common.delete} onConfirm={() => run(() => removeSeoPropertyAction(clientId, p.id), tc.settings.resourceRemoved)} /> : null}
            </li>
          ))}
        </ul>
      )}
    </ListShell>
  );
}

// ── Akun iklan ───────────────────────────────────────────────

export function AdAccountsList({ clientId, accounts, canManage }: { clientId: string; accounts: SettingsData["adAccounts"]; canManage: boolean }) {
  const { pending, run } = useRun();
  const [open, setOpen] = React.useState(false);
  const [externalId, setExternalId] = React.useState("");
  const [name, setName] = React.useState("");

  const submit = () =>
    run(
      () => addAdAccountAction(clientId, { externalId, name }),
      tc.settings.resourceAdded,
      () => {
        setOpen(false);
        setExternalId("");
        setName("");
      },
    );

  return (
    <ListShell
      title={t.settings.adAccounts}
      icon={<Megaphone className="size-3.5" />}
      count={accounts.length}
      addButton={
        canManage ? (
          <AddDialog title={tc.settings.addAds} trigger={<AddTrigger label={tc.settings.addAds} />} open={open} onOpenChange={setOpen} onSubmit={submit} pending={pending}>
            <div className="space-y-2">
              <Label htmlFor="ads-id">{tc.settings.adAccountExternalId}</Label>
              <Input id="ads-id" value={externalId} onChange={(e) => setExternalId(e.target.value)} placeholder={tc.settings.adAccountPlaceholder} required className="font-mono text-sm" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ads-name">{tc.settings.adAccountName}</Label>
              <Input id="ads-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Kopi Nusantara Ads" required />
            </div>
          </AddDialog>
        ) : null
      }
    >
      {accounts.length === 0 ? (
        <EmptyRow />
      ) : (
        <ul className="divide-y rounded-md border">
          {accounts.map((a) => (
            <li key={a.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
              <span className="flex min-w-0 items-center gap-2">
                <Megaphone className="size-4 shrink-0 text-muted-foreground" />
                <span className="truncate">{a.name}</span>
                <span className="truncate font-mono text-xs text-muted-foreground">{a.externalId}</span>
              </span>
              {canManage ? <RemoveButton pending={pending} label={t.common.delete} onConfirm={() => run(() => removeAdAccountAction(clientId, a.id), tc.settings.resourceRemoved)} /> : null}
            </li>
          ))}
        </ul>
      )}
    </ListShell>
  );
}
