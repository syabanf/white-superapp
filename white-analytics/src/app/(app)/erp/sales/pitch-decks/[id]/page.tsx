import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ChevronLeft } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { ErpError } from "@/features/erp/shared/states";
import { Panel } from "@/features/erp/sales/components/blocks";
import { DeckDownload, GeneratePitchDeckDialog, SlidePreview } from "@/features/erp/sales/components/decks";
import { Info } from "@/features/erp/sales/components/ui";
import { fileSize, salesCaps } from "@/features/erp/sales/lib";
import { getPitchDeck, listPitchDecks } from "@/features/erp/sales/queries";
import { DECK_TONE, lbl, ss } from "@/features/erp/sales/strings";
import { formatDateTime } from "@/lib/format";
import { cn } from "@/lib/utils";

const s = ss.decks;
export const metadata: Metadata = { title: s.title };

export default async function PitchDeckPage(props: PageProps<"/erp/sales/pitch-decks/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const res = await load(() => getPitchDeck(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (!res.data) return <ErpError message={res.error?.message} />;
  const deck = res.data;
  const siblings = deck.dealId ? await load(() => listPitchDecks({ dealId: deck.dealId })) : null;
  const versions = (siblings?.data ?? []).sort((a, b) => b.version - a.version);
  const o = deck.options;

  return (
    <>
      <Link href="/erp/sales/pitch-decks" className="-mb-3 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground">
        <ChevronLeft className="size-4" /> {s.title}
      </Link>
      <PageHeader
        eyebrow={`${s.title} · ${s.version(deck.version)}`}
        title={deck.title}
        description={`${s.slides(deck.slides.length)} · ${fileSize(deck.sizeBytes)} · ${formatDateTime(deck.createdAt)}`}
        actions={
          <>
            {caps.write && deck.dealId ? <GeneratePitchDeckDialog dealId={deck.dealId} label={s.regenerate} variant="outline" /> : null}
            <DeckDownload deck={deck} />
          </>
        }
      />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-4">
        <div className="lg:col-span-3">
          <SlidePreview deck={deck} />
        </div>
        <div className="flex flex-col gap-4">
          <Panel title={ss.detail}>
            <dl className="grid gap-3">
              <Info label={s.tone}>{lbl(DECK_TONE, o?.tone)}</Info>
              <Info label={s.includePricing}>{o?.includePricing ? ss.yes : ss.no}</Info>
              <Info label={s.includeCases}>{o?.includeCaseStudies ? ss.yes : ss.no}</Info>
              {o?.extraNotes ? <Info label={s.extraNotes}>{o.extraNotes}</Info> : null}
              <Info label={s.model}>{deck.model}</Info>
              {deck.dealId ? (
                <Info label={s.deal}>
                  <Link href={`/erp/sales/pipeline/${deck.dealId}`} className="hover:underline">
                    {ss.detail}
                  </Link>
                </Info>
              ) : null}
            </dl>
          </Panel>
          {versions.length > 1 ? (
            <Panel title={s.versions} flush>
              <ul className="divide-y text-sm">
                {versions.map((v) => (
                  <li key={v.id}>
                    <Link
                      href={`/erp/sales/pitch-decks/${v.id}`}
                      className={cn("flex justify-between gap-2 px-5 py-2.5 hover:bg-accent/50", v.id === deck.id && "font-medium")}
                    >
                      <span>{s.version(v.version)}</span>
                      <span className="text-xs text-muted-foreground">{formatDateTime(v.createdAt)}</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </Panel>
          ) : null}
        </div>
      </div>
    </>
  );
}
