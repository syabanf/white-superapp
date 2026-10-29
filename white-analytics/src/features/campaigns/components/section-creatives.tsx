"use client";

import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { CTA_OPTIONS, HEADLINE_MAX, HEADLINE_RECOMMENDED, PRIMARY_TEXT_MAX, PRIMARY_TEXT_RECOMMENDED, type Creative } from "@/features/campaigns/lib";
import type { BuilderImage } from "@/features/campaigns/queries";
import { c } from "@/features/campaigns/strings";
import type { BuilderState, SetField } from "./builder-state";
import { Block, FieldIssues } from "./block";
import { ImagePicker } from "./image-picker";

/* eslint-disable @next/next/no-img-element */

function Counter({ n, rec }: { n: number; rec: number }) {
  return <span className={cn("tabular text-xs", n > rec ? "text-[color:var(--status-warning)]" : "text-muted-foreground")}>{c.counter(n, rec)}</span>;
}

export function SectionCreatives({
  state,
  set,
  images,
  disabled,
  issues,
  active,
  onActive,
}: {
  state: BuilderState;
  set: SetField;
  images: BuilderImage[];
  disabled: boolean;
  issues: (field: string) => string[];
  active: number;
  onActive: (i: number) => void;
}) {
  const patch = (i: number, next: Partial<Creative>) => set("creatives", state.creatives.map((cr, j) => (j === i ? { ...cr, ...next } : cr)));
  return (
    <Block id="step-creatives" index="04" title={c.sectionCreatives}>
      <div className="space-y-4">
        {state.creatives.map((cr, i) => (
          <div
            key={i}
            id={`creative-${i}`}
            onFocusCapture={() => onActive(i)}
            className={cn("rounded-2xl bg-muted/50 p-4 transition-shadow", active === i && "ring-2 ring-brand/40")}
          >
            <div className="mb-3 flex items-center justify-between gap-2">
              <button type="button" onClick={() => onActive(i)} className="label-mono text-left font-semibold text-foreground">
                {c.creativeLabel(i + 1)}
              </button>
            </div>
            <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_112px]">
              <div className="min-w-0 space-y-3">
                <div className="space-y-1">
                  <div className="flex items-center justify-between gap-2">
                    <Label htmlFor={`cr-${i}-headline`}>{c.headlineLabel}</Label>
                    <Counter n={cr.headline.length} rec={HEADLINE_RECOMMENDED} />
                  </div>
                  <Input id={`cr-${i}-headline`} value={cr.headline} onChange={(e) => patch(i, { headline: e.target.value })} maxLength={HEADLINE_MAX} disabled={disabled} />
                </div>
                <div className="space-y-1">
                  <div className="flex items-center justify-between gap-2">
                    <Label htmlFor={`cr-${i}-text`}>{c.primaryTextLabel}</Label>
                    <Counter n={cr.primaryText.length} rec={PRIMARY_TEXT_RECOMMENDED} />
                  </div>
                  <Textarea id={`cr-${i}-text`} value={cr.primaryText} onChange={(e) => patch(i, { primaryText: e.target.value })} maxLength={PRIMARY_TEXT_MAX} rows={3} disabled={disabled} />
                </div>
                <div className="space-y-1">
                  <Label htmlFor={`cr-${i}-cta`}>{c.ctaLabel}</Label>
                  <Select value={cr.cta || undefined} onValueChange={(v) => patch(i, { cta: v })} disabled={disabled}>
                    <SelectTrigger id={`cr-${i}-cta`} className="w-full sm:w-64">
                      <SelectValue placeholder={c.ctaPlaceholder} />
                    </SelectTrigger>
                    <SelectContent>
                      {CTA_OPTIONS.map((o) => (
                        <SelectItem key={o} value={o}>
                          {c.ctas[o]}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-1.5">
                <span className="text-sm font-medium">{c.imageLabel}</span>
                {cr.mediaUrl ? (
                  <div className="relative aspect-square w-28 overflow-hidden rounded-xl bg-muted">
                    <img src={cr.mediaUrl} alt="" loading="lazy" className="size-full object-cover" />
                    {!disabled ? (
                      <Button type="button" size="icon-xs" variant="secondary" className="absolute top-1 right-1" onClick={() => patch(i, { mediaUrl: null })} aria-label={c.imageRemove}>
                        <X />
                      </Button>
                    ) : null}
                  </div>
                ) : null}
                {!disabled ? <ImagePicker images={images} onPick={(url) => patch(i, { mediaUrl: url })} label={cr.mediaUrl ? c.imageChange : c.imagePick} /> : null}
              </div>
            </div>
            <FieldIssues messages={issues(`creative-${i}`)} />
          </div>
        ))}
        <FieldIssues messages={issues("creatives")} />
      </div>
    </Block>
  );
}
