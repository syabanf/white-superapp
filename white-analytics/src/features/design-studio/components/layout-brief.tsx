"use client";

import { Copy } from "lucide-react";
import { toast } from "@/lib/toast";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Button } from "@/components/ui/button";
import { COMPOSITION_LABEL, designerBrief, type DraftRow } from "@/features/design-studio/lib";
import { ds } from "@/features/design-studio/strings";

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid gap-1 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-3">
      <dt className="label-mono text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-sm break-words">{children}</dd>
    </div>
  );
}

/** Layout spec as a "brief untuk desainer" card with copy-to-clipboard. */
export function LayoutBrief({ draft }: { draft: DraftRow }) {
  const l = draft.layout;
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(designerBrief(draft));
      toast.success(ds.copied);
    } catch {
      toast.error(ds.copyFailed);
    }
  };

  return (
    <section className="flex h-full flex-col gap-4 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
      <header className="flex items-center justify-between gap-3">
        <h3 className="flex items-center gap-1.5 text-base font-semibold tracking-[-0.01em]">
          {ds.briefCard} <InfoHint>{ds.briefCardHint}</InfoHint>
        </h3>
        <Button type="button" variant="secondary" size="sm" onClick={copy}>
          <Copy /> {ds.copy}
        </Button>
      </header>
      <dl className="grid gap-3">
        <Row label={ds.headline}>
          <span className="text-base font-semibold">{l.headline || "–"}</span>
        </Row>
        <Row label={ds.subline}>{l.subline || "–"}</Row>
        <Row label={ds.cta}>
          {l.cta ? <span className="inline-flex rounded-full bg-foreground px-3 py-1 text-xs font-medium text-background">{l.cta}</span> : "–"}
        </Row>
        <Row label={ds.composition}>{COMPOSITION_LABEL[l.composition]}</Row>
        <Row label={ds.palette}>
          <span className="flex flex-wrap gap-2">
            {l.palette.map((c, i) => (
              <span key={`${c}-${i}`} className="inline-flex items-center gap-1.5 rounded-full bg-muted/60 py-1 pr-2.5 pl-1 text-xs tabular">
                <span className="size-5 rounded-full ring-1 ring-foreground/10" style={{ background: c }} />
                {c}
              </span>
            ))}
          </span>
        </Row>
        {l.elements.length ? (
          <Row label={ds.elements}>
            <ul className="list-disc space-y-1 pl-4">
              {l.elements.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          </Row>
        ) : null}
        {l.imagePrompt ? (
          <Row label={ds.imagePrompt}>
            <p className="rounded-xl bg-muted/50 px-3 py-2 font-mono text-xs leading-relaxed text-muted-foreground">{l.imagePrompt}</p>
          </Row>
        ) : null}
      </dl>
    </section>
  );
}
