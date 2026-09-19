"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { estimateApifyCost } from "@/features/setup/integrations";
import { formatCurrency, formatNumber } from "@/lib/format";
import { ts } from "@/features/setup/strings";

function Field({ id, label, value, onChange, step = 1 }: { id: string; label: string; value: number; onChange: (v: number) => void; step?: number }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs">
        {label}
      </Label>
      <Input id={id} type="number" inputMode="decimal" min={0} step={step} value={Number.isFinite(value) ? value : 0} onChange={(e) => onChange(Number(e.target.value))} className="tabular" />
    </div>
  );
}

export function ApifyCostEstimator({ trackedKeywords, socialAccounts, usdIdr }: { trackedKeywords: number; socialAccounts: number; usdIdr: number }) {
  const [kw, setKw] = React.useState(trackedKeywords);
  const [acc, setAcc] = React.useState(socialAccounts);
  const [posts, setPosts] = React.useState(30);
  const [rate, setRate] = React.useState(usdIdr);
  const est = estimateApifyCost({ trackedKeywords: kw, socialAccounts: acc, postsPerAccount: posts, usdIdr: rate });
  const usd = (v: number) => `$${formatNumber(v, 2)}`;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{ts.apifyCostTitle}</CardTitle>
        <CardDescription className="text-xs">{ts.apifyCostDesc}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          <Field id="ap-kw" label={ts.costKeywords} value={kw} onChange={setKw} />
          <Field id="ap-acc" label={ts.apifyAccounts} value={acc} onChange={setAcc} />
          <Field id="ap-posts" label={ts.apifyPosts} value={posts} onChange={setPosts} />
          <Field id="ap-rate" label={ts.costRate} value={rate} onChange={setRate} step={100} />
        </div>
        <div className="-mt-px -ml-px flex flex-wrap border-t border-l">
          <div className="min-w-[140px] flex-1 border-r border-b px-3 py-2.5">
            <p className="label-mono text-muted-foreground">{ts.apifyCostSerp}</p>
            <p className="mt-1 text-sm font-semibold tabular">{usd(est.serpUsd)}</p>
          </div>
          <div className="min-w-[140px] flex-1 border-r border-b px-3 py-2.5">
            <p className="label-mono text-muted-foreground">{ts.apifyCostSocial}</p>
            <p className="mt-1 text-sm font-semibold tabular">{usd(est.socialUsd)}</p>
          </div>
          <div className="min-w-[180px] flex-1 border-r border-b bg-accent/50 px-3 py-2.5">
            <p className="label-mono text-brand">{ts.costTotal}</p>
            <p className="mt-1 text-lg font-bold tracking-[-0.02em] tabular">{usd(est.totalUsd)}</p>
            <p className="text-xs text-muted-foreground tabular">≈ {formatCurrency(est.totalIdr, "IDR")}{ts.perMonth}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
