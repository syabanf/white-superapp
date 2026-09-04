"use client";

import * as React from "react";
import Link from "next/link";
import { Check, Copy, ExternalLink, KeyRound, Settings } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

const MODULE_LABEL: Record<string, string> = { SOCIAL: t.overview.socialCard, SEO: t.overview.seoCard, ADS: t.overview.adsCard };

/** Read-only summary of the public share settings; the form itself lives in Settings. */
export function ShareSummaryCard({
  slug,
  enabled,
  pinSet,
  modules,
  shareUrl,
}: {
  slug: string;
  enabled: boolean;
  pinSet: boolean;
  modules: string[];
  shareUrl: string;
}) {
  const [copied, setCopied] = React.useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopied(true);
      toast.success(rs.share.urlCopied);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      toast.error(t.common.error);
    }
  };

  return (
    <Card className="max-w-2xl">
      <CardHeader>
        <CardTitle className="text-base">{rs.share.statusTitle}</CardTitle>
        <CardDescription>{rs.share.statusDesc}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge kind={enabled ? "good" : "neutral"}>{enabled ? rs.share.active : rs.share.inactive}</StatusBadge>
          <StatusBadge kind={pinSet ? "info" : "neutral"}>
            {pinSet ? rs.share.pinSet : rs.share.pinNone}
          </StatusBadge>
        </div>

        <div className="space-y-2">
          <Label>{t.reports.shareUrl}</Label>
          <div className="flex items-center gap-2">
            <Input readOnly value={shareUrl} className="font-mono text-xs" onFocus={(e) => e.currentTarget.select()} />
            <Button type="button" variant="outline" size="icon" className="size-8 shrink-0" onClick={copy} aria-label={t.common.copy}>
              {copied ? <Check className="size-4 text-[var(--status-good)]" /> : <Copy className="size-4" />}
            </Button>
            <Button asChild variant="outline" size="icon" className="size-8 shrink-0" aria-label={rs.share.openInNewTab}>
              <a href={`/share/${slug}`} target="_blank" rel="noreferrer">
                <ExternalLink className="size-4" />
              </a>
            </Button>
          </div>
        </div>

        <div className="space-y-2">
          <Label>{rs.share.modulesShown}</Label>
          <div className="flex flex-wrap gap-1.5">
            {modules.length === 0 ? (
              <span className="text-sm text-muted-foreground">{t.common.none}</span>
            ) : (
              modules.map((m) => (
                <Badge key={m} variant="secondary">
                  {MODULE_LABEL[m] ?? m}
                </Badge>
              ))
            )}
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2 border-t pt-4">
          <Button asChild variant="outline" size="sm">
            <Link href={`/clients/${slug}/settings`}>
              <Settings className="size-4" /> {rs.share.changeSettings}
            </Link>
          </Button>
          {pinSet ? (
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <KeyRound className="size-3.5" /> {t.reports.sharePinHint}
            </p>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}
