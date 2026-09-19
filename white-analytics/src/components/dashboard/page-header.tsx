import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";

/** Page header: quiet eyebrow, large headline, one short supporting line (clamped), actions on the right. */
export function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  eyebrow?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-4 md:flex-row md:items-end md:justify-between", className)}>
      <div className="min-w-0">
        {eyebrow ? <div className="label-mono mb-1.5 text-muted-foreground">{eyebrow}</div> : null}
        <h1 className="truncate text-[28px] leading-[1.1] font-semibold md:text-[34px]">{title}</h1>
        {description ? (
          <p className="mt-1.5 line-clamp-1 max-w-2xl text-sm text-muted-foreground">{description}</p>
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
    <section className={cn("space-y-4", className)}>
      {title || actions ? (
        <div className="flex items-end justify-between gap-3">
          <div className="min-w-0">
            {title ? (
              <h2 className="flex items-baseline gap-2 text-base font-semibold tracking-[-0.02em]">
                {index ? <span className="label-mono text-muted-foreground">{index}</span> : null}
                {title}
                {description ? <InfoHint className="self-center">{description}</InfoHint> : null}
              </h2>
            ) : null}
          </div>
          {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
