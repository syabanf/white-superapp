"use client";

import * as React from "react";
import { ArrowLeft, ListChecks, Loader2, Pencil, Plus, Settings2, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect, useFieldErrors } from "@/features/erp/hr/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { cn } from "@/lib/utils";
import { deleteQuestionAction, listQuestionsAction, saveInstrumentAction, saveQuestionAction } from "../actions";
import { INSTRUMENT_KIND, rs } from "../strings";
import type { Instrument, McqOption, Question } from "../types";

const b = rs.bank;
const MCQ_KEYS = ["a", "b", "c", "d", "e", "f"];

export function InstrumentsView({ instruments, canWrite }: { instruments: Instrument[]; canWrite: boolean }) {
  if (!instruments.length) return <ErpEmpty />;
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {instruments.map((i) => (
        <Card key={i.id} className="flex flex-col gap-3 p-4">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <h3 className="font-semibold">{i.name}</h3>
              <p className="text-xs text-muted-foreground">
                {INSTRUMENT_KIND[i.kind] ?? i.kind} · {rs.minutes(Math.round(i.durationSeconds / 60))}
              </p>
            </div>
            <ErpStatusBadge status={i.isActive ? "ACTIVE" : "INACTIVE"} />
          </div>
          {i.instructions ? <p className="line-clamp-3 text-sm text-muted-foreground">{i.instructions}</p> : null}
          <p className="text-xs text-muted-foreground">
            {i.kind === "drawing"
              ? b.drawingNoQuestions
              : [b.activeQuestions(i.activeQuestions), i.questionCount ? `${b.questionCount}: ${i.questionCount}` : b.allQuestions, i.shuffle ? b.shuffle : null]
                  .filter(Boolean)
                  .join(" · ")}
          </p>
          <div className="mt-auto flex flex-wrap gap-1.5">
            {i.kind !== "drawing" ? (
              <FormDialog
                title={`${b.questions} · ${i.name}`}
                wide
                trigger={
                  <Button size="xs" variant="outline">
                    <ListChecks className="size-3.5" /> {b.manageQuestions}
                  </Button>
                }
              >
                {() => <QuestionManager instrument={i} canWrite={canWrite} />}
              </FormDialog>
            ) : null}
            {canWrite ? (
              <FormDialog
                title={b.editInstrument}
                trigger={
                  <Button size="xs" variant="ghost">
                    <Settings2 className="size-3.5" /> {b.editInstrument}
                  </Button>
                }
              >
                {(close) => <InstrumentForm instrument={i} close={close} />}
              </FormDialog>
            ) : null}
          </div>
        </Card>
      ))}
    </div>
  );
}

