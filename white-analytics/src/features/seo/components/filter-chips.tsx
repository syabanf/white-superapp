import Link from "next/link";
import { X } from "lucide-react";
import { t } from "@/i18n/id";

export type FilterChip = { label: string; value: string; removeHref: string };

/** Active drill filters as removable chips, with one link that clears them all. */
export function FilterChips({ chips, clearHref }: { chips: FilterChip[]; clearHref: string }) {
  if (chips.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2" aria-label="Filter aktif">
      {chips.map((c) => (
        <Link
          key={c.label}
          href={c.removeHref}
          scroll={false}
          className="inline-flex max-w-full items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={`Hapus filter ${c.label}: ${c.value}`}
        >
          <span className="text-muted-foreground">{c.label}:</span>
          <span className="truncate font-medium">{c.value}</span>
          <X className="size-3 shrink-0" />
        </Link>
      ))}
      {chips.length > 1 ? (
        <Link href={clearHref} scroll={false} className="text-xs text-muted-foreground underline-offset-4 hover:underline">
          {t.common.all}
        </Link>
      ) : null}
    </div>
  );
}
