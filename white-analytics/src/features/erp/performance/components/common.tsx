"use client";

import * as React from "react";
import { Loader2 } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { StatusKind } from "@/components/dashboard/status-badge";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { useUrlFilters } from "@/features/erp/shared/url-state";
import { cn } from "@/lib/utils";
import { formatDecimal } from "@/lib/format";
import { scoreTone } from "../lib";
import { sp } from "../strings";

const TASK_KINDS: Record<string, StatusKind> = {
  pending: "neutral",
  in_progress: "info",
  done: "warning",
  approved: "good",
  rejected: "critical",
};

export function TaskStatusBadge({ status, overdue }: { status: string; overdue?: boolean }) {
  if (overdue) return <ErpStatusBadge status="overdue" labels={{ overdue: sp.tasks.overdueTag }} kinds={{ overdue: "critical" }} />;
  return <ErpStatusBadge status={status} labels={sp.taskStatus} kinds={TASK_KINDS} />;
}

const PRIORITY_KINDS: Record<string, StatusKind> = { low: "neutral", high: "warning", urgent: "critical" };

/** Hidden for normal priority to keep rows quiet. */
export function PriorityBadge({ priority }: { priority: string }) {
  if (priority === "normal") return null;
  return <ErpStatusBadge status={priority} labels={sp.priority} kinds={PRIORITY_KINDS} />;
}

export const DRAFT_FINAL_KINDS: Record<string, StatusKind> = { draft: "neutral", final: "good" };
export const DRAFT_FINAL_LABELS: Record<string, string> = { draft: sp.common.draft, final: sp.common.final };

export function Score({ value, digits = 1, className }: { value?: number | null; digits?: number; className?: string }) {
  return (
    <span className={cn("font-semibold tabular", scoreTone(value), className)}>
      {value == null ? "–" : formatDecimal(value, digits)}
    </span>
  );
}

/** Select bound to a URL param without an "all" option; empty param means `fallback`. */
export function UrlSelect({
  param,
  options,
  fallback,
  label,
  className,
}: {
  param: string;
  options: { value: string; label: string }[];
  fallback: string;
  label: string;
  className?: string;
}) {
  const { values, set } = useUrlFilters([param]);
  return (
    <Select value={values[param] ?? fallback} onValueChange={(v) => set({ [param]: v === fallback ? null : v })}>
      <SelectTrigger className={cn("min-w-28", className)} aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Single date bound to a URL param ("YYYY-MM-DD"); empty param means `fallback`. */
export function UrlDate({ param = "date", fallback, className }: { param?: string; fallback: string; className?: string }) {
  const { values, set } = useUrlFilters([param]);
  return (
    <DatePicker
      value={values[param] ?? fallback}
      onChange={(v) => set({ [param]: v && v !== fallback ? v : null })}
      className={cn("w-44", className)}
    />
  );
}

/** Button that asks for confirmation before running `onConfirm`. */
export function ConfirmAction({
  trigger,
  title,
  description,
  confirmLabel,
  destructive,
  onConfirm,
}: {
  trigger: React.ReactNode;
  title: string;
  description?: string;
  confirmLabel: string;
  destructive?: boolean;
  onConfirm: () => Promise<unknown> | void;
}) {
  const [open, setOpen] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description ? <AlertDialogDescription>{description}</AlertDialogDescription> : null}
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={busy}>{sp.common.cancel}</AlertDialogCancel>
          <AlertDialogAction
            variant={destructive ? "destructive" : "default"}
            disabled={busy}
            onClick={async (e) => {
              e.preventDefault();
              setBusy(true);
              try {
                await onConfirm();
                setOpen(false);
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? <Loader2 className="size-4 animate-spin" /> : null}
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Labelled form row. */
export function FormRow({
  label,
  htmlFor,
  hint,
  error,
  className,
  children,
}: {
  label: string;
  htmlFor?: string;
  hint?: string;
  error?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-1.5", className)}>
      <label htmlFor={htmlFor} className="text-sm font-medium">
        {label}
      </label>
      {children}
      {error ? (
        <p className="text-xs text-destructive">{error}</p>
      ) : hint ? (
        <p className="text-xs text-muted-foreground">{hint}</p>
      ) : null}
    </div>
  );
}
