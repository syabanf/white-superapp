"use client";

import * as React from "react";
import { Check, ChevronsUpDown, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { ss } from "./strings";

export type SearchOption = { value: string; label: string; hint?: string; disabled?: boolean };

/** Combobox for long option lists (accounts, employees). Value "" means none. */
export function SearchableSelect({
  options,
  value,
  onChange,
  id,
  placeholder = "Pilih…",
  searchPlaceholder = ss.search,
  emptyText = ss.noResults,
  disabled,
  invalid,
  clearable,
  className,
}: {
  options: SearchOption[];
  value: string;
  onChange: (value: string) => void;
  id?: string;
  placeholder?: string;
  searchPlaceholder?: string;
  emptyText?: string;
  disabled?: boolean;
  invalid?: boolean;
  clearable?: boolean;
  className?: string;
}) {
  const [open, setOpen] = React.useState(false);
  const selected = options.find((o) => o.value === value);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          className={cn("w-full justify-between font-normal shadow-none", !selected && "text-muted-foreground", className)}
        >
          <span className="min-w-0 truncate text-left">
            {selected ? (
              <>
                {selected.hint ? <span className="mr-1.5 font-mono text-xs text-muted-foreground">{selected.hint}</span> : null}
                {selected.label}
              </>
            ) : (
              placeholder
            )}
          </span>
          {clearable && selected ? (
            <X
              className="size-3.5 shrink-0 opacity-60"
              role="button"
              aria-label={ss.reset}
              onClick={(e) => {
                e.stopPropagation();
                onChange("");
              }}
            />
          ) : (
            <ChevronsUpDown className="size-3.5 shrink-0 opacity-50" />
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-(--radix-popover-trigger-width) min-w-72 p-0">
        <Command
          filter={(itemValue, search) => (itemValue.toLowerCase().includes(search.toLowerCase()) ? 1 : 0)}
        >
          <CommandInput placeholder={searchPlaceholder} />
          <CommandList>
            <CommandEmpty>{emptyText}</CommandEmpty>
            {options.map((o) => (
              <CommandItem
                key={o.value}
                value={`${o.hint ?? ""} ${o.label} ${o.value}`}
                disabled={o.disabled}
                onSelect={() => {
                  onChange(o.value);
                  setOpen(false);
                }}
              >
                <Check className={cn("size-4", o.value === value ? "opacity-100" : "opacity-0")} />
                {o.hint ? <span className="font-mono text-xs text-muted-foreground">{o.hint}</span> : null}
                <span className="truncate">{o.label}</span>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
