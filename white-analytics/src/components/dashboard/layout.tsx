import type { ComponentProps, ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * Layout helpers for the alignment system (rules in LAYOUT.md). Every gap and padding
 * comes from the CSS tokens in globals.css (--gap, --gap-inner, --card-pad, --control-h),
 * so pages line up without their own spacing classes.
 */

/** Filter bar or table toolbar: search and filters on the left, actions pushed right. */
export function Toolbar({
  children,
  className,
  label,
  ...rest
}: ComponentProps<"div"> & {
  /** accessible name; set it when the toolbar holds filters */
  label?: string;
}) {
  return (
    <div
      role={label ? "toolbar" : undefined}
      aria-label={label}
      {...rest}
      className={cn("flex min-w-0 flex-wrap items-center gap-(--gap-inner)", className)}
    >
      {children}
    </div>
  );
}

/** Right-aligned group inside a Toolbar (export, create, view switch). */
export function ToolbarActions({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("ml-auto flex flex-wrap items-center gap-2", className)}>{children}</div>;
}

const GRID_COLS = {
  1: "grid-cols-1",
  2: "grid-cols-1 md:grid-cols-2",
  3: "grid-cols-1 md:grid-cols-2 xl:grid-cols-3",
  4: "grid-cols-1 sm:grid-cols-2 xl:grid-cols-4",
} as const;

/** Grid of cards with the page gap. `cols` is the widest layout; it steps down on smaller screens. */
export function CardGrid({
  children,
  cols = 2,
  className,
}: {
  children: ReactNode;
  cols?: keyof typeof GRID_COLS;
  className?: string;
}) {
  return <div className={cn("grid min-w-0 gap-(--gap)", GRID_COLS[cols], className)}>{children}</div>;
}

/** Form fields: label above input, one field gap, two columns from sm when `cols={2}`. */
export function FormGrid({
  children,
  cols = 1,
  className,
}: {
  children: ReactNode;
  cols?: 1 | 2;
  className?: string;
}) {
  return (
    <div className={cn("grid items-start gap-4", cols === 2 && "sm:grid-cols-2", className)}>{children}</div>
  );
}

/** Form footer: right-aligned, secondary button first, primary last. Stacks primary-on-top on phones. */
export function FormActions({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:items-center sm:justify-end", className)}>
      {children}
    </div>
  );
}

/** Detail page: main column plus a fixed-width aside from lg up (aside drops below on smaller screens). */
export function DetailLayout({
  children,
  aside,
  className,
}: {
  children: ReactNode;
  aside?: ReactNode;
  className?: string;
}) {
  if (!aside) return <div className={cn("page-stack min-w-0", className)}>{children}</div>;
  return (
    <div className={cn("grid min-w-0 items-start gap-(--gap) lg:grid-cols-[minmax(0,1fr)_340px]", className)}>
      <div className="page-stack min-w-0">{children}</div>
      <aside className="page-stack min-w-0">{aside}</aside>
    </div>
  );
}

/** Key-value list with the labels in one column, so values line up. */
export function KeyValueList({
  items,
  className,
}: {
  items: { label: ReactNode; value: ReactNode; key?: string }[];
  className?: string;
}) {
  return (
    <dl className={cn("grid grid-cols-[minmax(7rem,36%)_minmax(0,1fr)] gap-x-4 gap-y-2.5 text-sm", className)}>
      {items.map((it, i) => (
        <div key={it.key ?? i} className="contents">
          <dt className="text-muted-foreground">{it.label}</dt>
          <dd className="min-w-0 break-words">{it.value}</dd>
        </div>
      ))}
    </dl>
  );
}