function InstrumentForm({ instrument, close }: { instrument: Instrument; close: () => void }) {
  const [v, setV] = React.useState({
    name: instrument.name,
    isActive: instrument.isActive,
    minutes: Math.round(instrument.durationSeconds / 60),
    questionCount: instrument.questionCount ? String(instrument.questionCount) : "",
    shuffle: instrument.shuffle,
    instructions: instrument.instructions,
  });
  const set = <K extends keyof typeof v>(k: K, val: (typeof v)[K]) => setV((p) => ({ ...p, [k]: val }));
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            saveInstrumentAction(instrument.id, {
              name: v.name,
              isActive: v.isActive,
              durationSeconds: v.minutes * 60,
              questionCount: v.questionCount ? Number(v.questionCount) : null,
              shuffle: v.shuffle,
              instructions: v.instructions,
            }),
          { success: b.instrumentSaved, onOk: close, onFail },
        );
      }}
    >
      <Fld label={b.name} htmlFor="ins-name" error={errors?.name}>
        <Input id="ins-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <div className="grid grid-cols-2 gap-4">
        <Fld label={b.duration} htmlFor="ins-dur" error={errors?.durationSeconds}>
          <Input id="ins-dur" type="number" min={1} value={v.minutes} onChange={(e) => set("minutes", Number(e.target.value))} />
        </Fld>
        {instrument.kind !== "drawing" ? (
          <Fld label={b.questionCount} htmlFor="ins-qc" hint={b.allQuestions} error={errors?.questionCount}>
            <Input id="ins-qc" type="number" min={1} value={v.questionCount} onChange={(e) => set("questionCount", e.target.value)} />
          </Fld>
        ) : null}
      </div>
      <Fld label={b.instructions} htmlFor="ins-instr">
        <Textarea id="ins-instr" rows={3} value={v.instructions} onChange={(e) => set("instructions", e.target.value)} />
      </Fld>
      <div className="flex flex-wrap gap-6">
        <div className="flex items-center gap-2">
          <Switch id="ins-active" checked={v.isActive} onCheckedChange={(x) => set("isActive", x)} />
          <Label htmlFor="ins-active">{b.active}</Label>
        </div>
        {instrument.kind === "mcq" ? (
          <div className="flex items-center gap-2">
            <Switch id="ins-shuffle" checked={v.shuffle} onCheckedChange={(x) => set("shuffle", x)} />
            <Label htmlFor="ins-shuffle">{b.shuffle}</Label>
          </div>
        ) : null}
      </div>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function QuestionManager({ instrument, canWrite }: { instrument: Instrument; canWrite: boolean }) {
  const [questions, setQuestions] = React.useState<Question[] | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [editing, setEditing] = React.useState<Question | "new" | null>(null);
  const [version, setVersion] = React.useState(0);
  const reload = () => setVersion((n) => n + 1);
  React.useEffect(() => {
    let alive = true;
    void listQuestionsAction(instrument.id).then((res) => {
      if (!alive) return;
      if (res.ok) setQuestions(res.data);
      else setError(res.error);
    });
    return () => {
      alive = false;
    };
  }, [instrument.id, version]);

  if (editing)
    return (
      <div className="flex flex-col gap-4">
        <Button variant="ghost" size="sm" className="-ml-2 self-start" onClick={() => setEditing(null)}>
          <ArrowLeft className="size-4" /> {b.back}
        </Button>
        <QuestionForm
          instrument={instrument}
          question={editing === "new" ? undefined : editing}
          nextOrder={(questions?.length ?? 0) + 1}
          done={() => {
            setEditing(null);
            reload();
          }}
        />
      </div>
    );
  if (error) return <ErpError compact message={error} />;
  if (!questions)
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="size-5 animate-spin text-muted-foreground" />
      </div>
    );
  return (
    <div className="flex flex-col gap-3">
      {canWrite ? (
        <Button size="sm" className="self-start" onClick={() => setEditing("new")}>
          <Plus className="size-4" /> {b.newQuestion}
        </Button>
      ) : null}
      {questions.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{b.noQuestions}</p>
      ) : (
        <ol className="flex flex-col gap-2">
          {questions.map((q, i) => (
            <li key={q.id} className={cn("rounded-lg border p-3 text-sm", !q.isActive && "opacity-60")}>
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="font-medium">
                    {i + 1}. {q.body || [q.statementA?.text, q.statementB?.text].filter(Boolean).join(" / ")}
                  </p>
                  {instrument.kind === "mcq" ? (
                    <p className="mt-1 text-xs text-muted-foreground">
                      {q.options.map((o) => `${o.key.toUpperCase()}. ${o.text}`).join("  ·  ")}
                      {q.correctKey ? ` · ${b.correctKey}: ${q.correctKey.toUpperCase()}` : null}
                    </p>
                  ) : (
                    <p className="mt-1 text-xs text-muted-foreground">
                      A ({q.statementA?.scale}) · B ({q.statementB?.scale})
                    </p>
                  )}
                </div>
                {canWrite ? (
                  <div className="flex shrink-0 gap-1">
                    <Button size="icon-sm" variant="ghost" onClick={() => setEditing(q)} aria-label={rs.edit}>
                      <Pencil className="size-4" />
                    </Button>
                    <ConfirmAction
                      title={b.deleteQuestionTitle}
                      confirmLabel={rs.delete}
                      success={b.questionDeleted}
                      action={async () => {
                        const res = await deleteQuestionAction(q.id);
                        if (res.ok) setQuestions((list) => list?.filter((x) => x.id !== q.id) ?? null);
                        return res;
                      }}
                      trigger={
                        <Button size="icon-sm" variant="ghost" aria-label={rs.delete}>
                          <Trash2 className="size-4" />
                        </Button>
                      }
                    />
                  </div>
                ) : null}
              </div>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}

const PAPI_SCALES = ["N", "G", "A", "L", "P", "I", "T", "V", "X", "S", "B", "O", "R", "D", "C", "Z", "E", "K", "F", "W"].map((s) => ({
  value: s,
  label: s,
}));

function QuestionForm({ instrument, question, nextOrder, done }: { instrument: Instrument; question?: Question; nextOrder: number; done: () => void }) {
  const mcq = instrument.kind === "mcq";
  const [body, setBody] = React.useState(question?.body ?? "");
  const [options, setOptions] = React.useState<McqOption[]>(
    question?.options.length ? question.options : MCQ_KEYS.slice(0, 4).map((key) => ({ key, text: "" })),
  );
  const [correctKey, setCorrectKey] = React.useState(question?.correctKey ?? "a");
  const [a, setA] = React.useState(question?.statementA ?? { text: "", scale: "N" });
  const [bb, setB] = React.useState(question?.statementB ?? { text: "", scale: "G" });
  const [sortOrder, setSortOrder] = React.useState(question?.sortOrder ?? nextOrder);
  const [isActive, setIsActive] = React.useState(question?.isActive ?? true);
  const { errors, onFail } = useFieldErrors();
  const { run, pending } = useErpAction();

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(
          () =>
            saveQuestionAction(instrument.id, question?.id ?? null, {
              body,
              sortOrder,
              isActive,
              ...(mcq ? { options: options.filter((o) => o.text.trim()), correctKey } : { statementA: a, statementB: bb }),
            }),
          { success: b.questionSaved, onOk: done, onFail },
        );
      }}
    >
      <Fld label={mcq ? b.body : b.bodyOptional} htmlFor="q-body" error={errors?.body}>
        <Textarea id="q-body" rows={2} value={body} onChange={(e) => setBody(e.target.value)} />
      </Fld>
      {mcq ? (
        <fieldset className="flex flex-col gap-2">
          <legend className="mb-1 text-sm font-medium">{b.options}</legend>
          {options.map((o, i) => (
            <div key={o.key} className="flex items-center gap-2">
              <span className="w-5 text-center text-sm font-semibold uppercase">{o.key}</span>
              <Input
                value={o.text}
                onChange={(e) => setOptions((list) => list.map((x, j) => (j === i ? { ...x, text: e.target.value } : x)))}
                aria-label={`${b.options} ${o.key.toUpperCase()}`}
              />
            </div>
          ))}
          {options.length < MCQ_KEYS.length ? (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="self-start"
              onClick={() => setOptions((list) => [...list, { key: MCQ_KEYS[list.length], text: "" }])}
            >
              <Plus className="size-4" /> {b.addOption}
            </Button>
          ) : null}
          {errors?.options ? <p className="text-xs text-destructive">{errors.options[0]}</p> : null}
        </fieldset>
      ) : null}
      {mcq ? (
        <Fld label={b.correctKey} htmlFor="q-key" error={errors?.correctKey}>
          <SimpleSelect id="q-key" value={correctKey} onChange={setCorrectKey} options={options.map((o) => ({ value: o.key, label: o.key.toUpperCase() }))} />
        </Fld>
      ) : (
        <>
          {([
            [b.statementA, a, setA, "q-a"],
            [b.statementB, bb, setB, "q-b"],
          ] as const).map(([label, st, setSt, id]) => (
            <div key={id} className="grid grid-cols-[1fr_6rem] gap-2">
              <Fld label={label} htmlFor={id}>
                <Input id={id} value={st.text} onChange={(e) => setSt({ ...st, text: e.target.value })} />
              </Fld>
              <Fld label={b.scale} htmlFor={`${id}-scale`}>
                <SimpleSelect id={`${id}-scale`} value={st.scale} onChange={(x) => setSt({ ...st, scale: x })} options={PAPI_SCALES} />
              </Fld>
            </div>
          ))}
        </>
      )}
      <div className="flex flex-wrap items-end gap-6">
        <Fld label={b.sortOrder} htmlFor="q-order">
          <Input id="q-order" type="number" min={0} value={sortOrder} onChange={(e) => setSortOrder(Number(e.target.value))} className="w-24" />
        </Fld>
        <div className="flex items-center gap-2 pb-2">
          <Switch id="q-active" checked={isActive} onCheckedChange={setIsActive} />
          <Label htmlFor="q-active">{b.active}</Label>
        </div>
      </div>
      <FormFooter onCancel={done} pending={pending} />
    </form>
  );
}
