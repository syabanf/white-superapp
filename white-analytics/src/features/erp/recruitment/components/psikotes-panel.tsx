"use client";

import * as React from "react";
import { Eye, Loader2, Send, ShieldAlert } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/hr/components/ui";
import type { ActionResult } from "@/lib/action-result";
import { useErpAction } from "@/features/erp/shared/client";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { cn } from "@/lib/utils";
import { formatDateTime } from "@/lib/format";
import {
  getTestAnswersAction,
  invitePsikotesAction,
  listProctorEventsAction,
  reviewTestAction,
  savePsikotesSummaryAction,
} from "../actions";
import { INSTRUMENT_KIND, PROCTOR_EVENT, RECOMMENDATION, SESSION_STATUS, TEST_STATUS, kinds, rs } from "../strings";
import type { Instrument, ProctorEvent, PsikotesPanel as Panel, PsikotesSession, SessionTest, TestAnswers } from "../types";
import { Info, PortalLink } from "./ui";

const t = rs.psikotes;

export function PsikotesPanel({
  candidateId,
  panel,
  instruments,
  canWrite,
}: {
  candidateId: string;
  panel: Panel;
  instruments: Instrument[];
  canWrite: boolean;
}) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <div className="flex flex-col gap-4 lg:col-span-2">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-base font-semibold">{t.title}</h2>
          {canWrite ? <InviteButton candidateId={candidateId} instruments={instruments} /> : null}
        </div>
        {panel.sessions.length === 0 ? (
          <ErpEmpty compact title={t.empty} />
        ) : (
          panel.sessions.map((s, i) => <SessionCard key={s.id} session={s} index={panel.sessions.length - i} canWrite={canWrite} />)
        )}
      </div>
      <SummaryCard candidateId={candidateId} summary={panel.summary} canWrite={canWrite} />
    </div>
  );
}

