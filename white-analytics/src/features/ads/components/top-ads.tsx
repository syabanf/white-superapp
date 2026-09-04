"use client";

import * as React from "react";
import { ImageOff } from "lucide-react";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { formatCurrency, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { resultTypeLabel, s } from "@/features/ads/strings";
import type { AdsTopAd } from "@/features/ads/queries";

function Thumb({ src, alt }: { src: string | null; alt: string }) {
  const [failed, setFailed] = React.useState(false);
  if (!src || failed) {
    return (
      <div aria-hidden className="flex size-14 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <ImageOff className="size-5" />
      </div>
    );
  }
  return (
    // eslint-disable-next-line @next/next/no-img-element -- thumbnail eksternal (picsum/CDN Meta), bukan kandidat next/image
    <img
      src={src}
      alt={alt}
      loading="lazy"
      className="size-14 shrink-0 rounded-lg border object-cover"
      onError={() => setFailed(true)}
    />
  );
}

/** Kartu "Iklan terbaik" — 6 iklan dengan biaya per hasil terendah bervolume memadai. */
export function TopAds({ ads, currency }: { ads: AdsTopAd[]; currency: string }) {
  if (ads.length === 0) {
    return <p className="rounded-lg border border-dashed py-10 text-center text-sm text-muted-foreground">{s.topAdsEmpty}</p>;
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {ads.map((ad) => (
        <Card key={ad.id} className="gap-0 p-4">
          <div className="flex items-start gap-3">
            <Thumb src={ad.thumbnailUrl} alt={ad.name} />
            <div className="min-w-0 flex-1">
              <div className="flex items-start justify-between gap-2">
                <p className="truncate text-sm font-medium" title={ad.name}>
                  {ad.name}
                </p>
                <StatusBadge kind={ad.status === "ACTIVE" ? "good" : "neutral"} className="shrink-0 px-1.5 py-0 text-[10px]">
                  {ad.status === "ACTIVE" ? t.common.active : ad.status === "PAUSED" ? t.common.paused : ad.status}
                </StatusBadge>
              </div>
              <p className="mt-0.5 truncate text-xs text-muted-foreground" title={ad.campaignName}>
                {ad.campaignName}
              </p>
            </div>
          </div>
          <dl className="mt-3 grid grid-cols-4 gap-2 border-t pt-3 text-xs">
            <div className="min-w-0">
              <dt className="truncate text-muted-foreground">{t.ads.spend}</dt>
              <dd className="mt-0.5 truncate font-semibold tabular">{formatCurrency(ad.spend, currency, { compact: true })}</dd>
            </div>
            <div className="min-w-0">
              <dt className="truncate text-muted-foreground" title={resultTypeLabel(ad.resultType)}>
                {resultTypeLabel(ad.resultType)}
              </dt>
              <dd className="mt-0.5 truncate font-semibold tabular">{formatNumber(ad.results)}</dd>
            </div>
            <div className="min-w-0">
              <dt className="truncate text-muted-foreground">{t.ads.cpr}</dt>
              <dd className="mt-0.5 truncate font-semibold tabular">{formatCurrency(ad.cpr, currency, { compact: true })}</dd>
            </div>
            <div className="min-w-0">
              <dt className="truncate text-muted-foreground">{t.ads.ctr}</dt>
              <dd className="mt-0.5 truncate font-semibold tabular">{formatPercent(ad.ctr)}</dd>
            </div>
          </dl>
        </Card>
      ))}
    </div>
  );
}
