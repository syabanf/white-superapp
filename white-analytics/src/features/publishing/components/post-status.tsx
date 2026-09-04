import { CalendarClock, CheckCircle2, CircleAlert, CircleDashed, Clock, Eye, Loader2, XCircle, type LucideIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { STATUS_ORDER, STATUS_TONE, type PostStatus, type StatusTone } from "@/features/publishing/lib";
import { p } from "@/features/publishing/strings";

/**
 * PostStatus → colour token. Status colours are reserved for meaning: a bare
 * dot is only used inside calendar chips, and the legend/badge always pairs
 * the colour with an icon + label.
 */
const TONE_VAR: Record<StatusTone, string> = {
  neutral: "var(--muted-foreground)",
  warning: "var(--status-warning)",
  info: "var(--chart-3)",
  brand: "var(--chart-1)",
  good: "var(--status-good)",
  critical: "var(--status-critical)",
};

const ICON: Record<PostStatus, LucideIcon> = {
  DRAFT: CircleDashed,
  IN_REVIEW: Eye,
  APPROVED: CheckCircle2,
  SCHEDULED: CalendarClock,
  PUBLISHING: Loader2,
  PUBLISHED: CheckCircle2,
  FAILED: CircleAlert,
  REJECTED: XCircle,
};

export function statusColor(status: PostStatus): string {
  return TONE_VAR[STATUS_TONE[status]];
}

export function statusLabel(status: PostStatus): string {
  return p.status[status];
}

export function StatusDot({ status, className }: { status: PostStatus; className?: string }) {
  return <span aria-hidden className={cn("inline-block size-2 shrink-0 rounded-full", className)} style={{ background: statusColor(status) }} />;
}

export function PostStatusBadge({ status, className }: { status: PostStatus; className?: string }) {
  const Icon = ICON[status];
  const color = statusColor(status);
  return (
    <Badge
      variant="outline"
      className={cn("gap-1 border-transparent font-medium", className)}
      style={{ background: `color-mix(in oklab, ${color} 12%, transparent)`, color: `color-mix(in oklab, ${color} 85%, var(--foreground))` }}
    >
      <Icon className={cn("size-3", status === "PUBLISHING" && "animate-spin")} />
      {statusLabel(status)}
    </Badge>
  );
}

export function StatusLegend({ className, statuses = STATUS_ORDER }: { className?: string; statuses?: PostStatus[] }) {
  return (
    <div className={cn("flex flex-wrap items-center gap-x-4 gap-y-1.5", className)} aria-label={p.legend}>
      {statuses.map((s) => {
        const Icon = ICON[s];
        return (
          <span key={s} className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <StatusDot status={s} />
            <Icon className="size-3" style={{ color: statusColor(s) }} />
            {statusLabel(s)}
          </span>
        );
      })}
    </div>
  );
}

export { Clock as ClockIcon };
