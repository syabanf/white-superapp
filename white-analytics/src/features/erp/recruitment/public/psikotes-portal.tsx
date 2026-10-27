"use client";

import * as React from "react";
import { ArrowLeft, CheckCircle2, ChevronLeft, ChevronRight, Clock, Hourglass, Loader2, PencilRuler, PlayCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDateTime } from "@/lib/format";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { formatCountdown } from "../lib";
import { TEST_STATUS, cp, kinds } from "../strings";
import type { PortalAction, PortalTestRun, PsikotesPortal as Portal, Question } from "../types";
import { errorMessage, portalFetch, portalPost } from "./client";
import { requestFullscreen, useCountdown, useProctoring } from "./hooks";
import { PortalNotice } from "./states";

const t = cp.psikotes;
const TERMINAL = ["selesai", "perlu_review", "reviewed"];

export function PsikotesPortal({ token, initial }: { token: string; initial: Portal }) {
  const base = `/v1/recruitment/public/psikotes/${encodeURIComponent(token)}`;
  const [portal, setPortal] = React.useState(initial);
  const [run, setRun] = React.useState<PortalTestRun | null>(null);
  const [busy, setBusy] = React.useState<string | null>(null);
  useProctoring(`${base}/proctor-events`, portal.status === "in_progress");

  const act = async (key: string, fn: () => Promise<void>) => {
    setBusy(key);
    try {
      await fn();
    } catch (e) {
      toast.error(errorMessage(e));
    } finally {
      setBusy(null);
    }
  };
  const refresh = async () => setPortal(await portalFetch<Portal>(base));

  if (portal.status === "completed")
    return <PortalNotice icon={<CheckCircle2 className="size-5" />} title={t.completedTitle} description={t.completedDesc} />;
  if (portal.status === "expired")
    return <PortalNotice icon={<Hourglass className="size-5" />} title={t.expiredTitle} description={t.expiredDesc} />;

  const header = (
    <div>
      <p className="label-mono text-muted-foreground">{t.eyebrow}</p>
      <h1 className="mt-1 text-[26px] leading-tight font-semibold">{t.hello(portal.candidateName)}</h1>
      <p className="mt-1 text-sm text-muted-foreground">{t.intro(portal.positionTitle)}</p>
    </div>
  );

  if (portal.status !== "in_progress")
    return (
      <>
        {header}
        <Card className="flex flex-col gap-4 p-(--card-pad)">
          <h2 className="font-semibold">{t.rulesTitle}</h2>
          <ul className="list-disc space-y-1.5 pl-5 text-sm">
            {t.rules.map((r) => (
              <li key={r}>{r}</li>
            ))}
          </ul>
          <p className="text-sm">
            {t.tests}: {portal.tests.map((x) => x.instrument?.name).filter(Boolean).join(", ")}
          </p>
          {portal.expiresAt ? <p className="text-xs text-muted-foreground">{t.expiresAt(formatDateTime(portal.expiresAt))}</p> : null}
          <Button
            size="lg"
            className="w-full sm:w-auto sm:self-start"
            disabled={busy !== null}
            onClick={() => {
              requestFullscreen();
              void act("start", async () => setPortal(await portalPost<Portal>(`${base}/start`, { webcamConsent: false })));
            }}
          >
            {busy === "start" ? <Loader2 className="size-4 animate-spin" /> : <PlayCircle className="size-4" />}
            {t.start}
          </Button>
        </Card>
      </>
    );

  if (run) {
    const finished = async () => {
      setRun(null);
      await refresh().catch(() => undefined);
    };
    return run.test.instrument?.kind === "drawing" ? (
      <DrawingRunner base={base} run={run} onDone={finished} />
    ) : (
      <QuestionRunner base={base} run={run} onDone={finished} />
    );
  }

  const allDone = portal.tests.every((x) => TERMINAL.includes(x.status));
  return (
    <>
      {header}
      <div className="flex flex-col gap-3">
        <h2 className="text-base font-semibold">{t.tests}</h2>
        {portal.tests.map((test) => {
          const done = TERMINAL.includes(test.status);
          return (
            <Card key={test.id} className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between p-(--card-pad)">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2 font-medium">
                  {test.instrument?.name}
                  <ErpStatusBadge status={test.status} labels={TEST_STATUS} kinds={kinds.test} />
                </div>
                <p className="mt-0.5 flex items-center gap-1 text-xs text-muted-foreground">
                  <Clock className="size-3.5" /> {t.duration(Math.round((test.instrument?.durationSeconds ?? 0) / 60))}
                </p>
                {test.instrument?.instructions && !done ? <p className="mt-1.5 text-sm text-muted-foreground">{test.instrument.instructions}</p> : null}
              </div>
              {done ? (
                <span className="flex items-center gap-1 text-sm text-positive">
                  <CheckCircle2 className="size-4" /> {t.done}
                </span>
              ) : (
                <Button
                  className="w-full sm:w-auto"
                  disabled={busy !== null}
                  onClick={() =>
                    void act(test.id, async () =>
                      setRun(await portalPost<PortalTestRun>(`${base}/tests/${encodeURIComponent(test.id)}/start`)),
                    )
                  }
                >
                  {busy === test.id ? <Loader2 className="size-4 animate-spin" /> : null}
                  {test.status === "in_progress" ? t.continueTest : t.startTest}
                </Button>
              )}
            </Card>
          );
        })}
      </div>
      <div className="flex flex-col gap-2 sm:items-end">
        <Button
          size="lg"
          disabled={!allDone || busy !== null}
          className="w-full sm:w-auto"
          onClick={() =>
            void act("finish", async () => {
              const res = await portalPost<PortalAction>(`${base}/finish`);
              setPortal((p) => ({ ...p, status: res.status || "completed" }));
            })
          }
        >
          {busy === "finish" ? <Loader2 className="size-4 animate-spin" /> : null}
          {t.finishAll}
        </Button>
        {!allDone ? <p className="text-xs text-muted-foreground">{t.finishHint}</p> : null}
      </div>
    </>
  );
}