function SessionCard({ session, index, canWrite }: { session: PsikotesSession; index: number; canWrite: boolean }) {
  const flags = session.proctor?.flags ?? 0;
  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle className="flex items-center gap-2">
          {t.session(index)}
          <ErpStatusBadge status={session.status} labels={SESSION_STATUS} kinds={kinds.session} />
        </CardTitle>
        <ProctorButton sessionType="psikotes" sessionId={session.id} flags={flags} />
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Info label={t.invitedAt}>{formatDateTime(session.invitedAt)}</Info>
          <Info label={t.expiresAt}>{formatDateTime(session.expiresAt)}</Info>
          <Info label={t.completedAt}>{session.completedAt ? formatDateTime(session.completedAt) : null}</Info>
        </dl>
        {canWrite && session.token && ["draft", "sent", "in_progress"].includes(session.status) ? (
          <div>
            <p className="mb-1.5 text-xs text-muted-foreground">
              {t.link} · {rs.portalLinks.hint}
            </p>
            <PortalLink path={`/psikotes/${session.token}`} />
          </div>
        ) : null}
        <ul className="divide-y rounded-lg border">
          {session.tests.map((test) => (
            <TestRow key={test.id} test={test} canWrite={canWrite} />
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function TestRow({ test, canWrite }: { test: SessionTest; canWrite: boolean }) {
  const d = test.scoreDetail;
  const done = ["selesai", "perlu_review", "reviewed"].includes(test.status);
  return (
    <li className="flex flex-col gap-2 px-3 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2 font-medium">
          {test.instrumentName}
          <ErpStatusBadge status={test.status} labels={TEST_STATUS} kinds={kinds.test} />
        </div>
        <div className="mt-0.5 text-xs text-muted-foreground">
          {INSTRUMENT_KIND[test.instrumentKind] ?? test.instrumentKind}
          {test.instrumentKind === "mcq" && d && d.mcqTotal ? ` · ${t.mcqScore(d.mcqCorrect, d.mcqTotal)}` : null}
          {test.score != null ? ` · ${t.score} ${test.score}` : null}
        </div>
        {test.instrumentKind === "forced_choice" && d?.papiDominant.length ? (
          <div className="mt-1.5 text-xs">
            <span className="text-muted-foreground">{t.papiDominant}: </span>
            {d.papiDominant
              .slice(0, 5)
              .map((x) => `${x.code} ${x.label}`)
              .join(", ")}
          </div>
        ) : null}
        {test.reviewNotes ? (
          <p className="mt-1.5 text-xs whitespace-pre-wrap">
            <span className="text-muted-foreground">{t.reviewedBy(test.reviewedByName || rs.none)}: </span>
            {test.reviewNotes}
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 flex-wrap gap-1.5">
        {done && test.instrumentKind !== "drawing" ? <AnswersButton test={test} /> : null}
        {test.hasAttachment ? (
          <DownloadButton href={`/api/erp/v1/recruitment/psikotes/session-tests/${test.id}/attachment`} size="xs">
            {t.attachment}
          </DownloadButton>
        ) : null}
        {canWrite && test.instrumentKind === "drawing" && done ? <ReviewButton test={test} /> : null}
      </div>
    </li>
  );
}

/** Loads a server action result once when mounted. */
function useLoad<T>(fn: () => Promise<ActionResult<T>>) {
  const [state, setState] = React.useState<{ data?: T; error?: string }>({});
  React.useEffect(() => {
    let alive = true;
    void fn().then((res) => {
      if (alive) setState(res.ok ? { data: res.data } : { error: res.error });
    });
    return () => {
      alive = false;
    };
    // load once per mount; the dialog remounts its body on every open
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return state;
}

function Loading() {
  return (
    <div className="flex justify-center py-8">
      <Loader2 className="size-5 animate-spin text-muted-foreground" />
    </div>
  );
}

function AnswersButton({ test }: { test: SessionTest }) {
  return (
    <FormDialog
      title={`${t.answersTitle} · ${test.instrumentName}`}
      wide
      trigger={
        <Button size="xs" variant="outline">
          <Eye className="size-3.5" /> {t.answers}
        </Button>
      }
    >
      {() => <AnswersBody testId={test.id} />}
    </FormDialog>
  );
}

function AnswersBody({ testId }: { testId: string }) {
  const { data, error } = useLoad<TestAnswers>(() => getTestAnswersAction(testId));
  if (error) return <ErpError compact message={error} />;
  if (!data) return <Loading />;
  if (!data.items.length) return <ErpEmpty compact />;
  return (
    <ol className="flex flex-col gap-3 text-sm">
      {data.items.map((it, i) => (
        <li key={it.questionId} className="rounded-lg border p-3">
          <p className="font-medium">
            {i + 1}. {it.body || rs.none}
          </p>
          {data.kind === "mcq" ? (
            <ul className="mt-2 flex flex-col gap-1">
              {it.options.map((o) => (
                <li
                  key={o.key}
                  className={cn(
                    "rounded px-2 py-1",
                    o.key === it.correctKey && "bg-positive/10 font-medium",
                    o.key === it.given && o.key !== it.correctKey && "bg-destructive/10",
                  )}
                >
                  <span className="uppercase">{o.key}.</span> {o.text}
                </li>
              ))}
            </ul>
          ) : (
            <ul className="mt-2 flex flex-col gap-1">
              {(["a", "b"] as const).map((k) => {
                const st = k === "a" ? it.statementA : it.statementB;
                return (
                  <li key={k} className={cn("rounded px-2 py-1", it.given === k && "bg-brand/10 font-medium")}>
                    <span className="uppercase">{k}.</span> {st?.text}
                    {st?.scale ? <span className="ml-1 text-xs text-muted-foreground">({st.scale})</span> : null}
                  </li>
                );
              })}
            </ul>
          )}
          {!it.given ? <p className="mt-1 text-xs text-muted-foreground">{t.notAnswered}</p> : null}
        </li>
      ))}
    </ol>
  );
}

function ReviewButton({ test }: { test: SessionTest }) {
  return (
    <FormDialog
      title={t.reviewTitle}
      trigger={
        <Button size="xs" variant="outline">
          {t.review}
        </Button>
      }
    >
      {(close) => <ReviewForm test={test} close={close} />}
    </FormDialog>
  );
}

function ReviewForm({ test, close }: { test: SessionTest; close: () => void }) {
  const [notes, setNotes] = React.useState(test.reviewNotes);
  const { run, pending } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => reviewTestAction(test.id, notes), { success: t.reviewed, onOk: close });
      }}
    >
      {test.hasAttachment ? (
        <DownloadButton href={`/api/erp/v1/recruitment/psikotes/session-tests/${test.id}/attachment`} className="self-start">
          {t.attachment}
        </DownloadButton>
      ) : null}
      <Fld label={t.reviewNotes} htmlFor="review-notes">
        <Textarea id="review-notes" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)} autoFocus />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

/** Proctor flag count that opens the event log. */
export function ProctorButton({ sessionType, sessionId, flags }: { sessionType: "psikotes" | "interview"; sessionId: string; flags: number }) {
  return (
    <FormDialog
      title={t.proctorTitle}
      trigger={
        <Button size="xs" variant={flags ? "outline" : "ghost"} className={cn(flags > 0 && "text-amber-600 dark:text-amber-400")}>
          <ShieldAlert className="size-3.5" /> {t.flags(flags)}
        </Button>
      }
    >
      {() => <ProctorBody sessionType={sessionType} sessionId={sessionId} />}
    </FormDialog>
  );
}

function ProctorBody({ sessionType, sessionId }: { sessionType: "psikotes" | "interview"; sessionId: string }) {
  const { data, error } = useLoad<ProctorEvent[]>(() => listProctorEventsAction(sessionType, sessionId));
  if (error) return <ErpError compact message={error} />;
  if (!data) return <Loading />;
  if (!data.length) return <p className="py-6 text-center text-sm text-muted-foreground">{t.noProctor}</p>;
  return (
    <ul className="flex flex-col divide-y text-sm">
      {data.map((e) => (
        <li key={e.id} className="flex items-center justify-between gap-3 py-2">
          <span>
            {PROCTOR_EVENT[e.eventType] ?? e.eventType}
            {Object.keys(e.meta ?? {}).length ? (
              <span className="ml-1 text-xs text-muted-foreground">
                ({Object.entries(e.meta).map(([k, v]) => `${k}: ${v}`).join(", ")})
              </span>
            ) : null}
          </span>
          <span className="text-xs text-muted-foreground tabular">{formatDateTime(e.createdAt)}</span>
        </li>
      ))}
    </ul>
  );
}

function InviteButton({ candidateId, instruments }: { candidateId: string; instruments: Instrument[] }) {
  return (
    <FormDialog
      title={t.inviteTitle}
      trigger={
        <Button size="sm">
          <Send className="size-4" /> {t.invite}
        </Button>
      }
    >
      {(close) => <InviteForm candidateId={candidateId} instruments={instruments.filter((i) => i.isActive)} close={close} />}
    </FormDialog>
  );
}

function InviteForm({ candidateId, instruments, close }: { candidateId: string; instruments: Instrument[]; close: () => void }) {
  const [picked, setPicked] = React.useState<string[]>(instruments.map((i) => i.id));
  const [days, setDays] = React.useState(3);
  const [link, setLink] = React.useState<string | null>(null);
  const { run, pending } = useErpAction();
  if (link)
    return (
      <div className="flex flex-col gap-3">
        <p className="text-sm text-muted-foreground">{rs.portalLinks.hint}</p>
        <PortalLink path={link} />
        <Button onClick={close} className="self-end">
          {rs.done}
        </Button>
      </div>
    );
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => invitePsikotesAction(candidateId, picked, days), {
          success: t.invited,
          onOk: (s) => (s.token ? setLink(`/psikotes/${s.token}`) : close()),
        });
      }}
    >
      <fieldset className="flex flex-col gap-2">
        <legend className="mb-1 text-sm font-medium">{t.instruments}</legend>
        {instruments.map((i) => (
          <div key={i.id} className="flex items-center gap-2">
            <Checkbox
              id={`ins-${i.id}`}
              checked={picked.includes(i.id)}
              onCheckedChange={(on) => setPicked((p) => (on ? [...p, i.id] : p.filter((x) => x !== i.id)))}
            />
            <Label htmlFor={`ins-${i.id}`} className="font-normal">
              {i.name}
              <span className="text-xs text-muted-foreground">
                {" "}
                · {INSTRUMENT_KIND[i.kind] ?? i.kind} · {rs.minutes(Math.round(i.durationSeconds / 60))}
              </span>
            </Label>
          </div>
        ))}
      </fieldset>
      <Fld label={t.expiresDays} htmlFor="inv-days">
        <Input id="inv-days" type="number" min={1} max={30} value={days} onChange={(e) => setDays(Number(e.target.value))} className="w-28" />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={t.invite} />
    </form>
  );
}

