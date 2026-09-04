"use client";

import * as React from "react";
import { useTransition } from "react";
import { parseAsString, useQueryState } from "nuqs";
import { Download, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

const PRESET_OPTIONS = [
  { key: "7d", label: rs.share.preset7d },
  { key: "28d", label: rs.share.preset28d },
  { key: "90d", label: rs.share.preset90d },
] as const;

/** Preset picker + PDF export for the public share top bar (state lives in the URL). */
export function ShareControls({ slug }: { slug: string }) {
  const [isPending, startTransition] = useTransition();
  const [preset, setPreset] = useQueryState("preset", parseAsString.withOptions({ shallow: false, history: "replace", startTransition }));
  const active = PRESET_OPTIONS.some((p) => p.key === preset) ? (preset as string) : "28d";

  return (
    <div className="flex items-center gap-2">
      <Select value={active} onValueChange={(v) => void setPreset(v)}>
        <SelectTrigger size="sm" className="w-[110px]" aria-label={rs.share.presetLabel}>
          {isPending ? <Loader2 className="size-3.5 animate-spin" /> : null}
          <SelectValue>{PRESET_OPTIONS.find((p) => p.key === active)?.label}</SelectValue>
        </SelectTrigger>
        <SelectContent align="end">
          {PRESET_OPTIONS.map((p) => (
            <SelectItem key={p.key} value={p.key}>
              {p.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Button asChild size="sm" variant="outline">
        <a href={`/api/share/${slug}/pdf?preset=${active}`}>
          <Download className="size-4" /> {t.common.exportPdf}
        </a>
      </Button>
    </div>
  );
}
