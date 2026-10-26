"use client";

import * as React from "react";
import { Bot, CheckCircle2, Hourglass, Loader2, PlayCircle, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { cp } from "../strings";
import type { InterviewPortal as Portal, InterviewTurnResponse, PortalTurn } from "../types";
import { errorMessage, portalPost } from "./client";
import { useProctoring } from "./hooks";
import { PortalNotice } from "./states";

const t = cp.interview;

/** Text AI interview: one question at a time, answered by typing. */
export function InterviewPortal({ token, initial }: { token: string; initial: Portal }) {
  const base = `/v1/recruitment/public/interviews/${encodeURIComponent(token)}`;
  const [status, setStatus] = React.useState(initial.status);
  const [turns, setTurns] = React.useState<PortalTurn[]>(initial.turns.filter((x) => x.answeredAt));
  const [current, setCurrent] = React.useState<PortalTurn | null>(initial.currentTurn);
  const [consent, setConsent] = React.useState(false);
  const [consentError, setConsentError] = React.useState(false);
  const [answer, setAnswer] = React.useState("");
  const [busy, setBusy] = React.useState(false);
  const endRef = React.useRef<HTMLDivElement>(null);
  useProctoring(`${base}/proctor-events`, status === "in_progress");

  React.useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "end" });
  }, [turns.length, current?.id, busy]);

  const apply = (res: InterviewTurnResponse) => {
    if (res.done) setStatus("completed");
    else setCurrent(res.turn);
  };

  const start = async () => {
    if (!consent) return setConsentError(true);
    setBusy(true);
    try {
      apply(await portalPost<InterviewTurnResponse>(`${base}/start`, { webcamConsent: true }));
      setStatus("in_progress");
    } catch (e) {
      toast.error(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const send = async () => {
    const text = answer.trim();
    if (!current || !text) return;
    setBusy(true);
    try {
      const res = await portalPost<InterviewTurnResponse>(`${base}/answer`, { turnId: current.id, answerText: text });
      setTurns((list) => [...list, { ...current, answerTranscript: text, answeredAt: new Date().toISOString() }]);
      setAnswer("");
      setCurrent(null);
      apply(res);
    } catch (e) {
      toast.error(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  if (status === "completed")
    return <PortalNotice icon={<CheckCircle2 className="size-5" />} title={t.completedTitle} description={t.completedDesc} />;
  if (status === "expired")
    return <PortalNotice icon={<Hourglass className="size-5" />} title={t.expiredTitle} description={t.expiredDesc} />;

  const header = (
    <div>
      <p className="label-mono text-muted-foreground">{t.eyebrow}</p>
      <h1 className="mt-1 text-[26px] leading-tight font-semibold">{t.hello(initial.candidateName)}</h1>
      <p className="mt-1 text-sm text-muted-foreground">{t.intro(initial.positionTitle, initial.maxQuestions)}</p>
    </div>
  );

  if (status !== "in_progress")
    return (
      <>
        {header}
        <Card className="flex flex-col gap-4 p-5">
          <ul className="list-disc space-y-1.5 pl-5 text-sm">
            {t.rules.map((r) => (
              <li key={r}>{r}</li>
            ))}
          </ul>
          <div className="flex items-start gap-2">
            <Checkbox
              id="iv-consent"
              checked={consent}
              onCheckedChange={(v) => {
                setConsent(v === true);
                setConsentError(false);
              }}
              aria-invalid={consentError || undefined}
            />
            <Label htmlFor="iv-consent" className="leading-snug font-normal">
              {t.consent}
            </Label>
          </div>
          {consentError ? <p className="text-xs text-destructive">{t.consentRequired}</p> : null}
          <Button size="lg" className="w-full sm:w-auto sm:self-start" onClick={() => void start()} disabled={busy}>
            {busy ? <Loader2 className="size-4 animate-spin" /> : <PlayCircle className="size-4" />}
            {t.start}
          </Button>
        </Card>
      </>
    );

  return (
    <>
      {header}
      <div className="flex flex-col gap-3" aria-live="polite">
        {turns.map((turn) => (
          <React.Fragment key={turn.id}>
            <Bubble who="ai">{turn.question}</Bubble>
            <Bubble who="me">{turn.answerTranscript}</Bubble>
          </React.Fragment>
        ))}
        {current ? <Bubble who="ai">{current.question}</Bubble> : null}
        {busy && !current ? (
          <p className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin" /> {t.thinking}
          </p>
        ) : null}
        <div ref={endRef} />
      </div>
      {current ? (
        <form
          className="sticky bottom-0 -mx-4 flex flex-col gap-2 border-t bg-canvas/95 px-4 py-3 backdrop-blur md:mx-0 md:rounded-xl md:border md:px-4"
          onSubmit={(e) => {
            e.preventDefault();
            void send();
          }}
        >
          <p className="text-xs text-muted-foreground">{t.progress(current.turnNo, initial.maxQuestions)}</p>
          <Textarea
            rows={4}
            value={answer}
            onChange={(e) => setAnswer(e.target.value)}
            placeholder={t.placeholder}
            aria-label={t.placeholder}
            disabled={busy}
            maxLength={4000}
          />
          <Button type="submit" disabled={busy || !answer.trim()} className="w-full sm:w-auto sm:self-end">
            {busy ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
            {t.send}
          </Button>
        </form>
      ) : null}
    </>
  );
}

function Bubble({ who, children }: { who: "ai" | "me"; children: React.ReactNode }) {
  return (
    <div className={cn("flex gap-2", who === "me" && "flex-row-reverse")}>
      {who === "ai" ? (
        <span className="mt-1 flex size-7 shrink-0 items-center justify-center rounded-full bg-brand/10 text-brand" aria-label={t.interviewer}>
          <Bot className="size-4" />
        </span>
      ) : null}
      <div
        className={cn(
          "max-w-[85%] rounded-2xl px-4 py-2.5 text-sm whitespace-pre-wrap",
          who === "ai" ? "rounded-tl-sm bg-card shadow-xs" : "rounded-tr-sm bg-brand text-white",
        )}
      >
        <span className="sr-only">{who === "ai" ? t.interviewer : t.you}: </span>
        {children}
      </div>
    </div>
  );
}
