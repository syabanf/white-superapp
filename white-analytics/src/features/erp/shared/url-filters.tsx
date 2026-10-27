"use client";

import * as React from "react";
import { Loader2, Search, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { SearchableSelect, type SearchOption } from "./searchable-select";
import { useUrlFilters } from "./url-state";
import { ss } from "./strings";

const ALL = "__all__";

/** Debounced search box bound to a URL param (default `q`). */
export function SearchFilter({
  param = "q",
  placeholder = ss.search,
  className,
}: {
  param?: string;
  placeholder?: string;
  className?: string;
}) {
  const { values, set, isPending } = useUrlFilters([param]);
  const [text, setText] = React.useState(values[param] ?? "");
  const timer = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  React.useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);
  return (
    <div className={cn("relative max-sm:w-full", className)}>
      {isPending ? (
        <Loader2 className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 animate-spin text-muted-foreground" />
      ) : (
        <Search className="pointer-events-none absolute top-1/2 left-3.5 size-4 -translate-y-1/2 text-muted-foreground" />
      )}
      <Input
        value={text}
        placeholder={placeholder}
        aria-label={placeholder}
        className="w-full pl-9 sm:w-64"
        onChange={(e) => {
          const v = e.target.value;
          setText(v);
          if (timer.current) clearTimeout(timer.current);
          timer.current = setTimeout(() => set({ [param]: v.trim() || null }), 350);
        }}
      />
    </div>
  );
}

export type FilterOption = { value: string; label: string };

/** Single-select bound to a URL param; the "all" option clears it. */
export function SelectFilter({
  param,
  options,
  allLabel = ss.all,
  label,
  className,
}: {
  param: string;
  options: FilterOption[];
  allLabel?: string;
  /** accessible name */
  label: string;
  className?: string;
}) {
  const { values, set } = useUrlFilters([param]);
  return (
    <Select value={values[param] ?? ALL} onValueChange={(v) => set({ [param]: v === ALL ? null : v })}>
      <SelectTrigger className={cn("min-w-36", className)} aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL}>{allLabel}</SelectItem>
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

/** Clears the given URL params; hidden when none is set. */
export function ResetFilters({ params }: { params: string[] }) {
  const { values, set } = useUrlFilters(params);
  if (!params.some((p) => values[p])) return null;
  return (
    <Button variant="ghost" onClick={() => set(Object.fromEntries(params.map((p) => [p, null])))}>
      <X /> {ss.reset}
    </Button>
  );
}

/** Searchable single-select bound to a URL param, for long lists (accounts, employees). */
export function SearchSelectFilter({
  param,
  options,
  placeholder,
  className,
}: {
  param: string;
  options: SearchOption[];
  placeholder: string;
  className?: string;
}) {
  const { values, set } = useUrlFilters([param]);
  return (
    <SearchableSelect
      options={options}
      value={values[param] ?? ""}
      onChange={(v) => set({ [param]: v || null })}
      placeholder={placeholder}
      className={cn("w-full sm:w-72", className)}
      clearable
    />
  );
}
