"use client";

import { Badge } from "@/components/ui/badge";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { Sparkline } from "@/components/dashboard/sparkline";
import { formatNumber } from "@/lib/format";
import { difficultyBand, type DifficultyBand } from "@/lib/metrics/seo-suite";
import type { KeywordIntentKey } from "@/lib/providers/dataforseo/types";
import { cn } from "@/lib/utils";
import { BAND_LABELS, INTENT_LABELS, serpFeatureLabel } from "@/features/seo-suite/lib";

const BAND_KIND: Record<DifficultyBand, StatusKind> = { easy: "good", medium: "warning", hard: "serious", very_hard: "critical" };

/** KD number + band label (status colours are legitimate here: it is a real difficulty rating). */
export function DifficultyBadge({ kd, className }: { kd: number | null | undefined; className?: string }) {
  const band = difficultyBand(kd);
  if (band == null) return <span className="text-muted-foreground">–</span>;
  return (
    <span className={cn("inline-flex items-center justify-end gap-1.5", className)}>
      <span className="tabular font-medium">{formatNumber(kd)}</span>
      <StatusBadge kind={BAND_KIND[band]}>{BAND_LABELS[band]}</StatusBadge>
    </span>
  );
}

export function IntentBadge({ intent }: { intent: KeywordIntentKey | null | undefined }) {
  if (!intent) return <span className="text-muted-foreground">–</span>;
  return (
    <Badge variant="outline" className="font-normal text-muted-foreground">
      {INTENT_LABELS[intent]}
    </Badge>
  );
}

export function SerpFeatureChips({ features, max = 3 }: { features: string[]; max?: number }) {
  if (!features?.length) return <span className="text-muted-foreground">–</span>;
  const shown = features.slice(0, max);
  const rest = features.length - shown.length;
  return (
    <span className="inline-flex flex-wrap gap-1" title={features.map(serpFeatureLabel).join(", ")}>
      {shown.map((f) => (
        <Badge key={f} variant="secondary" className="px-1.5 py-0 text-[10px] font-normal">
          {serpFeatureLabel(f)}
        </Badge>
      ))}
      {rest > 0 ? <span className="text-[10px] text-muted-foreground">+{rest}</span> : null}
    </span>
  );
}

export function TrendCell({ values }: { values: number[] }) {
  return <Sparkline values={values} width={64} height={20} className="inline-block align-middle" />;
}
