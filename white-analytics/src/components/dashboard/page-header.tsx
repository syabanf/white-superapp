import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * Page header in the WHITE voice: mono eyebrow, tight bold headline with the
 * signature blue underline, supporting line, actions on the right.
 */
export function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  className,
  rule = true,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  eyebrow?: ReactNode;
  className?: string;
  /** the blue underline under the headline */
  rule?: boolean;
}) {
  return (
    <div className={cn("flex flex-col gap-4 md:flex-row md:items-end md:justify-between", className)}>
      <div className="min-w-0">
        {eyebrow ? <div className="label-mono mb-2 text-brand">{eyebrow}</div> : null}
        <h1 className="truncate text-[26px] leading-[1.05] font-bold tracking-[-0.035em] md:text-[30px]">
          {title}
        </h1>
        {rule ? <span aria-hidden className="mt-2.5 block h-0.5 w-14 rounded-full bg-brand" /> : null}
        {description ? (
          <p className="mt-3 max-w-2xl text-sm leading-relaxed text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function Section({
  title,
  description,
  actions,
  children,
  className,
  index,
}: {
  title?: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  /** technical index like "01" shown before the title */
  index?: string;
}) {
  return (
    <section className={cn("space-y-3", className)}>
      {title || actions ? (
        <div className="flex items-end justify-between gap-3">
          <div className="min-w-0">
            {title ? (
              <h2 className="flex items-baseline gap-2 text-base font-semibold tracking-[-0.02em]">
                {index ? <span className="label-mono text-brand">{index}</span> : null}
                {title}
              </h2>
            ) : null}
            {description ? <p className="mt-0.5 text-sm text-muted-foreground">{description}</p> : null}
          </div>
          {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
