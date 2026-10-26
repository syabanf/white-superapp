import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { ss } from "./strings";

const DEFAULT_KIND: Record<string, StatusKind> = {
  DRAFT: "neutral",
  PENDING: "warning",
  SUBMITTED: "info",
  IN_REVIEW: "info",
  IN_PROGRESS: "info",
  OPEN: "info",
  APPROVED: "good",
  POSTED: "good",
  PAID: "good",
  SUCCESS: "good",
  COMPLETED: "good",
  DONE: "good",
  ACTIVE: "good",
  PARTIAL: "warning",
  PARTIALLY_PAID: "warning",
  UNPAID: "warning",
  OVERDUE: "critical",
  REJECTED: "critical",
  FAILED: "critical",
  VOID: "serious",
  VOIDED: "serious",
  CANCELLED: "neutral",
  CANCELED: "neutral",
  CLOSED: "neutral",
  LOCKED: "neutral",
  INACTIVE: "neutral",
  EXPIRED: "neutral",
};

/** Proto enums often carry a type prefix ("CLAIM_STATUS_APPROVED"); keep the last known word. */
export function normalizeStatus(status: string): string {
  const upper = status.toUpperCase();
  const known = Object.keys(DEFAULT_KIND).find((k) => upper === k || upper.endsWith(`_${k}`));
  return known ?? upper;
}

/**
 * Badge for any workflow state. Pass `labels`/`kinds` to override or extend
 * the shared map for module-specific states.
 */
export function ErpStatusBadge({
  status,
  labels,
  kinds,
  className,
}: {
  status: string | null | undefined;
  labels?: Record<string, string>;
  kinds?: Record<string, StatusKind>;
  className?: string;
}) {
  if (!status) return null;
  const key = normalizeStatus(status);
  const label = labels?.[status] ?? labels?.[key] ?? ss.status[key] ?? key.replace(/_/g, " ").toLowerCase();
  const kind = kinds?.[status] ?? kinds?.[key] ?? DEFAULT_KIND[key] ?? "neutral";
  return (
    <StatusBadge kind={kind} className={className}>
      {label}
    </StatusBadge>
  );
}
