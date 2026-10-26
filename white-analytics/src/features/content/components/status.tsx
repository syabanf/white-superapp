import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { DESIGN_TONE, ITEM_TONE, PLAN_TONE, TONE_VAR, type Tone } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { DesignStatus, ItemStatus, PlanStatus } from "@/features/content/types";

/** Tinted status pill, same treatment as the publishing status badge. */
function TonePill({ tone, label, className }: { tone: Tone; label: string; className?: string }) {
  const color = TONE_VAR[tone];
  return (
    <Badge
      variant="outline"
      className={cn("gap-1.5 border-transparent font-medium", className)}
      style={{ background: `color-mix(in oklab, ${color} 12%, transparent)`, color: `color-mix(in oklab, ${color} 85%, var(--foreground))` }}
    >
      <span aria-hidden className="size-1.5 rounded-full" style={{ background: color }} />
      {label}
    </Badge>
  );
}

export function ItemStatusBadge({ status, className }: { status: ItemStatus; className?: string }) {
  return <TonePill tone={ITEM_TONE[status] ?? "neutral"} label={c.itemStatus[status] ?? status} className={className} />;
}

export function PlanStatusBadge({ status, className }: { status: PlanStatus; className?: string }) {
  return <TonePill tone={PLAN_TONE[status] ?? "neutral"} label={c.planStatus[status] ?? status} className={className} />;
}

export function DesignStatusBadge({ status, className }: { status: DesignStatus; className?: string }) {
  return <TonePill tone={DESIGN_TONE[status] ?? "neutral"} label={c.designStatus[status] ?? status} className={className} />;
}

export function ItemStatusDot({ status }: { status: ItemStatus }) {
  return <span aria-hidden className="inline-block size-2 shrink-0 rounded-full" style={{ background: TONE_VAR[ITEM_TONE[status] ?? "neutral"] }} />;
}
