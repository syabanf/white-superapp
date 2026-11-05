"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import { fs } from "../strings";

const COLUMNS = [
  { key: "doc", label: fs.reports.docNo },
  { key: "desc", label: fs.description },
];

/** Shows or hides ledger columns through the URL, so a chosen view can be shared. */
export function SubsidiaryColumnToggles({ hidden }: { hidden: string[] }) {
  const pathname = usePathname();
  const params = useSearchParams();
  return (
    <div className="flex items-center gap-1.5" aria-label={fs.reports.columns}>
      <span className="text-xs text-muted-foreground">{fs.reports.columns}:</span>
      {COLUMNS.map((c) => {
        const off = hidden.includes(c.key);
        const next = new URLSearchParams(params);
        const value = (off ? hidden.filter((h) => h !== c.key) : [...hidden, c.key]).join(",");
        if (value) next.set("hide", value);
        else next.delete("hide");
        return (
          <Link
            key={c.key}
            href={`${pathname}?${next}`}
            scroll={false}
            aria-pressed={!off}
            className={cn("rounded-full border px-2.5 py-1 text-xs", off ? "text-muted-foreground line-through" : "bg-muted")}
          >
            {c.label}
          </Link>
        );
      })}
    </div>
  );
}
