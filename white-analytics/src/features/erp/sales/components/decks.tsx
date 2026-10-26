"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Pencil, Plus, Presentation, Sparkles, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { formatDateTime } from "@/lib/format";
import { deleteCaseStudyAction, generatePitchDeckAction, saveCaseStudyAction } from "../actions";
import { fileSize } from "../lib";
import { DECK_TONE, SERVICE_LINE, lbl, opts, ss } from "../strings";
import type { CaseStudy, PitchDeck } from "../types";
import { ChipToggle, ConfirmAction, Fld, FormDialog, FormFooter, type Option, Picker, SimpleSelect } from "./ui";

const s = ss.decks;

export function DeckDownload({ deck, size = "sm" }: { deck: PitchDeck; size?: "sm" | "xs" }) {
  return (
    <DownloadButton href={`/api/erp/api/sales/pitch-decks/${deck.id}/file`} fileName={`${deck.title}.pptx`} size={size}>
      {s.download}
    </DownloadButton>
  );
}

/** Deck cards, newest first; versions of the same deal stack under one title. */
export function DeckList({ decks, dealNames }: { decks: PitchDeck[]; dealNames: Record<string, string> }) {
  if (!decks.length) return <p className="rounded-xl border border-dashed py-10 text-center text-sm text-muted-foreground">{s.empty}</p>;
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {decks.map((d) => (
        <Card key={d.id} className="gap-3 px-5 py-4">
          <div className="flex items-start gap-3">
            <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-brand/10 text-brand">
              <Presentation className="size-4" />
            </span>
            <div className="min-w-0 flex-1">
              <Link href={`/erp/sales/pitch-decks/${d.id}`} className="line-clamp-2 font-medium hover:underline">
                {d.title}
              </Link>
              <p className="text-xs text-muted-foreground">
                {s.version(d.version)} · {s.slides(d.slides.length)} · {fileSize(d.sizeBytes)}
              </p>
            </div>
          </div>
          <p className="truncate text-xs text-muted-foreground">
            {dealNames[d.dealId] ?? ss.dash} · {formatDateTime(d.createdAt)}
          </p>
          <div className="flex justify-end">
            <DeckDownload deck={d} size="xs" />
          </div>
        </Card>
      ))}
    </div>
  );
}

export function GeneratePitchDeckDialog({
  deals,
  dealId,
  label = s.generate,
  variant = "default",
}: {
  deals?: Option[];
  dealId?: string;
  label?: string;
  variant?: "default" | "outline";
}) {
  return (
    <FormDialog
      title={s.generateTitle}
      trigger={
        <Button size="sm" variant={variant}>
          <Sparkles className="size-3.5" /> {label}
        </Button>
      }
    >
      {(close) => <GenerateForm deals={deals ?? []} dealId={dealId} close={close} />}
    </FormDialog>
  );
}

function GenerateForm({ deals, dealId, close }: { deals: Option[]; dealId?: string; close: () => void }) {
  const router = useRouter();
  const [deal, setDeal] = React.useState(dealId ?? "");
  const [tone, setTone] = React.useState("profesional");
  const [includePricing, setIncludePricing] = React.useState(true);
  const [includeCaseStudies, setIncludeCaseStudies] = React.useState(true);
  const [extraNotes, setExtraNotes] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!deal) return;
        void run(() => generatePitchDeckAction({ dealId: deal, options: { tone, includePricing, includeCaseStudies, extraNotes } }), {
          success: s.generated,
          onOk: (d) => {
            close();
            router.push(`/erp/sales/pitch-decks/${d.id}`);
          },
        });
      }}
    >
      {!dealId ? (
        <Fld label={`${s.deal} *`} htmlFor="pd-deal">
          <Picker id="pd-deal" value={deal} onChange={setDeal} options={deals} />
        </Fld>
      ) : null}
      <Fld label={s.tone} htmlFor="pd-tone">
        <SimpleSelect id="pd-tone" value={tone} onChange={setTone} options={opts(DECK_TONE)} />
      </Fld>
      <div className="flex flex-wrap gap-4 text-sm">
        <label className="flex items-center gap-2">
          <Checkbox checked={includePricing} onCheckedChange={(c) => setIncludePricing(c === true)} /> {s.includePricing}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={includeCaseStudies} onCheckedChange={(c) => setIncludeCaseStudies(c === true)} /> {s.includeCases}
        </label>
      </div>
      <Fld label={s.extraNotes} htmlFor="pd-notes">
        <Textarea id="pd-notes" rows={3} value={extraNotes} onChange={(e) => setExtraNotes(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={pending ? s.generating : s.generate} disabled={!deal} />
    </form>
  );
}

