"use client";

import * as React from "react";
import { Check, Loader2, MessageSquare, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDateLong, formatDateTime } from "@/lib/format";
import { toast } from "@/lib/toast";
import { OPEN_OFFER } from "../lib";
import { OFFER_STATUS, cp, kinds } from "../strings";
import type { OfferPortal as Portal } from "../types";
import { errorMessage, portalPost } from "./client";

const t = cp.offer;

export function OfferPortal({ token, initial }: { token: string; initial: Portal }) {
  const base = `/v1/recruitment/public/offers/${encodeURIComponent(token)}`;
  const [offer, setOffer] = React.useState(initial);
  const [note, setNote] = React.useState("");
  const [busy, setBusy] = React.useState<string | null>(null);
  const [noteError, setNoteError] = React.useState(false);
  const open = OPEN_OFFER.includes(offer.status);

  const respond = async (action: "accept" | "negotiate" | "decline") => {
    if (action === "negotiate" && !note.trim()) {
      setNoteError(true);
      return false;
    }
    setBusy(action);
    try {
      setOffer(await portalPost<Portal>(`${base}/respond`, { action, note: note.trim() }));
      setNote("");
      toast.success(t.sent);
      return true;
    } catch (e) {
      toast.error(errorMessage(e));
      return false;
    } finally {
      setBusy(null);
    }
  };

  return (
    <>
      <div>
        <p className="label-mono text-muted-foreground">
          {t.eyebrow} · {t.version(offer.version)}
        </p>
        <h1 className="mt-1 text-[26px] leading-tight font-semibold">{t.hello(offer.candidateName)}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t.intro(offer.companyName, offer.positionTitle)}</p>
      </div>
      <Card className="flex flex-col gap-5 p-(--card-pad)">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <ErpStatusBadge status={offer.status} labels={OFFER_STATUS} kinds={kinds.offer} />
          <DownloadButton href={`/api/public-erp${base}/letter`} fileName="surat-penawaran.pdf">
            {t.download}
          </DownloadButton>
        </div>
        <p className="text-sm">{t.status[offer.status] ?? ""}</p>
        <dl className="grid gap-4 sm:grid-cols-2">
          <div>
            <dt className="text-xs text-muted-foreground">{t.position}</dt>
            <dd className="mt-0.5 font-medium">{offer.positionTitle}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">{t.baseSalary}</dt>
            <dd className="mt-0.5 text-lg font-semibold">
              <Money value={offer.baseSalary} />
            </dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">{t.startDate}</dt>
            <dd className="mt-0.5">{offer.startDate ? formatDateLong(offer.startDate) : "–"}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted-foreground">{t.expiresAt}</dt>
            <dd className="mt-0.5">{formatDateTime(offer.expiresAt)}</dd>
          </div>
        </dl>
        {offer.benefits.length ? (
          <div>
            <h2 className="mb-1.5 text-sm font-semibold">{t.benefits}</h2>
            <ul className="list-disc space-y-1 pl-5 text-sm">
              {offer.benefits.map((b) => (
                <li key={b}>{b}</li>
              ))}
            </ul>
          </div>
        ) : null}
        {offer.responseNote ? (
          <div className="rounded-lg bg-muted/50 p-3 text-sm">
            <p className="text-xs text-muted-foreground">
              {t.yourNote} · {formatDateTime(offer.respondedAt)}
            </p>
            <p className="mt-1 whitespace-pre-wrap">{offer.responseNote}</p>
          </div>
        ) : null}
      </Card>
      {open ? (
        <Card className="flex flex-col gap-4 p-(--card-pad)">
          <h2 className="font-semibold">{t.respondTitle}</h2>
          <div className="grid gap-1.5">
            <Label htmlFor="offer-note">{t.note}</Label>
            <Textarea
              id="offer-note"
              rows={3}
              value={note}
              onChange={(e) => {
                setNote(e.target.value);
                setNoteError(false);
              }}
              placeholder={t.noteNegotiate}
              aria-invalid={noteError || undefined}
            />
            {noteError ? <p className="text-xs text-destructive">{t.noteRequired}</p> : null}
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <ConfirmAction
              title={t.decline}
              description={t.confirmDecline}
              confirmLabel={t.decline}
              destructive
              onConfirm={() => respond("decline")}
              trigger={
                <Button variant="outline" disabled={busy !== null} className="w-full sm:w-auto">
                  <X className="size-4" /> {t.decline}
                </Button>
              }
            />
            <Button variant="outline" disabled={busy !== null} className="w-full sm:w-auto" onClick={() => void respond("negotiate")}>
              {busy === "negotiate" ? <Loader2 className="size-4 animate-spin" /> : <MessageSquare className="size-4" />}
              {t.negotiate}
            </Button>
            <ConfirmAction
              title={t.accept}
              description={t.confirmAccept}
              confirmLabel={t.accept}
              onConfirm={() => respond("accept")}
              trigger={
                <Button disabled={busy !== null} className="w-full sm:w-auto">
                  <Check className="size-4" /> {t.accept}
                </Button>
              }
            />
          </div>
        </Card>
      ) : null}
    </>
  );
}
