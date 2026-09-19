import type { ReactNode } from "react";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { cn } from "@/lib/utils";

export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
  compact,
  state = "empty",
}: {
  icon?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
  compact?: boolean;
  state?: "empty" | "zero" | "error" | "disconnected";
}) {
  return (
    <Empty
      role={state === "error" ? "alert" : "status"}
      data-state={state}
      className={cn(
        "rounded-xl border border-dashed bg-card/50",
        state === "error" && "border-destructive/40 bg-destructive/5",
        state === "disconnected" && "border-amber-500/40",
        compact ? "py-8" : "py-14",
        className,
      )}
    >
      <EmptyHeader>
        {icon ? <EmptyMedia variant="icon">{icon}</EmptyMedia> : null}
        <EmptyTitle>{title}</EmptyTitle>
        {description ? <EmptyDescription>{description}</EmptyDescription> : null}
      </EmptyHeader>
      {action ? <EmptyContent>{action}</EmptyContent> : null}
    </Empty>
  );
}
