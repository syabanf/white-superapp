"use client";

import * as React from "react";
import { CheckCheck, Loader2, RotateCcw, Send } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { replyToDemoChat } from "../actions";
import { CHAT_SCENARIOS, type ChatStep } from "../data";
import { Panel } from "./shared";

function Bubble({ step, mine }: { step: ChatStep; mine: boolean }) {
  const bot = step.from === "bot";
  return (
    <div className={cn("flex", mine && "justify-end")}>
      <div className={cn("max-w-[85%] rounded-2xl px-3.5 py-2.5 text-sm shadow-(--card-shadow)", mine ? "bg-foreground text-background" : bot ? "border border-brand/25 bg-brand/5" : "bg-card")}>
        <p className={cn("mb-0.5 text-xs font-semibold", bot && "text-brand")}>{step.name}</p>
        <p className="break-words whitespace-pre-line">{step.text}</p>
      </div>
    </div>
  );
}

/** Live chat: the viewer types as one person in the room and the text AI answers that text. */
export function WhatsappDemo() {
  const [id, setId] = React.useState(CHAT_SCENARIOS[0].id);
  const [turns, setTurns] = React.useState<ChatStep[]>([]);
  const [draft, setDraft] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, startTransition] = React.useTransition();
  const log = React.useRef<HTMLDivElement>(null);
  const scenario = CHAT_SCENARIOS.find((s) => s.id === id) ?? CHAT_SCENARIOS[0];
  const steps = [...scenario.opening, ...turns];
  const effects = steps.flatMap((s) => (s.effect ? [s.effect] : []));

  React.useEffect(() => {
    log.current?.scrollTo({ top: log.current.scrollHeight });
  }, [turns, pending]);

  function reset() {
    setTurns([]);
    setDraft("");
    setError(null);
  }

  function send(event: React.FormEvent) {
    event.preventDefault();
    const text = draft.trim();
    if (!text || pending) return;
    const next = [...turns, { ...scenario.you, text }];
    setTurns(next);
    setDraft("");
    setError(null);
    startTransition(async () => {
      const res = await replyToDemoChat({ scenarioId: scenario.id, turns: next });
      if (res.ok) {
        setTurns([...next, { from: "bot", name: "Asisten Mixist", text: res.data.reply, effect: res.data.effect }]);
      } else {
        setTurns(turns);
        setDraft(text);
        setError(res.error);
      }
    });
  }

  return (
    <>
      <Tabs
        value={id}
        onValueChange={(v) => {
          setId(v);
          reset();
        }}
      >
        <TabsList aria-label="Skenario" className="max-w-full overflow-x-auto">
          {CHAT_SCENARIOS.map((s) => (
            <TabsTrigger key={s.id} value={s.id} disabled={pending}>
              {s.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <p className="text-sm text-muted-foreground">{scenario.intro}</p>

      <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <Panel
          title={scenario.room}
          action={
            turns.length ? (
              <Button size="sm" variant="outline" disabled={pending} onClick={reset}>
                <RotateCcw className="size-4" /> Ulangi
              </Button>
            ) : null
          }
        >
          <div ref={log} aria-live="polite" className="flex h-96 flex-col gap-2.5 overflow-y-auto bg-muted/40 p-(--card-pad)">
            {steps.map((s, i) => (
              <Bubble key={i} step={s} mine={s.name === scenario.you.name} />
            ))}
            {pending ? (
              <p className="flex items-center gap-2 text-xs text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" /> Asisten Mixist sedang mengetik
              </p>
            ) : null}
          </div>
          <form onSubmit={send} className="flex flex-col gap-2 border-t p-3">
            {error ? (
              <p role="alert" className="text-xs text-negative">
                {error}
              </p>
            ) : null}
            <div className="flex gap-2">
              <Input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                maxLength={2000}
                placeholder={scenario.placeholder}
                aria-label={`Ketik pesan sebagai ${scenario.you.name}`}
              />
              <Button type="submit" size="sm" disabled={pending || !draft.trim()}>
                <Send className="size-4" /> Kirim
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">Anda mengetik sebagai {scenario.you.name}.</p>
          </form>
        </Panel>

        <Panel title="Tercatat di dashboard" hint="Hasil yang akan muncul otomatis setelah terhubung">
          <ul className="flex flex-col gap-3 p-(--card-pad)">
            {effects.length ? (
              effects.map((e, i) => (
                <li key={i} className="flex gap-2.5 text-sm">
                  <CheckCheck className="mt-0.5 size-4 shrink-0 text-positive" />
                  <span>{e}</span>
                </li>
              ))
            ) : (
              <li className="text-sm text-muted-foreground">Belum ada catatan. Kirim pesan untuk memulai.</li>
            )}
          </ul>
        </Panel>
      </div>
    </>
  );
}
