"use client";

import * as React from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { DEFAULT_SEO_PRICES, estimateSeoCost, type SeoCostPrices } from "@/features/setup/integrations";
import { formatCurrency, formatNumber } from "@/lib/format";
import { ts } from "@/features/setup/strings";

function NumberField({ id, label, value, onChange, step = 1, min = 0 }: { id: string; label: string; value: number; onChange: (v: number) => void; step?: number; min?: number }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id} className="text-xs">
        {label}
      </Label>
      <Input
        id={id}
        type="number"
        inputMode="decimal"
        min={min}
        step={step}
        value={Number.isFinite(value) ? value : 0}
        onChange={(e) => onChange(Number(e.target.value))}
        className="tabular"
      />
    </div>
  );
}

/** Pure client-side calculator over `estimateSeoCost`; prefilled from the DB counts. */
export function SeoCostEstimator({ trackedKeywords, competitorDomains, usdIdr }: { trackedKeywords: number; competitorDomains: number; usdIdr: number }) {
  const [kw, setKw] = React.useState(trackedKeywords);
  const [comp, setComp] = React.useState(competitorDomains);
  const [research, setResearch] = React.useState(20);
  const [rate, setRate] = React.useState(usdIdr);
  const [prices, setPrices] = React.useState<SeoCostPrices>(DEFAULT_SEO_PRICES);
  const est = estimateSeoCost({ trackedKeywords: kw, competitorDomains: comp, researchRunsPerMonth: research, usdIdr: rate, prices });

  const usd = (v: number) => `$${formatNumber(v, 2)}`;
  const rows: { label: string; usd: number }[] = [
    { label: ts.costRank, usd: est.rankUsd },
    { label: ts.costBacklinks, usd: est.backlinksUsd },
    { label: ts.costDomains, usd: est.domainsUsd },
    { label: ts.costResearchRow, usd: est.researchUsd },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{ts.costTitle}</CardTitle>
        <CardDescription className="text-xs">{ts.costDesc}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <NumberField id="cost-kw" label={ts.costKeywords} value={kw} onChange={setKw} />
          <NumberField id="cost-comp" label={ts.costCompetitors} value={comp} onChange={setComp} />
          <NumberField id="cost-research" label={ts.costResearch} value={research} onChange={setResearch} />
          <NumberField id="cost-rate" label={ts.costRate} value={rate} onChange={setRate} step={100} />
        </div>

        <details className="rounded-lg border px-3 py-2.5 text-sm">
          <summary className="cursor-pointer text-xs font-medium">{ts.costAdvanced}</summary>
          <div className="mt-3 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <NumberField id="p-serp" label={ts.priceSerp} value={prices.serpPerKeyword} step={0.0005} onChange={(v) => setPrices({ ...prices, serpPerKeyword: v })} />
            <NumberField id="p-bl" label={ts.priceBacklinks} value={prices.backlinkSummaryPerDay} step={0.005} onChange={(v) => setPrices({ ...prices, backlinkSummaryPerDay: v })} />
            <NumberField id="p-dom" label={ts.priceDomain} value={prices.domainOverviewPerDomainPerDay} step={0.005} onChange={(v) => setPrices({ ...prices, domainOverviewPerDomainPerDay: v })} />
            <NumberField id="p-res" label={ts.priceResearch} value={prices.researchPerRun} step={0.01} onChange={(v) => setPrices({ ...prices, researchPerRun: v })} />
          </div>
        </details>

        <div className="-mt-px -ml-px flex flex-wrap border-t border-l">
          {rows.map((r) => (
            <div key={r.label} className="min-w-[140px] flex-1 border-r border-b px-3 py-2.5">
              <p className="label-mono text-muted-foreground">{r.label}</p>
              <p className="mt-1 text-sm font-semibold tabular">{usd(r.usd)}</p>
            </div>
          ))}
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
