"use client";

import * as React from "react";
import { FilePlus2, MessageSquareReply } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate, formatDateTime } from "@/lib/format";
import { createOfferAction, recordOfferResponseAction } from "../actions";
import { OPEN_OFFER, splitLines } from "../lib";
import { OFFER_STATUS, kinds, rs } from "../strings";
import type { Offer, OfferPanel as Panel, SalaryReference } from "../types";
import { Info, PortalLink } from "./ui";

const o = rs.offers;

export function OffersPanel({ candidateId, panel, canWrite }: { candidateId: string; panel: Panel; canWrite: boolean }) {
  const ref = panel.salaryReference;
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div className="flex flex-col gap-4 lg:col-span-2">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{o.title}</h2>
          {canWrite ? <CreateOfferButton candidateId={candidateId} reference={ref} last={panel.offers[0]} /> : null}
        </div>
        {panel.offers.length === 0 ? (
          <ErpEmpty compact title={o.empty} />
        ) : (
          panel.offers.map((offer) => <OfferCard key={offer.id} offer={offer} canWrite={canWrite} />)
        )}
      </div>
      {ref ? (
        <Card className="self-start">
          <CardHeader>
            <CardTitle>{o.reference}</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="flex flex-col gap-3">
              <Info label={o.position}>{ref.positionTitle}</Info>
              <Info label={o.range}>
                {ref.salaryMin || ref.salaryMax ? (
                  <>
                    <Money value={ref.salaryMin} /> – <Money value={ref.salaryMax} />
                  </>
                ) : null}
              </Info>
              <Info label={o.expected}>{ref.expectedSalary ? <Money value={ref.expectedSalary} /> : null}</Info>
              <Info label={o.confirmed}>{ref.confirmedSalary ? <Money value={ref.confirmedSalary} /> : null}</Info>
              <Info label={o.interviewExpectation}>{ref.interviewExpectation}</Info>
            </dl>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

function OfferCard({ offer, canWrite }: { offer: Offer; canWrite: boolean }) {
  const open = OPEN_OFFER.includes(offer.status);
  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle className="flex items-center gap-2">
          {o.version(offer.version)}
          <ErpStatusBadge status={offer.status} labels={OFFER_STATUS} kinds={kinds.offer} />
        </CardTitle>
        <div className="flex flex-wrap gap-1.5">
          <DownloadButton href={`/api/erp/v1/recruitment/offers/${offer.id}/letter`} size="xs" fileName={`penawaran-v${offer.version}.pdf`}>
            {o.letter}
          </DownloadButton>
          {canWrite && open ? <RecordResponseButton offer={offer} /> : null}
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Info label={o.position}>{offer.positionTitle}</Info>
          <Info label={o.baseSalary}>
            <Money value={offer.baseSalary} />
          </Info>
          <Info label={o.startDate}>{offer.startDate ? formatDate(offer.startDate) : null}</Info>
          <Info label={o.expiresAt}>{formatDateTime(offer.expiresAt)}</Info>
        </dl>
        {offer.benefits.length ? (
          <ul className="list-disc space-y-1 pl-5 text-sm">
            {offer.benefits.map((b) => (
              <li key={b}>{b}</li>
            ))}
          </ul>
        ) : null}
        {offer.notes ? (
          <p className="text-xs whitespace-pre-wrap text-muted-foreground">
            {o.notes}: {offer.notes}
          </p>
        ) : null}
        {offer.respondedAt ? (
          <div className="rounded-lg bg-muted/40 p-3 text-sm">
            <p className="text-xs text-muted-foreground">{o.respondedVia(offer.responseSource, formatDateTime(offer.respondedAt))}</p>
            {offer.responseNote ? <p className="mt-1 whitespace-pre-wrap">{offer.responseNote}</p> : null}
          </div>
        ) : null}
        {canWrite && open && offer.token ? <PortalLink path={`/offer/${offer.token}`} /> : null}
      </CardContent>
    </Card>
  );
}

function RecordResponseButton({ offer }: { offer: Offer }) {
  return (
    <FormDialog
      title={o.recordTitle}
      description={o.recordHint}
      trigger={
        <Button size="xs" variant="outline">
          <MessageSquareReply className="size-3.5" /> {o.record}
        </Button>
      }
    >
      {(close) => <RecordForm offer={offer} close={close} />}
    </FormDialog>
  );
}

function RecordForm({ offer, close }: { offer: Offer; close: () => void }) {
  const [status, setStatus] = React.useState("accepted");
  const [note, setNote] = React.useState("");
  const { run, pending } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => recordOfferResponseAction(offer.id, status, note), { success: o.recorded, onOk: close });
      }}
    >
      <Fld label={o.responseStatus} htmlFor="or-status">
        <SimpleSelect
          id="or-status"
          value={status}
          onChange={setStatus}
          options={Object.entries(o.responseOptions).map(([value, label]) => ({ value, label }))}
        />
      </Fld>
      <Fld label={o.responseNote} htmlFor="or-note">
        <Textarea id="or-note" rows={3} value={note} onChange={(e) => setNote(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function CreateOfferButton({ candidateId, reference, last }: { candidateId: string; reference: SalaryReference | null; last?: Offer }) {
  return (
    <FormDialog
      title={o.createTitle}
      description={last ? o.createHint : undefined}
      wide
      trigger={
        <Button size="sm">
          <FilePlus2 className="size-4" /> {o.create}
        </Button>
      }
    >
      {(close) => <OfferForm candidateId={candidateId} reference={reference} last={last} close={close} />}
    </FormDialog>
  );
}

function OfferForm({
  candidateId,
  reference,
  last,
  close,
}: {
  candidateId: string;
  reference: SalaryReference | null;
  last?: Offer;
  close: () => void;
}) {
  const [v, setV] = React.useState({
    positionTitle: last?.positionTitle ?? reference?.positionTitle ?? "",
    baseSalary: last?.baseSalary ?? reference?.confirmedSalary ?? reference?.expectedSalary ?? "",
    benefits: (last?.benefits ?? []).join("\n"),
    startDate: last?.startDate ?? "",
    notes: "",
    expiresDays: 7,
  });
  const set = <K extends keyof typeof v>(k: K, val: (typeof v)[K]) => setV((p) => ({ ...p, [k]: val }));
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createOfferAction(candidateId, { ...v, benefits: splitLines(v.benefits) }), {
          success: o.created,
          onOk: close,
          onFail,
        });
      }}
    >
      <Fld label={o.position} htmlFor="of-pos" error={errors?.positionTitle}>
        <Input id="of-pos" value={v.positionTitle} onChange={(e) => set("positionTitle", e.target.value)} />
      </Fld>
      <Fld label={o.baseSalary} htmlFor="of-sal" error={errors?.baseSalary}>
        <MoneyInput id="of-sal" value={v.baseSalary} onChange={(x) => set("baseSalary", x)} />
      </Fld>
      <Fld label={o.startDate} htmlFor="of-start" error={errors?.startDate}>
        <DatePicker id="of-start" value={v.startDate} onChange={(x) => set("startDate", x)} clearable />
      </Fld>
      <Fld label={o.expiresDays} htmlFor="of-days" error={errors?.expiresDays}>
        <Input id="of-days" type="number" min={1} max={30} value={v.expiresDays} onChange={(e) => set("expiresDays", Number(e.target.value))} />
      </Fld>
      <Fld label={o.benefits} htmlFor="of-ben" error={errors?.benefits} className="sm:col-span-2">
        <Textarea id="of-ben" rows={4} value={v.benefits} onChange={(e) => set("benefits", e.target.value)} />
      </Fld>
      <Fld label={o.notes} htmlFor="of-notes" className="sm:col-span-2">
        <Textarea id="of-notes" rows={2} value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} submitLabel={o.create} />
      </div>
    </form>
  );
}
