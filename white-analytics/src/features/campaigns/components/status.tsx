import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { c } from "@/features/campaigns/strings";

const KIND: Record<string, StatusKind> = { DRAFT: "neutral", READY: "info", SUBMITTED: "good" };

/** Draft lifecycle badge (icon + label, status colours only for real status). */
export function DraftStatusBadge({ status }: { status: string }) {
  return <StatusBadge kind={KIND[status] ?? "neutral"}>{c.status[status] ?? status}</StatusBadge>;
}

/** Shown on every simulated submission: nothing was created in Meta. */
export function SimulatedBadge({ short = false }: { short?: boolean }) {
  return (
    <StatusBadge kind="warning" className="max-w-full whitespace-normal text-left">
      {short ? c.simulatedShort : c.simulatedBadge}
    </StatusBadge>
  );
}
