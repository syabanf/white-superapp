"use client";

import * as React from "react";
import { CheckCheck, Play, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { CHAT_SCENARIOS, type ChatStep } from "../data";
import { Panel } from "./shared";

function Bubble({ step }: { step: ChatStep }) {
  const mine = step.from === "me";
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

/** Scripted chat: each click reveals one message and what it records in the dashboard. */
export function WhatsappDemo() {
  const [id, setId] = React.useState(CHAT_SCENARIOS[0].id);
  const [shown, setShown] = React.useState(1);
  const scenario = CHAT_SCENARIOS.find((s) => s.id === id) ?? CHAT_SCENARIOS[0];
  const steps = scenario.steps.slice(0, shown);
  const effects = steps.flatMap((s) => (s.effect ? [s.effect] : []));
  const finished = shown >= scenario.steps.length;

  return (
    <>
      <Tabs
        value={id}
        onValueChange={(v) => {
          setId(v);
          setShown(1);
        }}
      >
        <TabsList aria-label="Skenario" className="max-w-full overflow-x-auto">
          {CHAT_SCENARIOS.map((s) => (
            <TabsTrigger key={s.id} value={s.id}>
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
            finished ? (
              <Button size="sm" variant="outline" onClick={() => setShown(1)}>
                <RotateCcw className="size-4" /> Ulangi
              </Button>
            ) : (
              <Button size="sm" onClick={() => setShown((n) => n + 1)}>
                <Play className="size-4" /> Pesan berikutnya
              </Button>
            )
          }
        >
          <div aria-live="polite" className="flex min-h-80 flex-col gap-2.5 bg-muted/40 p-(--card-pad)">
            {steps.map((s, i) => (
              <Bubble key={i} step={s} />
            ))}
          </div>
        </Panel>

        <Panel title="Tercatat di dashboard" hint="Hasil yang akan muncul otomatis setelah terhubung">
          <ul className="flex flex-col gap-3 p-(--card-pad)">
            {effects.length ? (
              effects.map((e) => (
                <li key={e} className="flex gap-2.5 text-sm">
                  <CheckCheck className="mt-0.5 size-4 shrink-0 text-positive" />
                  <span>{e}</span>
                </li>
              ))
            ) : (
              <li className="text-sm text-muted-foreground">Belum ada catatan. Klik Pesan berikutnya.</li>
            )}
          </ul>
        </Panel>
      </div>
    </>
  );
}