function Timer({ remaining }: { remaining: number | null }) {
  if (remaining === null) return null;
  return (
    <span
      role="timer"
      aria-live="off"
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-semibold tabular",
        remaining <= 60 ? "bg-destructive/10 text-destructive" : "bg-muted",
      )}
    >
      <Clock className="size-3.5" /> {formatCountdown(remaining)}
    </span>
  );
}

/** Choices of a question: MCQ options, or the PAPI pair as "a"/"b". */
function choicesOf(q: Question, kind: string) {
  if (kind === "forced_choice")
    return [
      { key: "a", text: q.statementA?.text ?? "" },
      { key: "b", text: q.statementB?.text ?? "" },
    ];
  return q.options;
}

function QuestionRunner({ base, run, onDone }: { base: string; run: PortalTestRun; onDone: () => Promise<void> }) {
  const testPath = `${base}/tests/${encodeURIComponent(run.test.id)}`;
  const kind = run.test.instrument?.kind ?? "mcq";
  const questions = run.questions;
  const [answers, setAnswers] = React.useState<Record<string, string>>(run.savedAnswers ?? {});
  const [index, setIndex] = React.useState(() => Math.max(0, questions.findIndex((q) => !run.savedAnswers?.[q.id])));
  const [saving, setSaving] = React.useState(0);
  const [finishing, setFinishing] = React.useState(false);
  const answersRef = React.useRef(answers);
  const advance = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  React.useEffect(() => () => {
    if (advance.current) clearTimeout(advance.current);
  }, []);

  const finish = async () => {
    if (finishing) return;
    setFinishing(true);
    try {
      await portalPost<PortalAction>(`${testPath}/finish`, { answers: answersRef.current });
      await onDone();
    } catch (e) {
      toast.error(errorMessage(e));
      setFinishing(false);
    }
  };
  const remaining = useCountdown(run.endsAt, () => {
    toast.error(t.timeUp);
    void finish();
  });
  const locked = finishing || remaining === 0;
  const q = questions[index];
  const answered = Object.keys(answers).filter((k) => questions.some((x) => x.id === k)).length;

  const choose = (key: string) => {
    if (locked || !q) return;
    const next = { ...answersRef.current, [q.id]: key };
    answersRef.current = next;
    setAnswers(next);
    setSaving((n) => n + 1);
    void portalFetch(`${testPath}/answers`, { method: "PUT", body: JSON.stringify({ answers: { [q.id]: key } }) })
      .catch((e) => toast.error(errorMessage(e)))
      .finally(() => setSaving((n) => n - 1));
    if (index < questions.length - 1) {
      if (advance.current) clearTimeout(advance.current);
      advance.current = setTimeout(() => setIndex((i) => Math.min(i + 1, questions.length - 1)), 200);
    }
  };

  return (
    <Card className="flex flex-col gap-4 p-(--card-pad)">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-base font-semibold">{run.test.instrument?.name}</h1>
        <Timer remaining={remaining} />
      </div>
      <div className="flex flex-wrap gap-1.5" aria-label={t.answered(answered, questions.length)}>
        {questions.map((x, i) => (
          <button
            key={x.id}
            type="button"
            onClick={() => setIndex(i)}
            aria-label={t.question(i + 1)}
            aria-current={i === index ? "step" : undefined}
            className={cn(
              "size-8 rounded-md text-xs font-medium tabular transition-colors",
              i === index ? "bg-brand text-white" : answers[x.id] ? "bg-positive/15 text-positive" : "bg-muted text-muted-foreground",
            )}
          >
            {i + 1}
          </button>
        ))}
      </div>
      {q ? (
        <div className="flex flex-col gap-3">
          <p className="text-xs text-muted-foreground">
            {t.question(index + 1)}
            {kind === "forced_choice" ? ` · ${t.papiHint}` : null}
          </p>
          {q.body ? <p className="text-base font-medium whitespace-pre-wrap">{q.body}</p> : null}
          <div className="grid gap-2" role="radiogroup" aria-label={t.question(index + 1)}>
            {choicesOf(q, kind).map((o) => (
              <button
                key={o.key}
                type="button"
                role="radio"
                aria-checked={answers[q.id] === o.key}
                disabled={locked}
                onClick={() => choose(o.key)}
                className={cn(
                  "flex min-h-12 items-center gap-3 rounded-lg border px-4 py-3 text-left text-sm transition-colors disabled:opacity-60",
                  answers[q.id] === o.key ? "border-brand bg-brand/5 font-medium" : "hover:bg-accent/60",
                )}
              >
                <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold uppercase">{o.key}</span>
                {o.text}
              </button>
            ))}
          </div>
          <div className="flex justify-between gap-2">
            <Button variant="ghost" size="sm" disabled={index === 0} onClick={() => setIndex((i) => i - 1)}>
              <ChevronLeft className="size-4" /> {t.prev}
            </Button>
            <Button variant="ghost" size="sm" disabled={index >= questions.length - 1} onClick={() => setIndex((i) => i + 1)}>
              {t.next} <ChevronRight className="size-4" />
            </Button>
          </div>
        </div>
      ) : null}
      <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-3">
        <p className="text-xs text-muted-foreground">
          {t.answered(answered, questions.length)} · {saving > 0 ? t.saving : t.savedAuto}
        </p>
        <ConfirmAction
          title={t.finishTest}
          description={t.finishConfirm}
          confirmLabel={t.finishTest}
          onConfirm={finish}
          trigger={
            <Button disabled={finishing}>
              {finishing ? <Loader2 className="size-4 animate-spin" /> : null}
              {t.finishTest}
            </Button>
          }
        />
      </div>
    </Card>
  );
}

