"use client";

import * as React from "react";
import { ImageOff } from "lucide-react";
import { InfoHint } from "@/components/dashboard/info-hint";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { contrastLevel, contrastRatio, formatHashtags, normalizeHex, readableTextOn } from "@/features/brand/lib";
import { b } from "@/features/brand/strings";
import { formatDecimal } from "@/lib/format";

export type BrandPreview = {
  clientName: string;
  primaryColor: string;
  secondaryColor: string;
  accentColors: string[];
  headingFont: string;
  bodyFont: string;
  logoUrl: string;
  toneOfVoice: string;
  hashtags: string[];
};

const LEVEL: Record<ReturnType<typeof contrastLevel>, { kind: StatusKind; label: string }> = {
  good: { kind: "good", label: b.contrastGood },
  "large-only": { kind: "warning", label: b.contrastLarge },
  poor: { kind: "critical", label: b.contrastPoor },
};

function Logo({ url, color }: { url: string; color: string }) {
  const [failedUrl, setFailedUrl] = React.useState<string | null>(null);
  const failed = failedUrl === url;
  if (!url) return null;
  return failed ? (
    <span className="flex size-12 items-center justify-center rounded-full bg-white/15" title={b.logoPreviewFail} style={{ color }}>
      <ImageOff className="size-4" />
    </span>
  ) : (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={url} alt="" loading="lazy" onError={() => setFailedUrl(url)} className="size-12 rounded-full bg-white object-contain p-1.5" />
  );
}

/** Compact live preview of the kit: a poster-like swatch, palette chips, fonts and tone. */
export function BrandCard({ brand }: { brand: BrandPreview }) {
  const primary = normalizeHex(brand.primaryColor) ?? "#2b2265";
  const secondary = normalizeHex(brand.secondaryColor) ?? "#fbeaf2";
  const accents = brand.accentColors.map((c) => normalizeHex(c)).filter((c): c is string => Boolean(c));
  const ink = readableTextOn(primary);
  const ratio = contrastRatio(primary, ink);
  const level = LEVEL[contrastLevel(ratio)];
  const heading = brand.headingFont.trim();
  const body = brand.bodyFont.trim();
  const palette = [primary, secondary, ...accents];

  return (
    <div className="flex flex-col gap-5 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
      <div
        className="flex aspect-[4/3] flex-col justify-between rounded-[1.25rem] p-5"
        style={{ background: `linear-gradient(150deg, ${primary} 0%, ${primary} 55%, ${accents[0] ?? secondary} 140%)`, color: ink }}
      >
        <div className="flex items-start justify-between gap-3">
          <span className="text-xs font-medium opacity-80">{brand.clientName}</span>
          <Logo url={brand.logoUrl.trim()} color={ink} />
        </div>
        <div className="min-w-0">
          <p
            className="line-clamp-2 text-2xl leading-tight font-bold"
            style={{ fontFamily: heading ? `"${heading}", var(--font-sans)` : undefined }}
          >
            {brand.clientName}
          </p>
          <p className="mt-1 line-clamp-2 text-sm opacity-85" style={{ fontFamily: body ? `"${body}", var(--font-sans)` : undefined }}>
            {brand.toneOfVoice.trim() || b.noTone}
          </p>
          <span className="mt-3 inline-flex rounded-full px-3 py-1 text-xs font-semibold" style={{ background: secondary, color: readableTextOn(secondary) }}>
            {formatHashtags(brand.hashtags.slice(0, 2)) || b.previewTitle}
          </span>
        </div>
      </div>

      <dl className="grid gap-4 text-sm">
        <div>
          <dt className="label-mono mb-2 text-muted-foreground">{b.palette}</dt>
          <dd className="flex flex-wrap gap-2">
            {palette.map((c, i) => (
              <span key={`${c}-${i}`} className="inline-flex items-center gap-1.5 rounded-full bg-muted/60 py-1 pr-2.5 pl-1 text-xs tabular">
                <span className="size-5 rounded-full ring-1 ring-foreground/10" style={{ background: c }} />
                {c}
              </span>
            ))}
          </dd>
        </div>
        <div>
          <dt className="label-mono mb-1 text-muted-foreground">{b.fonts}</dt>
          <dd className="space-y-0.5">
            <p className="font-semibold" style={{ fontFamily: heading ? `"${heading}", var(--font-sans)` : undefined }}>
              {heading || b.defaultFont} <span className="font-normal text-muted-foreground">· {b.headingFont}</span>
            </p>
            <p style={{ fontFamily: body ? `"${body}", var(--font-sans)` : undefined }}>
              {body || b.defaultFont} <span className="text-muted-foreground">· {b.bodyFont}</span>
            </p>
          </dd>
        </div>
        <div>
          <dt className="label-mono mb-1 flex items-center gap-1.5 text-muted-foreground">
            {b.contrast}
            <InfoHint>{b.contrastHint(formatDecimal(ratio, 1))}</InfoHint>
          </dt>
          <dd>
            <StatusBadge kind={level.kind}>
              {level.label} · {formatDecimal(ratio, 1)}:1
            </StatusBadge>
          </dd>
        </div>
      </dl>
    </div>
  );
}
