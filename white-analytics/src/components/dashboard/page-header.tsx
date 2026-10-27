import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";
import { InfoHint } from "@/components/dashboard/info-hint";

/**
 * Page header: optional back link, quiet eyebrow, large headline, one short supporting line
 * (clamped), actions on the right from lg up (they wrap below the title on smaller screens).
 * Everything starts on the page container's left edge. `banner` sets it on the gradient banner.
 */
export function PageHeader({
  title,
  description,
  actions,
  eyebrow,
  back,
  className,
  banner = false,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  eyebrow?: ReactNode;
  /** "Back to list" link shown above the eyebrow, flush with the title */
  back?: { href: string; label: ReactNode };
  className?: string;
  banner?: boolean;
}) {
  const quiet = banner ? "text-white/85" : "text-muted-foreground";
  return (
    <div
      className={cn(
        "flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between",
        banner && "surface-banner rounded-[1.75rem] px-6 py-7 md:px-8 md:py-9",
        className,
      )}
    >
      <div className="min-w-0">
        {back ? (
          <Link
            href={back.href}
            data-print-hide
            className={cn(
              "group/nudge mb-2 inline-flex items-center gap-1.5 rounded-sm text-sm font-medium outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring",
              quiet,
            )}
          >
            <ArrowLeft className="nudge nudge-back size-4" /> {back.label}
          </Link>
        ) : null}
        {eyebrow ? <div className={cn("label-mono mb-1.5", quiet)}>{eyebrow}</div> : null}
        <h1
          className="truncate text-[28px] leading-[1.15] font-bold md:text-[36px]"
          title={typeof title === "string" ? title : undefined}
        >
          {title}
        </h1>
        {description ? (
          <p className={cn("mt-1.5 line-clamp-2 max-w-2xl text-sm sm:line-clamp-1", quiet)}>
            {description}
          </p>
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
    <section className={cn("flex min-w-0 flex-col gap-(--gap-inner)", className)}>
      {title || actions ? (
        <div className="flex min-h-8 flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            {title ? (
              <h2 className="flex items-baseline gap-2 text-lg font-bold tracking-[-0.015em]">
                {index ? <span className="label-mono text-muted-foreground">{index}</span> : null}
                {title}
                {description ? <InfoHint className="self-center">{description}</InfoHint> : null}
              </h2>
            ) : null}
          </div>
          {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
      {children}
    </section>
  );
}
