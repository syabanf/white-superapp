"use client";

import * as React from "react";
import { X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { c } from "@/features/campaigns/strings";

/** Free-text chips: Enter or comma adds, Backspace on empty removes the last one. */
export function TagInput({
  id,
  values,
  onChange,
  placeholder,
  max,
  disabled,
}: {
  id: string;
  values: string[];
  onChange: (next: string[]) => void;
  placeholder: string;
  max: number;
  disabled?: boolean;
}) {
  const [draft, setDraft] = React.useState("");
  const add = (raw: string) => {
    const v = raw.trim().replace(/,$/, "").trim();
    if (!v || values.some((x) => x.toLowerCase() === v.toLowerCase()) || values.length >= max) return;
    onChange([...values, v.slice(0, 80)]);
  };
  return (
    <div className="space-y-2">
      {values.length ? (
        <ul className="flex flex-wrap gap-1.5">
          {values.map((v) => (
            <li key={v} className="inline-flex max-w-full items-center gap-1 rounded-full bg-muted py-1 pr-1 pl-3 text-sm">
              <span className="truncate">{v}</span>
              {!disabled ? (
                <button
                  type="button"
                  onClick={() => onChange(values.filter((x) => x !== v))}
                  aria-label={c.removeTag(v)}
                  className="inline-flex size-5 items-center justify-center rounded-full text-muted-foreground hover:bg-background hover:text-foreground"
                >
                  <X className="size-3" />
                </button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
      {!disabled && values.length < max ? (
        <Input
          id={id}
          value={draft}
          placeholder={placeholder}
          onChange={(e) => {
            const v = e.target.value;
            if (v.endsWith(",")) {
              add(v);
              setDraft("");
            } else setDraft(v);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault();
              add(draft);
              setDraft("");
            } else if (e.key === "Backspace" && !draft && values.length) onChange(values.slice(0, -1));
          }}
          onBlur={() => {
            if (draft.trim()) {
              add(draft);
              setDraft("");
            }
          }}
        />
      ) : null}
    </div>
  );
}