function DrawingRunner({ base, run, onDone }: { base: string; run: PortalTestRun; onDone: () => Promise<void> }) {
  const testPath = `${base}/tests/${encodeURIComponent(run.test.id)}`;
  const [uploaded, setUploaded] = React.useState(run.test.hasAttachment);
  const [finishing, setFinishing] = React.useState(false);
  const finish = async () => {
    if (finishing) return;
    setFinishing(true);
    try {
      await portalPost<PortalAction>(`${testPath}/finish`, { answers: {} });
      await onDone();
    } catch (e) {
      toast.error(errorMessage(e));
      setFinishing(false);
    }
  };
  const remaining = useCountdown(run.endsAt, () => void finish());

  return (
    <Card className="flex flex-col gap-4 p-(--card-pad)">
      <div className="flex items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-base font-semibold">
          <PencilRuler className="size-4" /> {run.test.instrument?.name}
        </h1>
        <Timer remaining={remaining} />
      </div>
      {run.test.instrument?.instructions ? <p className="text-sm">{run.test.instrument.instructions}</p> : null}
      <p className="text-sm text-muted-foreground">{t.drawingHint}</p>
      {uploaded ? (
        <p className="flex items-center gap-1.5 text-sm text-positive">
          <CheckCircle2 className="size-4" /> {t.drawingUploaded}
        </p>
      ) : null}
      <FileUpload
        url={`/api/public-erp${testPath.replace(/^\/v1\//, "/api/")}/upload`}
        accept="image/jpeg,image/png,image/webp"
        acceptLabel="JPG, PNG, WEBP"
        maxMb={8}
        multiple={false}
        disabled={finishing || remaining === 0}
        onUploaded={() => setUploaded(true)}
      />
      <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-3">
        <Button variant="ghost" size="sm" onClick={() => void onDone()} disabled={finishing}>
          <ArrowLeft className="size-4" /> {t.back}
        </Button>
        <Button onClick={() => (uploaded ? void finish() : toast.error(t.drawingMissing))} disabled={finishing}>
          {finishing ? <Loader2 className="size-4 animate-spin" /> : null}
          {t.finishTest}
        </Button>
      </div>
    </Card>
  );
}
