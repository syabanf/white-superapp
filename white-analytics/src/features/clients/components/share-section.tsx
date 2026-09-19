"use client";

import * as React from "react";
import { Check, Copy, ExternalLink, Loader2, ShieldCheck } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { updateShareSettingsAction } from "@/features/clients/actions";
import { SHARE_MODULE_KEYS, type ShareModuleKey } from "@/features/clients/constants";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

const MODULE_LABEL: Record<ShareModuleKey, string> = {
  SOCIAL: t.overview.socialCard,
  SEO: t.overview.seoCard,
  ADS: t.overview.adsCard,
};

export function ShareSection({
  clientId,
  slug,
  shareEnabled,
  hasPin,
  shareModules,
  appUrl,
  canManage,
}: {
  clientId: string;
  slug: string;
  shareEnabled: boolean;
  hasPin: boolean;
  shareModules: string[];
  appUrl: string;
  canManage: boolean;
}) {
  const [enabled, setEnabled] = React.useState(shareEnabled);
  const [pin, setPin] = React.useState("");
  const [modules, setModules] = React.useState<ShareModuleKey[]>(
    SHARE_MODULE_KEYS.filter((k) => shareModules.includes(k)),
  );
  const [copied, setCopied] = React.useState(false);
  const [pending, startTransition] = React.useTransition();

  const shareUrl = `${appUrl.replace(/\/$/, "")}/share/${slug}`;

  const toggleModule = (key: ShareModuleKey, checked: boolean) => {
    setModules((prev) => (checked ? [...new Set([...prev, key])] : prev.filter((k) => k !== key)));
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      toast.success(t.common.copied);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(t.common.error);
    }
  };

  const save = () => {
    startTransition(async () => {
      const res = await updateShareSettingsAction(clientId, { shareEnabled: enabled, pin, modules });
      if (!res.ok) {
        toast.error(res.fieldErrors?.pin?.[0] ?? res.fieldErrors?.modules?.[0] ?? res.error);
        return;
      }
      setPin("");
      toast.success(t.reports.shareSaved);
    });
  };

  return (
    <Card>
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <Label htmlFor="share-enabled" className="text-sm font-medium">
              {t.reports.shareEnabled}
            </Label>
            <p className="mt-0.5 text-xs text-muted-foreground">{tc.settings.shareDisabledHint}</p>
          </div>
          <Switch
            id="share-enabled"
            checked={enabled}
            onCheckedChange={setEnabled}
            disabled={!canManage || pending}
          />
        </div>

        <div className="grid gap-5 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="share-pin" className="flex items-center gap-2">
              {t.reports.sharePin}
              {hasPin ? (
                <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-xs">
                  <ShieldCheck className="size-3" /> {tc.settings.pinSet}
                </Badge>
              ) : null}
            </Label>
            <Input
              id="share-pin"
              value={pin}
              onChange={(e) => setPin(e.target.value.replace(/\D/g, "").slice(0, 6))}
              inputMode="numeric"
              maxLength={6}
              placeholder={tc.settings.pinPlaceholder}
              disabled={!canManage || pending}
              className="w-40 font-mono tracking-[0.3em]"
            />
            <p className="text-xs text-muted-foreground">
              {t.reports.sharePinHint}
              {hasPin ? ` ${tc.settings.pinRemoveHint}` : ""}
            </p>
          </div>

          <div className="space-y-2">
            <span className="text-sm font-medium">{t.reports.shareModules}</span>
            <div className="space-y-2">
              {SHARE_MODULE_KEYS.map((key) => (
                <label key={key} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={modules.includes(key)}
                    onCheckedChange={(v) => toggleModule(key, v === true)}
                    disabled={!canManage || pending}
                    aria-label={MODULE_LABEL[key]}
                  />
                  {MODULE_LABEL[key]}
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="share-url">{t.reports.shareUrl}</Label>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              id="share-url"
              readOnly
              value={shareUrl}
              className="w-auto min-w-64 flex-1 font-mono text-xs"
              onFocus={(e) => e.currentTarget.select()}
            />
            <Button type="button" variant="outline" onClick={copy}>
              {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />} {t.common.copy}
            </Button>
            <Button asChild variant="outline">
              <a href={shareUrl} target="_blank" rel="noreferrer noopener">
                <ExternalLink className="size-3.5" /> {t.reports.openShare}
              </a>
            </Button>
          </div>
        </div>
      </CardContent>
      {canManage ? (
        <CardFooter className="justify-end">
          <Button size="sm" onClick={save} disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            Simpan akses
          </Button>
        </CardFooter>
      ) : null}
    </Card>
  );
}
