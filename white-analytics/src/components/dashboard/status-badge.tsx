import { AlertTriangle, CheckCircle2, CircleAlert, Info, type LucideIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { CwvRating } from "@/lib/metrics";

export type StatusKind = "good" | "warning" | "serious" | "critical" | "neutral" | "info";

const MAP: Record<StatusKind, { color: string; icon: LucideIcon }> = {
  good: { color: "var(--status-good)", icon: CheckCircle2 },
  warning: { color: "var(--status-warning)", icon: AlertTriangle },
  serious: { color: "var(--status-serious)", icon: AlertTriangle },
  critical: { color: "var(--status-critical)", icon: CircleAlert },
  neutral: { color: "var(--muted-foreground)", icon: Info },
  info: { color: "var(--chart-1)", icon: Info },
};

/** Status colors are reserved for meaning and always ship with an icon + label. */
export function StatusBadge({ kind, children, className }: { kind: StatusKind; children: React.ReactNode; className?: string }) {
  const { color, icon: Icon } = MAP[kind];
  return (
    <Badge
      variant="outline"
      className={cn("gap-1 border-transparent font-medium", className)}
      style={{ background: `color-mix(in oklab, ${color} 12%, transparent)`, color: `color-mix(in oklab, ${color} 85%, var(--foreground))` }}
    >
      <Icon className="size-3" />
      {children}
    </Badge>
  );
}

export function ratingToKind(r: CwvRating | null | undefined): StatusKind {
  if (r === "good") return "good";
  if (r === "needs-improvement") return "warning";
  if (r === "poor") return "critical";
  return "neutral";
}
