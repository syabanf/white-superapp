"use client";

import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { normalizeHex } from "@/features/brand/lib";
import { b } from "@/features/brand/strings";
import { cn } from "@/lib/utils";

/** Native color picker + hex text input, kept in sync. The text may be mid-edit (invalid) while typing. */
export function ColorField({
  id,
  label,
  value,
  onChange,
  onRemove,
  disabled,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  onRemove?: () => void;
  disabled?: boolean;
}) {
  const valid = normalizeHex(value);
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex items-center gap-2">
        <label
          className={cn(
            "relative size-(--control-h) shrink-0 overflow-hidden rounded-full shadow-(--card-shadow) ring-1 ring-foreground/10",
            disabled ? "opacity-60" : "cursor-pointer",
          )}
          style={{ background: valid ?? "transparent" }}
        >
          <span className="sr-only">{label}</span>
          <input
            type="color"
            value={valid ?? "#000000"}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
            className="absolute inset-0 size-full cursor-pointer opacity-0"
            aria-label={label}
          />
        </label>
        <Input
          id={id}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={() => valid && valid !== value && onChange(valid)}
          disabled={disabled}
          aria-invalid={!valid}
          className="tabular min-w-0 font-mono text-[13px]"
          maxLength={7}
          spellCheck={false}
        />
        {onRemove && !disabled ? (
          <Button type="button" variant="ghost" size="icon-sm" onClick={onRemove} aria-label={b.removeAccent}>
            <X />
          </Button>
        ) : null}
      </div>
      {!valid ? <p className="text-xs text-destructive">{b.invalidHex}</p> : null}
    </div>
  );
}
