"use client";

import { useState, useTransition } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Check, Radar, ShieldCheck, Sparkles } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { InfoHint } from "@/components/dashboard/info-hint";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { formatDateShort, formatRelative } from "@/lib/format";
import { checkAdAnomalies, resolveAdAlert } from "@/features/ads-automation/actions";
import type { AlertItem, AlertsPanelData } from "@/features/ads-automation/queries";
import { aa, ruleLabel } from "@/features/ads-automation/strings";

const KIND: Record<string, StatusKind> = { CRITICAL: "critical", WARNING: "warning", INFO: "info" };
const VISIBLE = 3;

function CheckNowButton({ slug }: { slug: string }) {
  const router = useRouter();
  const [pending, start] = useTransition();
  const run = () =>
    start(async () => {
      const res = await checkAdAnomalies(slug);
      if (res.ok) {
        toast.success(aa.alerts.checked(res.data.created));
        router.refresh();
      } else toast.error(res.error);
    });
  return (
    <Button variant="outline" size="sm" onClick={run} disabled={pending}>
      <Radar className={pending ? "size-4 animate-pulse" : "size-4"} />
      {pending ? aa.alerts.checking : aa.alerts.checkNow}
    </Button>
  );
}

function ResolveButton({ slug, id }: { slug: string; id: string }) {
  const router = useRouter();
  const [pending, start] = useTransition();
  const run = () =>
    start(async () => {
      const res = await resolveAdAlert(slug, id);
      if (res.ok) {
        toast.success(aa.alerts.resolved);
        router.refresh();
      } else toast.error(res.error);
    });
  return (
    <Button variant="ghost" size="icon-sm" onClick={run} disabled={pending} aria-label={aa.alerts.resolve} title={aa.alerts.resolve}>
      <Check className="size-4" />
    </Button>
  );
}

function AlertRow({ alert, slug, canManage }: { alert: AlertItem; slug: string; canManage: boolean }) {
  return (
    <li className="flex items-start gap-3 py-3 first:pt-0 last:pb-0">
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
          <StatusBadge kind={KIND[alert.severity] ?? "neutral"}>{aa.alerts.severity[alert.severity] ?? alert.severity}</StatusBadge>
          <span className="text-sm font-medium">{ruleLabel(alert.rule)}</span>
          <span className="min-w-0 truncate text-xs text-muted-foreground">
            {alert.campaignName ?? aa.alerts.account} · {formatDateShort(alert.date)}
            {alert.days > 1 ? ` · ${alert.days}×` : ""}
          </span>
        </div>
        <p className="text-sm text-muted-foreground">{alert.message}</p>
      </div>
      {canManage ? <ResolveButton slug={slug} id={alert.id} /> : null}
    </li>
  );
}

/** "Peringatan otomatis": compact panel at the top of the Ads page. */
export function AlertsPanel({ slug, canManage, data }: { slug: string; canManage: boolean; data: AlertsPanelData }) {
  const [expanded, setExpanded] = useState(false);
  const shown = expanded ? data.alerts : data.alerts.slice(0, VISIBLE);
  const hidden = data.alerts.length - shown.length;

  return (
    <Card>
      <CardHeader className="flex min-h-8 flex-row flex-wrap items-center justify-between gap-3 space-y-0">
        <div className="flex min-w-0 items-center gap-1.5">
          <CardTitle className="truncate text-[15px] font-semibold tracking-[-0.015em]">{aa.alerts.title}</CardTitle>
          <InfoHint>{aa.alerts.hint}</InfoHint>
        </div>
        <div className="flex shrink-0 items-center gap-3">
          <span className="hidden text-xs text-muted-foreground sm:inline">
            {data.lastCheckedAt ? `${aa.alerts.lastChecked} ${formatRelative(data.lastCheckedAt)}` : aa.alerts.neverChecked}
          </span>
          {canManage ? <CheckNowButton slug={slug} /> : null}
        </div>
      </CardHeader>
      <CardContent className="min-w-0">
        {data.alerts.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <span className="flex size-9 items-center justify-center rounded-full bg-muted text-muted-foreground">
              <ShieldCheck className="size-4" />
            </span>
            <p className="text-sm font-medium">{aa.alerts.emptyTitle}</p>
            <p className="text-xs text-muted-foreground">{aa.alerts.emptyDesc}</p>
          </div>
        ) : (
          <>
            <ul className="divide-y">
              {shown.map((a) => (
                <AlertRow key={a.id} alert={a} slug={slug} canManage={canManage} />
              ))}
            </ul>
            {data.alerts.length > VISIBLE ? (
              <Button variant="ghost" size="xs" className="mt-2" onClick={() => setExpanded((v) => !v)}>
                {hidden > 0 ? aa.alerts.more(hidden) : aa.alerts.showLess}
              </Button>
            ) : null}
          </>
        )}
        <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t pt-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1.5">
            <Sparkles className="size-3.5" />
            {data.weeklyInsight
              ? `${aa.alerts.weeklyInsight} · ${formatRelative(data.weeklyInsight.createdAt)}`
              : aa.alerts.weeklyInsightNone}
          </span>
          {data.weeklyInsight ? (
            <Link href={`/clients/${slug}/reports?tab=insight&preset=7d`} className="font-medium text-brand hover:underline">
              {aa.alerts.weeklyInsightOpen}
            </Link>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}