function SummaryCard({ candidateId, summary, canWrite }: { candidateId: string; summary: Panel["summary"]; canWrite: boolean }) {
  const [rec, setRec] = React.useState(summary?.recommendation ?? "");
  const [notes, setNotes] = React.useState(summary?.notes ?? "");
  const { run, pending } = useErpAction();
  return (
    <Card className="self-start">
      <CardHeader>
        <CardTitle>{t.summary}</CardTitle>
        {summary?.updatedByName ? (
          <p className="text-xs text-muted-foreground">{rs.screening.updatedBy(summary.updatedByName, formatDateTime(summary.updatedAt))}</p>
        ) : null}
      </CardHeader>
      <CardContent>
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => savePsikotesSummaryAction(candidateId, rec, notes), { success: t.summarySaved });
          }}
        >
          <Fld label={rs.screening.recommendation} htmlFor="psi-rec">
            <SimpleSelect
              id="psi-rec"
              value={rec}
              onChange={setRec}
              options={Object.entries(RECOMMENDATION).map(([value, label]) => ({ value, label }))}
              emptyLabel={rs.screening.noRecommendation}
              disabled={!canWrite}
            />
          </Fld>
          <Fld label={rs.screening.notes} htmlFor="psi-notes">
            <Textarea id="psi-notes" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)} disabled={!canWrite} />
          </Fld>
          {canWrite ? (
            <Button type="submit" size="sm" className="self-end" disabled={pending}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : null}
              {rs.save}
            </Button>
          ) : null}
        </form>
      </CardContent>
    </Card>
  );
}
