"use client";

import { Globe, Heart, MessageCircle, Send } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Creative } from "@/features/campaigns/lib";
import type { BuilderBrand } from "@/features/campaigns/queries";
import { c, ctaLabel } from "@/features/campaigns/strings";
import { Block } from "./block";

/* eslint-disable @next/next/no-img-element */

/**
 * Feed ad mock-up. Uses the brand kit colours for the avatar and the image
 * fallback so a draft without an image still reads as the client's ad.
 * Brand colours here are identity, not data, so they are applied inline.
 */
export function AdPreview({ brand, creatives, active, onActive }: { brand: BuilderBrand; creatives: Creative[]; active: number; onActive: (i: number) => void }) {
  const cr = creatives[active] ?? creatives[0];
  const headline = cr?.headline.trim() || c.previewHeadline;
  const text = cr?.primaryText.trim() || c.previewText;
  return (
    <Block
      title={c.sectionPreview}
      actions={
        <div role="tablist" className="inline-flex rounded-full bg-muted p-1">
          {creatives.map((_, i) => (
            <button
              key={i}
              type="button"
              role="tab"
              aria-selected={active === i}
              aria-label={c.creativeLabel(i + 1)}
              onClick={() => onActive(i)}
              className={cn("size-7 rounded-full text-xs font-semibold text-muted-foreground", active === i && "bg-card text-foreground shadow-(--card-shadow)")}
            >
              {i + 1}
            </button>
          ))}
        </div>
      }
    >
      <article className="overflow-hidden rounded-2xl border bg-background text-foreground">
        <header className="flex items-center gap-2.5 px-3 py-2.5">
          {brand.logoUrl ? (
            <img src={brand.logoUrl} alt="" className="size-8 rounded-full object-cover" />
          ) : (
            <span className="inline-flex size-8 items-center justify-center rounded-full text-sm font-bold" style={{ background: brand.primaryColor, color: brand.secondaryColor }}>
              {brand.clientName.slice(0, 1).toUpperCase()}
            </span>
          )}
          <div className="min-w-0 leading-tight">
            <div className="truncate text-sm font-semibold">{brand.clientName}</div>
            <div className="flex items-center gap-1 text-xs text-muted-foreground">
              {c.sponsored} · <Globe className="size-3" />
            </div>
          </div>
        </header>
        <p className="line-clamp-3 px-3 pb-2.5 text-sm whitespace-pre-line">{text}</p>
        <div className="relative aspect-square w-full overflow-hidden" style={{ background: `linear-gradient(140deg, ${brand.primaryColor}, color-mix(in oklab, ${brand.primaryColor} 55%, ${brand.secondaryColor}))` }}>
          {cr?.mediaUrl ? (
            <img src={cr.mediaUrl} alt="" className="size-full object-cover" />
          ) : (
            <div className="flex size-full items-center justify-center p-8 text-center text-2xl leading-tight font-bold" style={{ color: brand.secondaryColor }}>
              {headline}
            </div>
          )}
        </div>
        <div className="flex items-center justify-between gap-3 bg-muted/60 px-3 py-2.5">
          <div className="min-w-0">
            <div className="truncate text-sm font-semibold">{headline}</div>
          </div>
          <span className="shrink-0 rounded-md bg-card px-3 py-1.5 text-xs font-semibold shadow-(--card-shadow)">{cr?.cta ? ctaLabel(cr.cta) : c.ctaPlaceholder}</span>
        </div>
        <footer className="flex items-center gap-4 px-3 py-2.5 text-muted-foreground">
          <Heart className="size-5" />
          <MessageCircle className="size-5" />
          <Send className="size-5" />
        </footer>
      </article>
    </Block>
  );
}
