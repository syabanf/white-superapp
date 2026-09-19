"use client";

import * as React from "react";
import { Check, Copy, Loader2, PlugZap, Trash2 } from "lucide-react";
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
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/dashboard/status-badge";
import type { IntegrationStatus } from "@/lib/integrations";
import { clearCredentialAction, saveCredentialAction } from "@/features/setup/actions";
import type { IntegrationKey } from "@/features/setup/integrations";
import { formatDateTime } from "@/lib/format";
import { ts } from "@/features/setup/strings";

export type CredentialLabels = {
  title: string;
  description: string;
  publicLabel: string;
  secretLabel: string;
  publicPlaceholder?: string;
  where: string;
  /** hidden when the provider has no public half worth editing */
  publicOptional?: boolean;
};

/**
 * One provider's credentials: public id + secret, save / save-and-verify /
 * clear, and where the effective value comes from (app DB vs deployed env).
 */
export function CredentialCard({
  integration,
  status,
  labels,
  redirectUri,
  icon,
}: {
  integration: IntegrationKey;
  status: IntegrationStatus;
  labels: CredentialLabels;
  redirectUri?: string;
  icon?: React.ReactNode;
}) {
  const [publicId, setPublicId] = React.useState(status.publicId ?? "");
  const [secret, setSecret] = React.useState("");
  const [pending, startTransition] = React.useTransition();
  const [copied, setCopied] = React.useState(false);
  const idBase = `cred-${integration.toLowerCase()}`;

  const submit = (verify: boolean) =>
    startTransition(async () => {
      const res = await saveCredentialAction({ key: integration, publicId, secret, verify });
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      setSecret("");
      const v = res.data.verified;
      if (!v) toast.success(ts.saved);
      else if (v.ok) toast.success(v.message);
      else toast.error(v.message);
    });

  const clear = () =>
    startTransition(async () => {
      const res = await clearCredentialAction(integration);
      if (!res.ok) {
        toast.error(res.error);
        return;
      }
      setPublicId("");
      setSecret("");
      toast.success(ts.cleared);
    });

  const copy = async () => {
    if (!redirectUri) return;
    try {
      await navigator.clipboard.writeText(redirectUri);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      toast.error(ts.copy);
    }
  };

  const sourceLabel = status.source === "db" ? ts.sourceDb : status.source === "env" ? ts.sourceEnv : ts.sourceNone;

  return (
    <Card >
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2 text-sm font-semibold">
          {icon}
          {labels.title}
          <StatusBadge kind={status.configured ? "good" : "neutral"}>{status.configured ? ts.modeReal : ts.modeDemo}</StatusBadge>
        </CardTitle>
        <CardDescription className="text-xs">{labels.description}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-5 sm:grid-cols-2">
          <div className="space-y-1.5">
              <Label htmlFor={`${idBase}-public`}>
                {labels.publicLabel}
                {labels.publicOptional ? <span className="ml-1 text-xs font-normal text-muted-foreground">({ts.optionalWord})</span> : null}
              </Label>
              <Input
                id={`${idBase}-public`}
                value={publicId}
                onChange={(e) => setPublicId(e.target.value)}
                placeholder={labels.publicPlaceholder}
                autoComplete="off"
                spellCheck={false}
                className="font-mono text-[13px]"
                disabled={pending}
              />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor={`${idBase}-secret`}>{labels.secretLabel}</Label>
            <Input
              id={`${idBase}-secret`}
              type="password"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              placeholder={status.secretMasked ?? "••••••••"}
              autoComplete="new-password"
              className="font-mono text-[13px]"
              disabled={pending}
            />
            {status.secretMasked ? <p className="text-[11px] text-muted-foreground">{ts.secretKept}</p> : null}
          </div>
        </div>

        <p className="text-xs text-muted-foreground">{labels.where}</p>

        {redirectUri ? (
          <div className="rounded-lg border bg-muted/30 px-3 py-2.5">
            <p className="label-mono text-muted-foreground">{ts.redirectUri}</p>
            <div className="mt-1 flex items-center justify-between gap-2">
              <code className="min-w-0 truncate font-mono text-[12px]">{redirectUri}</code>
              <Button type="button" size="xs" variant="ghost" onClick={copy} aria-label={ts.copy}>
                {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />} {copied ? ts.copied : ts.copy}
              </Button>
            </div>
          </div>
        ) : null}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4">
          <dl className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-muted-foreground">
            <div className="flex gap-1">
              <dt className="label-mono">{ts.status}:</dt>
              <dd>{sourceLabel}</dd>
            </div>
            <div className="flex gap-1">
              <dt className="label-mono">{ts.verifiedAt}:</dt>
              <dd>{status.verifiedAt ? formatDateTime(status.verifiedAt) : ts.neverVerified}</dd>
            </div>
          </dl>
          <div className="flex flex-wrap items-center gap-2">
            {status.source === "db" ? (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button type="button" size="sm" variant="ghost" className="text-muted-foreground hover:text-destructive" disabled={pending}>
                    <Trash2 className="size-3.5" /> {ts.clear}
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{ts.clear}</AlertDialogTitle>
                    <AlertDialogDescription>{ts.clearConfirm}</AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{ts.back}</AlertDialogCancel>
                    <AlertDialogAction variant="destructive" onClick={clear}>
                      {ts.clear}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            ) : null}
            <Button type="button" size="sm" variant="outline" onClick={() => submit(false)} disabled={pending}>
              {ts.save}
            </Button>
            <Button type="button" size="sm" onClick={() => submit(true)} disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : <PlugZap className="size-4" />}
              {pending ? ts.verifying : ts.saveVerify}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
