"use client";

import * as React from "react";
import { Bot, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Fld, FormDialog, FormFooter } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDateTime } from "@/lib/format";
import { inviteAiInterviewAction } from "../actions";
import { RELEVANCE, SESSION_STATUS, kinds, rs } from "../strings";
import type { AiInterviewSession, InterviewSummary } from "../types";
import { ProctorButton } from "./psikotes-panel";
import { Chips, Info, PortalLink } from "./ui";

const a = rs.ai;

export function AiInterviewPanel({ candidateId, sessions, canWrite }: { candidateId: string; sessions: AiInterviewSession[]; canWrite: boolean }) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-base font-semibold">{a.title}</h2>
        {canWrite ? <InviteButton candidateId={candidateId} /> : null}
      </div>
      {sessions.length === 0 ? <ErpEmpty compact title={a.empty} icon={<Bot className="size-5" />} /> : sessions.map((s) => <SessionCard key={s.id} session={s} canWrite={canWrite} />)}
    </div>
  );
}

function SessionCard({ session, canWrite }: { session: AiInterviewSession; canWrite: boolean }) {
  const answered = session.turns.filter((t) => t.answeredAt).length;
  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle className="flex items-center gap-2">
          {formatDateTime(session.invitedAt)}
          <ErpStatusBadge status={session.status} labels={SESSION_STATUS} kinds={kinds.session} />
        </CardTitle>
        <ProctorButton sessionType="interview" sessionId={session.id} flags={session.proctor?.flags ?? 0} />
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Info label={rs.psikotes.expiresAt}>{formatDateTime(session.expiresAt)}</Info>
          <Info label={rs.psikotes.completedAt}>{session.completedAt ? formatDateTime(session.completedAt) : null}</Info>
          <Info label={a.maxQuestions}>{a.questionsOf(answered, session.maxQuestions)}</Info>
          <Info label={a.createdBy}>{session.createdByName}</Info>
        </dl>
        {canWrite && session.token && ["sent", "in_progress"].includes(session.status) ? <PortalLink path={`/wawancara/${session.token}`} /> : null}
        {session.status === "completed" ? <SummaryBlock summary={session.aiSummary} model={session.summaryModel} /> : null}
        {session.turns.length ? (
          <div>
            <h3 className="mb-2 text-sm font-semibold">{a.transcript}</h3>
            <ol className="flex flex-col gap-3">
              {session.turns.map((t) => (
                <li key={t.id} className="rounded-lg border p-3 text-sm">
                  <p className="font-medium">
                    {t.turnNo}. {t.question}
                  </p>
                  <p className="mt-1.5 whitespace-pre-wrap text-muted-foreground">{t.answerTranscript || a.notAnswered}</p>
                </li>
              ))}
            </ol>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function SummaryBlock({ summary, model }: { summary: InterviewSummary | null; model: string }) {
  if (!summary) return <p className="text-sm text-muted-foreground">{a.noSummary}</p>;
  return (
    <div className="flex flex-col gap-4 rounded-xl bg-muted/40 p-4 text-sm">
      <div>
        <div className="mb-1.5 flex items-baseline justify-between gap-2">
          <span className="text-xs text-muted-foreground">
            {a.relevance} · {RELEVANCE[summary.relevanceConclusion] ?? summary.relevanceConclusion}
          </span>
          <span className="text-lg font-semibold tabular">{summary.relevanceScore}/100</span>
        </div>
        <Progress value={summary.relevanceScore} />
        {summary.relevanceReason ? <p className="mt-2 text-muted-foreground">{summary.relevanceReason}</p> : null}
      </div>
      <p className="whitespace-pre-wrap">{summary.summary}</p>
      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <div className="mb-1.5 text-xs text-muted-foreground">{a.skills}</div>
          <Chips items={summary.skills} />
        </div>
        <Info label={a.salary}>{summary.salaryMentioned ? [summary.salaryValue, summary.salaryNote].filter(Boolean).join(" · ") : null}</Info>
        <List title={a.redFlags} items={summary.redFlags} />
        <List title={a.followUps} items={summary.followUps} />
      </div>
      {summary.limitations ? <p className="text-xs text-muted-foreground">{summary.limitations}</p> : null}
      {model ? <p className="text-xs text-muted-foreground">Model {model}</p> : null}
    </div>
  );
}

function List({ title, items }: { title: string; items: string[] }) {
  return (
    <div>
      <div className="mb-1.5 text-xs text-muted-foreground">{title}</div>
      {items.length ? (
        <ul className="list-disc space-y-1 pl-4">
          {items.map((x) => (
            <li key={x}>{x}</li>
          ))}
        </ul>
      ) : (
        <span className="text-muted-foreground">{rs.none}</span>
      )}
    </div>
  );
}

function InviteButton({ candidateId }: { candidateId: string }) {
  return (
    <FormDialog
      title={a.inviteTitle}
      trigger={
        <Button size="sm">
          <Send className="size-4" /> {a.invite}
        </Button>
      }
    >
      {(close) => <InviteForm candidateId={candidateId} close={close} />}
    </FormDialog>
  );
}

function InviteForm({ candidateId, close }: { candidateId: string; close: () => void }) {
  const [days, setDays] = React.useState(7);
  const [max, setMax] = React.useState(8);
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
        void run(() => inviteAiInterviewAction(candidateId, days, max), {
          success: a.invited,
          onOk: (s) => (s.token ? setLink(`/wawancara/${s.token}`) : close()),
        });
      }}
    >
      <div className="grid grid-cols-2 gap-4">
        <Fld label={a.maxQuestions} htmlFor="ai-max">
          <Input id="ai-max" type="number" min={3} max={15} value={max} onChange={(e) => setMax(Number(e.target.value))} />
        </Fld>
        <Fld label={a.expiresDays} htmlFor="ai-days">
          <Input id="ai-days" type="number" min={1} max={30} value={days} onChange={(e) => setDays(Number(e.target.value))} />
        </Fld>
      </div>
      <FormFooter onCancel={close} pending={pending} submitLabel={a.invite} />
    </form>
  );
}
