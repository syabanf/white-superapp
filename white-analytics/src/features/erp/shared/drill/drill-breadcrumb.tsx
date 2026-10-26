import Link from "next/link";
import { ArrowLeft, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { ds } from "./strings";

export type DrillCrumb = {
  label: string;
  /** link that returns to this level; the last crumb (current level) needs none */
  href?: string;
};

/**
 * Trail of in-dashboard drill levels: "Semua > Maret 2026 > Creative" with a
 * "Kembali" link to the parent level. Renders nothing at the top level.
 */
export function DrillBreadcrumb({ items, className }: { items: DrillCrumb[]; className?: string }) {
  if (items.length < 2) return null;
  const parent = items.at(-2);
  return (
    <nav
      aria-label={ds.trail}
      className={cn("flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1.5 text-xs", className)}
    >
      {parent?.href ? (
        <Link
          href={parent.href}
          scroll={false}
          className="inline-flex h-7 shrink-0 items-center gap-1 rounded-full border bg-card px-2.5 font-medium outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
        >
          <ArrowLeft className="size-3.5" aria-hidden />
          {ds.back}
        </Link>
      ) : null}
      <ol className="flex min-w-0 flex-wrap items-center gap-1">
        {items.map((item, i) => {
          const last = i === items.length - 1;
          return (
            <li key={`${item.label}-${i}`} className="flex min-w-0 items-center gap-1">
              {i > 0 ? <ChevronRight className="size-3 shrink-0 text-muted-foreground" aria-hidden /> : null}
              {last || !item.href ? (
                <span
                  aria-current={last ? "page" : undefined}
                  className={cn(
                    "max-w-[14rem] truncate rounded-full px-2 py-1",
                    last ? "bg-brand/10 font-medium text-foreground" : "text-muted-foreground",
                  )}
                  title={item.label}
                >
                  {item.label}
                </span>
              ) : (
                <Link
                  href={item.href}
                  scroll={false}
                  className="max-w-[14rem] truncate rounded-full px-2 py-1 text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
                  title={item.label}
                >
                  {item.label}
                </Link>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