/** Slide outline preview of a generated deck. */
export function SlidePreview({ deck }: { deck: PitchDeck }) {
  return (
    <ol className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {deck.slides.map((sl, i) => (
        <li key={i} className="flex aspect-video flex-col overflow-hidden rounded-xl border bg-card p-4">
          <span className="label-mono text-[10px] text-muted-foreground">
            {String(i + 1).padStart(2, "0")} · {sl.kind}
          </span>
          <h3 className="mt-1 line-clamp-2 text-sm font-semibold">{sl.title}</h3>
          {sl.subtitle ? <p className="line-clamp-1 text-xs text-muted-foreground">{sl.subtitle}</p> : null}
          <ul className="mt-2 list-disc space-y-0.5 overflow-hidden pl-4 text-[11px] leading-snug text-muted-foreground">
            {sl.bullets.slice(0, 5).map((b, j) => (
              <li key={j} className="line-clamp-1">
                {b}
              </li>
            ))}
          </ul>
        </li>
      ))}
    </ol>
  );
}

const EMPTY_CASE: CaseStudy = {
  id: "",
  title: "",
  clientName: "",
  industry: "",
  serviceLines: [],
  challenge: "",
  solution: "",
  results: "",
  sortOrder: 0,
  isActive: true,
};

/** Case studies the generator can cite. Editable for ADMIN. */
export function CaseStudies({ items, admin }: { items: CaseStudy[]; admin: boolean }) {
  return (
    <div className="flex flex-col gap-3">
      {admin ? (
        <div>
          <CaseDialog
            item={null}
            trigger={
              <Button size="sm">
                <Plus className="size-3.5" /> {s.caseAdd}
              </Button>
            }
          />
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">{ss.adminOnly}</p>
      )}
      <div className="grid gap-3 lg:grid-cols-2">
        {items.map((c) => (
          <Card key={c.id} className="gap-2 px-5 py-4">
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <h3 className="font-medium">{c.title}</h3>
                <p className="text-xs text-muted-foreground">
                  {c.clientName} · {c.industry}
                  {!c.isActive ? ` · ${ss.inactive}` : ""}
                </p>
              </div>
              {admin ? (
                <div className="flex shrink-0">
                  <CaseDialog
                    item={c}
                    trigger={
                      <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                        <Pencil className="size-3.5" />
                      </Button>
                    }
                  />
                  <ConfirmAction
                    title={s.deleteCase}
                    action={() => deleteCaseStudyAction(c.id)}
                    trigger={
                      <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                        <Trash2 className="size-3.5" />
                      </Button>
                    }
                  />
                </div>
              ) : null}
            </div>
            <div className="flex flex-wrap gap-1">
              {c.serviceLines.map((l) => (
                <span key={l} className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
                  {lbl(SERVICE_LINE, l)}
                </span>
              ))}
            </div>
            <p className="text-sm">{c.results}</p>
          </Card>
        ))}
      </div>
    </div>
  );
}

function CaseDialog({ item, trigger }: { item: CaseStudy | null; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={s.caseTitle} size="lg">
      {(close) => <CaseForm item={item ?? EMPTY_CASE} close={close} />}
    </FormDialog>
  );
}

function CaseForm({ item, close }: { item: CaseStudy; close: () => void }) {
  const [v, setV] = React.useState(item);
  const { pending, run } = useErpAction();
  const set = <K extends keyof CaseStudy>(k: K, val: CaseStudy[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveCaseStudyAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${s.caseName} *`} htmlFor="cs-title" className="sm:col-span-2">
        <Input id="cs-title" value={v.title} onChange={(e) => set("title", e.target.value)} />
      </Fld>
      <Fld label={`${s.client} *`} htmlFor="cs-client">
        <Input id="cs-client" value={v.clientName} onChange={(e) => set("clientName", e.target.value)} />
      </Fld>
      <Fld label={s.industry} htmlFor="cs-industry">
        <Input id="cs-industry" value={v.industry} onChange={(e) => set("industry", e.target.value)} />
      </Fld>
      <Fld label={ss.pipeline.serviceLines} className="sm:col-span-2">
        <ChipToggle value={v.serviceLines} onChange={(x) => set("serviceLines", x)} options={opts(SERVICE_LINE)} />
      </Fld>
      {(["challenge", "solution", "results"] as const).map((k) => (
        <Fld key={k} label={s[k]} htmlFor={`cs-${k}`} className="sm:col-span-2">
          <Textarea id={`cs-${k}`} rows={2} value={v[k]} onChange={(e) => set(k, e.target.value)} />
        </Fld>
      ))}
      <Fld label={ss.automation.order} htmlFor="cs-order">
        <Input id="cs-order" type="number" value={v.sortOrder} onChange={(e) => set("sortOrder", Number(e.target.value))} />
      </Fld>
      <label className="flex items-center gap-2 self-end text-sm">
        <Checkbox checked={v.isActive} onCheckedChange={(x) => set("isActive", x === true)} /> {ss.active}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
