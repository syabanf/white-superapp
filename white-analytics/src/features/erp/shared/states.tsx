import type { ReactNode } from "react";
import { AlertTriangle, Inbox, SearchX } from "lucide-react";
import { EmptyState } from "@/components/dashboard/empty-state";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { ss } from "./strings";

/** Empty list. `filtered` switches to the "no match" copy. */
export function ErpEmpty({
  title,
  description,
  action,
  icon,
  filtered,
  compact,
}: {
  title?: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  icon?: ReactNode;
  filtered?: boolean;
  compact?: boolean;
}) {
  return (
    <EmptyState
      compact={compact}
      state={filtered ? "zero" : "empty"}
      icon={icon ?? (filtered ? <SearchX className="size-5" /> : <Inbox className="size-5" />)}
      title={title ?? (filtered ? ss.noResults : ss.noData)}
      description={description}
      action={action}
    />
  );
}

/** Inline error for a section whose fetch failed (the page keeps rendering). */
export function ErpError({ message, action, compact }: { message?: string; action?: ReactNode; compact?: boolean }) {
  return (
    <EmptyState
      compact={compact}
      state="error"
      icon={<AlertTriangle className="size-5" />}
      title={ss.errorTitle}
      description={message ?? ss.unknownError}
      action={action}
    />
  );
}

/** Section-level failure from `load()`: the offline banner for NETWORK, an inline error otherwise. */
export function ErpLoadError({ error, compact }: { error: { code: string; message: string }; compact?: boolean }) {
  if (error.code === "NETWORK") return <OfflineNotice message={error.message} />;
  return <ErpError message={error.message} compact={compact} />;
}
