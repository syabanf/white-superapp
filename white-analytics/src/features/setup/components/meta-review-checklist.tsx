"use client";

import * as React from "react";
import { CheckCircle2, CircleDashed } from "lucide-react";
import { toast } from "@/lib/toast";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { saveMetaReviewAction } from "@/features/setup/actions";
import {
  META_PERMISSIONS,
  META_REVIEW_STATUSES,
  reviewProgress,
  reviewStatusOf,
  type MetaPermission,
  type MetaReviewState,
  type MetaReviewStatus,
} from "@/features/setup/integrations";
import { ts } from "@/features/setup/strings";
import { cn } from "@/lib/utils";

const STATUS_LABEL: Record<MetaReviewStatus, string> = {
  NOT_REQUESTED: ts.statusNotRequested,
  SUBMITTED: ts.statusSubmitted,
  APPROVED: ts.statusApproved,
};

const UNLOCK_LABEL: Record<MetaPermission["unlocks"], string> = {
  social: ts.unlockSocial,
  publish: ts.unlockPublish,
  ads: ts.unlockAds,
  engagement: ts.unlockEngagement,
};

/** App Review progress, saved on every change (optimistic local state). */
export function MetaReviewChecklist({ initial }: { initial: MetaReviewState }) {
  const [state, setState] = React.useState<MetaReviewState>(initial);
  const [pending, startTransition] = React.useTransition();
  const progress = reviewProgress(state);

  const persist = (next: MetaReviewState) => {
    setState(next);
    startTransition(async () => {
      const res = await saveMetaReviewAction(next);
      if (!res.ok) toast.error(res.error);
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center gap-2 text-sm font-semibold">
          {ts.reviewTitle}
          <span className="label-mono text-muted-foreground">
            {ts.reviewProgress(progress.approved, progress.total)}
          </span>
        </CardTitle>
        <CardDescription className="text-xs">{ts.reviewDesc}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div
          className={cn(
            "flex items-start gap-2 rounded-lg border px-3 py-2.5 text-sm",
            progress.publishReady ? "border-status-good/40 bg-status-good/5" : "bg-muted/30",
          )}
        >
          {progress.publishReady ? (
            <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-status-good" />
          ) : (
            <CircleDashed className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          )}
          <span>{progress.publishReady ? ts.publishReady : ts.publishNotReady}</span>
        </div>

        <ul className="divide-y rounded-lg border">
          {META_PERMISSIONS.map((p) => {
            const value = reviewStatusOf(state, p.name);
            return (
              <li
                key={p.name}
                className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-3 py-2.5"
              >
                <div className="min-w-0">
                  <code className="block truncate font-mono text-[12px]">{p.name}</code>
                  <span className="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                    {ts.unlocks}: {UNLOCK_LABEL[p.unlocks]}
                    {!p.required ? <span className="label-mono">· {ts.optional}</span> : null}
                  </span>
                </div>
                <Select
                  value={value}
                  onValueChange={(v) =>
                    persist({
                      ...state,
                      permissions: { ...state.permissions, [p.name]: v as MetaReviewStatus },
                    })
                  }
                  disabled={pending}
                >
                  <SelectTrigger size="sm" className="w-[150px]" aria-label={`${ts.status} ${p.name}`}>
                    <SelectValue>{STATUS_LABEL[value]}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {META_REVIEW_STATUSES.map((s) => (
                      <SelectItem key={s} value={s}>
                        {STATUS_LABEL[s]}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </li>
            );
          })}
        </ul>

        <div className="grid gap-3 sm:grid-cols-2">
          <label className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5">
            <Label className="text-sm font-normal">{ts.appLive}</Label>
            <Switch
              checked={state.appLive}
              onCheckedChange={(v) => persist({ ...state, appLive: v })}
              disabled={pending}
            />
          </label>
          <label className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5">
            <Label className="text-sm font-normal">{ts.businessVerified}</Label>
            <Switch
              checked={state.businessVerified}
              onCheckedChange={(v) => persist({ ...state, businessVerified: v })}
              disabled={pending}
            />
          </label>
        </div>

        <div className="flex flex-wrap gap-2">
          <StatusBadge kind={progress.adsReady ? "good" : "neutral"}>{ts.unlockAds}</StatusBadge>
          <StatusBadge kind={progress.publishReady ? "good" : "neutral"}>{ts.unlockPublish}</StatusBadge>
        </div>
      </CardContent>
    </Card>
  );
}
