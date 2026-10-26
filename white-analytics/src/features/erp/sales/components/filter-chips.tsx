"use client";

import { X } from "lucide-react";
import { useUrlFilters } from "@/features/erp/shared/url-state";

export type UrlChip = { label: string; value: string; params: string[] };

/**
 * Filters a dashboard drill put in the URL that have no control on the
 * page (industry, stage, lost reason...), as removable chips.
 */
export function UrlFilterChips({ chips }: { chips: UrlChip[] }) {
  const { set } = useUrlFilters(chips.flatMap((c) => c.params));
  if (chips.length === 0) return null;
  return (
    <div className="flex flex-wrap items-center gap-2" aria-label="Filter aktif">
      {chips.map((c) => (
        <button
          key={c.label}
          type="button"
          onClick={() => set(Object.fromEntries(c.params.map((p) => [p, null])))}
          className="inline-flex max-w-full items-center gap-1.5 rounded-full border bg-card px-3 py-1 text-xs outline-none hover:bg-muted focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={`Hapus filter ${c.label}: ${c.value}`}
        >
          <span className="text-muted-foreground">{c.label}:</span>
          <span className="truncate font-medium">{c.value}</span>
          <X className="size-3 shrink-0" />
        </button>
      ))}
    </div>
  );
}
