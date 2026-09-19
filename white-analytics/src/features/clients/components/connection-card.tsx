"use client";

import * as React from "react";
import { FlaskConical, Globe, Loader2, PlugZap, Unplug } from "lucide-react";
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { disconnectConnectionAction, simulateConnectionAction } from "@/features/clients/actions-resources";
import type { SettingsConnection } from "@/features/clients/queries";
import { formatDate } from "@/lib/format";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

function scopeLabel(scope: string): string {
  if (scope.startsWith("https://")) return scope.split("/").pop() ?? scope;
  return scope;
}

/**
 * One provider card (META or GOOGLE): connection status, connect / simulate /
 * disconnect controls, plus the linked resource lists passed as children.
 */
export function ConnectionCard({
  provider,
  clientId,
  slug,
  connections,
  configured,
  canManage,
  children,
}: {
  provider: "META" | "GOOGLE";
  clientId: string;
  slug: string;
  connections: SettingsConnection[];
  configured: boolean;
  canManage: boolean;
  children?: React.ReactNode;
}) {
  const [pending, startTransition] = React.useTransition();
  const isMeta = provider === "META";
  const title = isMeta ? t.settings.meta : t.settings.google;
  const description = isMeta ? t.settings.metaDesc : t.settings.googleDesc;
  const connectLabel = isMeta ? t.settings.connectMeta : t.settings.connectGoogle;
  const startHref = `/api/connections/${isMeta ? "meta" : "google"}/start?client=${slug}`;

  const simulate = () => {
    startTransition(async () => {
      const res = await simulateConnectionAction(clientId, provider);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(tc.settings.simulated);
    });
  };

  const disconnect = (connectionId: string) => {
    startTransition(async () => {
      const res = await disconnectConnectionAction(clientId, connectionId);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      toast.success(tc.settings.disconnected);
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-sm font-semibold">
          {isMeta ? (
            <PlatformIcon platform="FACEBOOK" className="size-4" />
          ) : (
            <Globe className="size-4 text-muted-foreground" />
          )}
          {title}
        </CardTitle>
        <CardDescription className="text-xs">{description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {connections.length === 0 ? (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-dashed px-3 py-2.5">
            <StatusBadge kind="neutral">{t.common.disconnected}</StatusBadge>
            {canManage ? (
              configured ? (
                <Button asChild size="xs">
                  {/* full navigation — route handler redirects to the provider dialog */}
                  <a href={startHref}>
                    <PlugZap className="size-3.5" /> {connectLabel}
                  </a>
                </Button>
              ) : (
                <Button size="xs" variant="outline" onClick={simulate} disabled={pending}>
                  {pending ? (
                    <Loader2 className="size-3.5 animate-spin" />
                  ) : (
                    <FlaskConical className="size-3.5" />
                  )}
                  {tc.settings.simulate}
                </Button>
              )
            ) : null}
          </div>
        ) : (
          <ul className="space-y-3">
            {connections.map((c) => (
              <li key={c.id} className="space-y-2 rounded-lg border px-3 py-2.5">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex min-w-0 items-center gap-2">
                    <StatusBadge kind="good">{t.common.connected}</StatusBadge>
                    <span className="truncate text-sm font-medium">{c.displayName}</span>
                  </span>
                  {canManage ? (
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="xs"
                          className="text-muted-foreground hover:text-destructive"
                        >
                          <Unplug className="size-3.5" /> {t.settings.disconnect}
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{t.settings.disconnect}</AlertDialogTitle>
                          <AlertDialogDescription>{tc.settings.disconnectConfirm}</AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{t.common.cancel}</AlertDialogCancel>
                          <AlertDialogAction
                            variant="destructive"
                            disabled={pending}
                            onClick={() => disconnect(c.id)}
                          >
                            {t.settings.disconnect}
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  ) : null}
                </div>
                <dl className="grid gap-x-4 gap-y-1 text-xs text-muted-foreground sm:grid-cols-2">
                  <div className="flex items-center gap-1.5">
                    <dt>{tc.settings.accountId}:</dt>
                    <dd className="truncate font-mono">{c.accountId}</dd>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <dt>{t.settings.expiresAt}:</dt>
                    <dd>{c.expiresAt ? formatDate(c.expiresAt) : "–"}</dd>
                  </div>
                </dl>
                {c.scopes.length > 0 ? (
                  <div className="flex flex-wrap items-center gap-1">
                    <span className="text-xs text-muted-foreground">{t.settings.scopes}:</span>
                    {c.scopes.map((s) => (
                      <Badge
                        key={s}
                        variant="outline"
                        className="px-1.5 py-0 font-mono text-xs font-normal text-muted-foreground"
                      >
                        {scopeLabel(s)}
                      </Badge>
                    ))}
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        )}

        {!configured ? <p className="text-xs text-muted-foreground">{t.settings.notConfigured}</p> : null}

        {connections.length > 0 && canManage && configured ? (
          <Button asChild variant="outline" size="xs">
            <a href={startHref}>
              <PlugZap className="size-3.5" /> {t.common.reconnect}
            </a>
          </Button>
        ) : null}

        {children ? (
          <>
            <Separator />
            <div className="space-y-4">{children}</div>
          </>
        ) : null}
      </CardContent>
    </Card>
  );
}
