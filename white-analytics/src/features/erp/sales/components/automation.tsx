"use client";

import * as React from "react";
import Link from "next/link";
import { History, Loader2, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDateTime } from "@/lib/format";
import { toast } from "@/lib/toast";
import {
  deleteCustomFieldAction,
  deleteScoringRuleAction,
  deleteWorkflowRuleAction,
  recalculateScoresAction,
  saveCustomFieldAction,
  saveScoringRuleAction,
  saveStageAction,
  saveWorkflowRuleAction,
} from "../actions";
import {
  ACTIVITY_TYPE,
  CONDITION_OP,
  CUSTOM_FIELD_TYPE,
  PRIORITY,
  RECORD_OBJECT,
  RUN_STATUS,
  SCORING_EVENT,
  SCORING_KIND,
  TRIGGER,
  WORKFLOW_ACTION,
  lbl,
  opts,
  ss,
} from "../strings";
import type { CustomField, Pipeline, ScoringRule, Stage, WorkflowAction, WorkflowRule, WorkflowRun } from "../types";
import { ChipToggle, ConfirmAction, Fld, FormDialog, FormFooter, type Option, Picker, SalesBadge, SimpleSelect } from "./ui";

const a = ss.automation;

/** Text box value → JSON value: numbers, booleans, arrays and objects parse, the rest stays a string. */
function parseValue(text: string): unknown {
  const t = text.trim();
  if (t === "") return null;
  try {
    return JSON.parse(t);
  } catch {
    return t;
  }
}
const showValue = (v: unknown) => (v == null ? "" : typeof v === "string" ? v : JSON.stringify(v));

const EMPTY_ACTION: WorkflowAction = {
  type: "create_task",
  title: "",
  activityType: "tugas",
  notes: "",
  dueInDays: 1,
  priority: "normal",
  assignTo: "owner",
  userId: "",
  strategy: "",
  userIds: [],
  onlyIfEmpty: false,
  field: "",
  value: null,
  to: "owner",
  message: "",
  role: "",
  hours: 0,
  days: 0,
};

// Workflows

export function WorkflowRules({ rules, owners, runs }: { rules: WorkflowRule[]; owners: Option[]; runs?: { ruleId: string; items: WorkflowRun[] } }) {
  return (
    <div className="flex flex-col gap-3">
      <div>
        <RuleDialog
          rule={null}
          owners={owners}
          trigger={
            <Button size="sm">
              <Plus className="size-3.5" /> {a.addRule}
            </Button>
          }
        />
      </div>
      <ul className="divide-y rounded-xl border bg-card">
        {rules.map((r) => (
          <li key={r.id} className="flex flex-col gap-2 px-5 py-3">
            <div className="flex flex-wrap items-center gap-3">
              <span className="min-w-0 flex-1">
                <span className="block font-medium">{r.name}</span>
                <span className="text-xs text-muted-foreground">
                  {lbl(RECORD_OBJECT, r.object)} · {lbl(TRIGGER, r.triggerType)} · {r.conditions.length} {a.conditions.toLowerCase()} ·{" "}
                  {r.actions.map((x) => lbl(WORKFLOW_ACTION, x.type)).join(", ")}
                </span>
              </span>
              <span className="text-xs text-muted-foreground">
                {a.runCount(r.runCount)}
                {r.lastRunAt ? ` · ${a.lastRun} ${formatDateTime(r.lastRunAt)}` : ""}
              </span>
              {!r.isActive ? <SalesBadge value="ended" map={{ ended: ss.inactive }} /> : null}
              <div className="flex">
                <Button size="icon-sm" variant="ghost" asChild aria-label={a.runs}>
                  <Link href={`?runs=${r.id}`} scroll={false}>
                    <History className="size-3.5" />
                  </Link>
                </Button>
                <RuleDialog
                  rule={r}
                  owners={owners}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
                <ConfirmAction
                  title={ss.confirmDelete}
                  action={() => deleteWorkflowRuleAction(r.id)}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </div>
            </div>
            {runs?.ruleId === r.id ? <RunList runs={runs.items} /> : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function RunList({ runs }: { runs: WorkflowRun[] }) {
  if (!runs.length) return <p className="text-xs text-muted-foreground">{a.noRuns}</p>;
  return (
    <ul className="max-h-64 space-y-1 overflow-y-auto rounded-lg bg-muted/40 p-2 text-xs">
      {runs.map((run) => (
        <li key={run.id} className="flex flex-wrap items-center gap-2">
          <span className="tabular text-muted-foreground">{formatDateTime(run.createdAt)}</span>
          <SalesBadge value={run.status} map={RUN_STATUS} />
          <span>
            {lbl(RECORD_OBJECT, run.subjectType)} <code className="font-mono">{run.subjectId.slice(0, 8)}</code>
          </span>
          {run.error ? <span className="text-destructive">{run.error}</span> : null}
        </li>
      ))}
    </ul>
  );
}

function RuleDialog({ rule, owners, trigger }: { rule: WorkflowRule | null; owners: Option[]; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={a.ruleTitle} size="xl">
      {(close) => <RuleForm rule={rule} owners={owners} close={close} />}
    </FormDialog>
  );
}

function RuleForm({ rule, owners, close }: { rule: WorkflowRule | null; owners: Option[]; close: () => void }) {
  const [v, setV] = React.useState<WorkflowRule>(
    rule ?? {
      id: "",
      name: "",
      object: "lead",
      triggerType: "created",
      triggerConfig: {},
      conditions: [],
      actions: [{ ...EMPTY_ACTION }],
      runOncePerRecord: false,
      isActive: true,
      runCount: 0,
      lastRunAt: null,
    },
  );
  const [configText, setConfigText] = React.useState(JSON.stringify(v.triggerConfig ?? {}, null, 0));
  const [configError, setConfigError] = React.useState<string>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof WorkflowRule>(k: K, val: WorkflowRule[K]) => setV((p) => ({ ...p, [k]: val }));
  const patchAction = (i: number, p: Partial<WorkflowAction>) => set("actions", v.actions.map((x, j) => (j === i ? { ...x, ...p } : x)));

  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        let triggerConfig: Record<string, unknown> = {};
        try {
          triggerConfig = configText.trim() ? JSON.parse(configText) : {};
        } catch {
          return setConfigError(a.invalidJson);
        }
        void run(() => saveWorkflowRuleAction({ ...v, triggerConfig }), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${a.name} *`} htmlFor="wf-name" className="sm:col-span-2">
        <Input id="wf-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={a.object} htmlFor="wf-object">
        <SimpleSelect id="wf-object" value={v.object} onChange={(x) => set("object", x)} options={opts(RECORD_OBJECT)} />
      </Fld>
      <Fld label={a.trigger} htmlFor="wf-trigger">
        <SimpleSelect id="wf-trigger" value={v.triggerType} onChange={(x) => set("triggerType", x)} options={opts(TRIGGER)} />
      </Fld>
      <Fld label={a.triggerConfig} htmlFor="wf-config" error={configError ? [configError] : undefined} hint='{"to_stage":"proposal"}' className="sm:col-span-2">
        <Input id="wf-config" value={configText} onChange={(e) => setConfigText(e.target.value)} className="font-mono text-xs" />
      </Fld>

      <fieldset className="flex flex-col gap-2 sm:col-span-2">
        <legend className="mb-1 text-sm font-medium">{a.conditions}</legend>
        {v.conditions.map((c, i) => (
          <div key={i} className="grid grid-cols-[1fr_160px_1fr_auto] gap-2">
            <Input
              aria-label={a.field}
              placeholder={a.field}
              value={c.field}
              onChange={(e) => set("conditions", v.conditions.map((x, j) => (j === i ? { ...x, field: e.target.value } : x)))}
            />
            <SimpleSelect
              value={c.op}
              onChange={(op) => set("conditions", v.conditions.map((x, j) => (j === i ? { ...x, op } : x)))}
              options={opts(CONDITION_OP)}
            />
            <Input
              aria-label={a.value}
              placeholder={a.value}
              defaultValue={showValue(c.value)}
              onBlur={(e) => set("conditions", v.conditions.map((x, j) => (j === i ? { ...x, value: parseValue(e.target.value) } : x)))}
            />
            <Button type="button" size="icon-sm" variant="ghost" aria-label={ss.delete} onClick={() => set("conditions", v.conditions.filter((_, j) => j !== i))}>
              <Trash2 className="size-3.5" />
            </Button>
          </div>
        ))}
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="self-start"
          onClick={() => set("conditions", [...v.conditions, { field: "", op: "eq", value: null }])}
        >
          <Plus className="size-3.5" /> {a.addCondition}
        </Button>
      </fieldset>

      <fieldset className="flex flex-col gap-2 sm:col-span-2">
        <legend className="mb-1 text-sm font-medium">{a.actions}</legend>
        {v.actions.map((act, i) => (
          <div key={i} className="grid gap-2 rounded-lg border p-3 sm:grid-cols-2">
            <div className="flex items-center gap-2 sm:col-span-2">
              <SimpleSelect value={act.type} onChange={(type) => patchAction(i, { type })} options={opts(WORKFLOW_ACTION)} className="w-56" />
              <Button
                type="button"
                size="icon-sm"
                variant="ghost"
                aria-label={ss.delete}
                className="ml-auto"
                onClick={() => set("actions", v.actions.filter((_, j) => j !== i))}
              >
                <Trash2 className="size-3.5" />
              </Button>
            </div>
            <ActionFields action={act} owners={owners} onChange={(p) => patchAction(i, p)} />
          </div>
        ))}
        <Button type="button" size="sm" variant="outline" className="self-start" onClick={() => set("actions", [...v.actions, { ...EMPTY_ACTION }])}>
          <Plus className="size-3.5" /> {a.addAction}
        </Button>
      </fieldset>

      <div className="flex flex-wrap gap-4 text-sm sm:col-span-2">
        <label className="flex items-center gap-2">
          <Checkbox checked={v.runOncePerRecord} onCheckedChange={(c) => set("runOncePerRecord", c === true)} /> {a.runOnce}
        </label>
        <label className="flex items-center gap-2">
          <Switch checked={v.isActive} onCheckedChange={(c) => set("isActive", c)} /> {ss.active}
        </label>
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function ActionFields({ action, owners, onChange }: { action: WorkflowAction; owners: Option[]; onChange: (p: Partial<WorkflowAction>) => void }) {
  switch (action.type) {
    case "create_task":
      return (
        <>
          <Fld label={a.actionTitle} className="sm:col-span-2">
            <Input value={action.title} onChange={(e) => onChange({ title: e.target.value })} placeholder="Hubungi {{lead.pic_name}}" />
          </Fld>
          <Fld label={ss.activity.type}>
            <SimpleSelect value={action.activityType} onChange={(x) => onChange({ activityType: x })} options={opts(ACTIVITY_TYPE)} />
          </Fld>
          <Fld label={a.dueInDays}>
            <Input type="number" min={0} step="any" value={action.dueInDays} onChange={(e) => onChange({ dueInDays: Number(e.target.value) })} />
          </Fld>
          <Fld label={ss.activity.priority}>
            <SimpleSelect value={action.priority || "normal"} onChange={(x) => onChange({ priority: x })} options={opts(PRIORITY)} />
          </Fld>
          <Fld label={a.assignTo}>
            <SimpleSelect
              value={action.assignTo}
              onChange={(x) => onChange({ assignTo: x })}
              options={[{ value: "owner", label: a.assignOwner }, { value: "creator", label: a.assignCreator }, ...owners]}
            />
          </Fld>
          <Fld label={ss.notes} className="sm:col-span-2">
            <Textarea rows={2} value={action.notes} onChange={(e) => onChange({ notes: e.target.value })} />
          </Fld>
        </>
      );
    case "assign_owner":
      return (
        <>
          <Fld label={a.strategy}>
            <SimpleSelect
              value={action.strategy}
              onChange={(x) => onChange({ strategy: x })}
              options={[{ value: "round_robin", label: a.roundRobin }]}
              emptyLabel={a.notifyUser}
            />
          </Fld>
          {action.strategy === "round_robin" ? (
            <Fld label={a.users} className="sm:col-span-2">
              <ChipToggle value={action.userIds} onChange={(x) => onChange({ userIds: x })} options={owners} />
            </Fld>
          ) : (
            <Fld label={a.notifyUser}>
              <Picker value={action.userId} onChange={(x) => onChange({ userId: x })} options={owners} />
            </Fld>
          )}
          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <Checkbox checked={action.onlyIfEmpty} onCheckedChange={(c) => onChange({ onlyIfEmpty: c === true })} /> {a.onlyIfEmpty}
          </label>
        </>
      );
    case "update_field":
      return (
        <>
          <Fld label={a.field}>
            <Input value={action.field} onChange={(e) => onChange({ field: e.target.value })} placeholder="temperature" />
          </Fld>
          <Fld label={a.value}>
            <Input defaultValue={showValue(action.value)} onBlur={(e) => onChange({ value: parseValue(e.target.value) })} />
          </Fld>
        </>
      );
    case "notify_in_app":
      return (
        <>
          <Fld label={a.notifyTo}>
            <SimpleSelect
              value={action.to || "owner"}
              onChange={(x) => onChange({ to: x })}
              options={[
                { value: "owner", label: a.notifyOwner },
                { value: "user", label: a.notifyUser },
                { value: "role", label: a.notifyRole },
              ]}
            />
          </Fld>
          {action.to === "user" ? (
            <Fld label={a.notifyUser}>
              <Picker value={action.userId} onChange={(x) => onChange({ userId: x })} options={owners} />
            </Fld>
          ) : action.to === "role" ? (
            <Fld label={a.role}>
              <Input value={action.role} onChange={(e) => onChange({ role: e.target.value })} placeholder="sales.approve" />
            </Fld>
          ) : (
            <span />
          )}
          <Fld label={a.message} className="sm:col-span-2">
            <Textarea rows={2} value={action.message} onChange={(e) => onChange({ message: e.target.value })} />
          </Fld>
        </>
      );
    case "wait":
      return (
        <>
          <Fld label={a.daysLabel}>
            <Input type="number" min={0} step="any" value={action.days} onChange={(e) => onChange({ days: Number(e.target.value) })} />
          </Fld>
          <Fld label={a.hours}>
            <Input type="number" min={0} step="any" value={action.hours} onChange={(e) => onChange({ hours: Number(e.target.value) })} />
          </Fld>
        </>
      );
    default:
      return null;
  }
}

// Scoring

export function ScoringRules({ rules }: { rules: ScoringRule[] }) {
  const { pending, run } = useErpAction();
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        <ScoringDialog
          rule={null}
          trigger={
            <Button size="sm">
              <Plus className="size-3.5" /> {a.scoringAdd}
            </Button>
          }
        />
        <Button
          size="sm"
          variant="outline"
          disabled={pending}
          onClick={() =>
            void run(recalculateScoresAction, {
              onOk: (r) => toast.success(a.recalcDone(r.leads, r.changed)),
            })
          }
        >
          {pending ? <Loader2 className="size-3.5 animate-spin" /> : <RefreshCw className="size-3.5" />} {a.recalc}
        </Button>
      </div>
      <ul className="divide-y rounded-xl border bg-card">
        {rules.map((r) => (
          <li key={r.id} className="flex items-center gap-3 px-5 py-3 text-sm">
            <span className="w-12 text-right font-semibold tabular">{r.points > 0 ? `+${r.points}` : r.points}</span>
            <span className="min-w-0 flex-1">
              <span className="block font-medium">{r.name}</span>
              <span className="text-xs text-muted-foreground">
                {lbl(SCORING_KIND, r.kind)} ·{" "}
                {r.kind === "event"
                  ? `${lbl(SCORING_EVENT, r.eventType)}${r.maxCount ? ` · maks ${r.maxCount}x` : ""}`
                  : `${r.field} ${lbl(CONDITION_OP, r.operator)} ${showValue(r.value)}`}
                {!r.isActive ? ` · ${ss.inactive}` : ""}
              </span>
            </span>
            <ScoringDialog
              rule={r}
              trigger={
                <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                  <Pencil className="size-3.5" />
                </Button>
              }
            />
            <ConfirmAction
              title={ss.confirmDelete}
              action={() => deleteScoringRuleAction(r.id)}
              trigger={
                <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                  <Trash2 className="size-3.5" />
                </Button>
              }
            />
          </li>
        ))}
      </ul>
    </div>
  );
}

function ScoringDialog({ rule, trigger }: { rule: ScoringRule | null; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={a.scoringTitle}>
      {(close) => <ScoringForm rule={rule} close={close} />}
    </FormDialog>
  );
}

function ScoringForm({ rule, close }: { rule: ScoringRule | null; close: () => void }) {
  const [v, setV] = React.useState<ScoringRule>(
    rule ?? { id: "", name: "", kind: "field", field: "", operator: "eq", value: null, eventType: "", windowDays: 0, maxCount: 1, points: 10, isActive: true },
  );
  const { pending, run } = useErpAction();
  const set = <K extends keyof ScoringRule>(k: K, val: ScoringRule[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveScoringRuleAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${a.name} *`} htmlFor="sr-name" className="sm:col-span-2">
        <Input id="sr-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={a.kind} htmlFor="sr-kind">
        <SimpleSelect id="sr-kind" value={v.kind} onChange={(x) => set("kind", x)} options={opts(SCORING_KIND)} />
      </Fld>
      <Fld label={a.points} htmlFor="sr-points">
        <Input id="sr-points" type="number" value={v.points} onChange={(e) => set("points", Number(e.target.value))} />
      </Fld>
      {v.kind === "event" ? (
        <>
          <Fld label={a.event} htmlFor="sr-event">
            <SimpleSelect id="sr-event" value={v.eventType} onChange={(x) => set("eventType", x)} options={opts(SCORING_EVENT)} />
          </Fld>
          <Fld label={a.value} htmlFor="sr-evalue" hint="meeting">
            <Input id="sr-evalue" defaultValue={showValue(v.value)} onBlur={(e) => set("value", parseValue(e.target.value))} />
          </Fld>
          <Fld label={a.windowDays} htmlFor="sr-window">
            <Input id="sr-window" type="number" min={0} value={v.windowDays} onChange={(e) => set("windowDays", Number(e.target.value))} />
          </Fld>
          <Fld label={a.maxCount} htmlFor="sr-max">
            <Input id="sr-max" type="number" min={0} value={v.maxCount} onChange={(e) => set("maxCount", Number(e.target.value))} />
          </Fld>
        </>
      ) : (
        <>
          <Fld label={a.field} htmlFor="sr-field" hint="temperature, source, pic_email">
            <Input id="sr-field" value={v.field} onChange={(e) => set("field", e.target.value)} />
          </Fld>
          <Fld label={a.op} htmlFor="sr-op">
            <SimpleSelect id="sr-op" value={v.operator} onChange={(x) => set("operator", x)} options={opts(CONDITION_OP)} />
          </Fld>
          <Fld label={a.value} htmlFor="sr-value" className="sm:col-span-2">
            <Input id="sr-value" defaultValue={showValue(v.value)} onBlur={(e) => set("value", parseValue(e.target.value))} />
          </Fld>
        </>
      )}
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <Checkbox checked={v.isActive} onCheckedChange={(c) => set("isActive", c === true)} /> {ss.active}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

// Pipeline stages

export function StagesEditor({ pipelines }: { pipelines: Pipeline[] }) {
  return (
    <div className="flex flex-col gap-4">
      {pipelines.map((p) => (
        <div key={p.id} className="rounded-xl border bg-card">
          <div className="flex items-center justify-between border-b px-5 py-3">
            <h3 className="font-medium">{p.name}</h3>
            <StageDialog
              stage={null}
              pipelineId={p.id}
              nextOrder={p.stages.length + 1}
              trigger={
                <Button size="sm" variant="outline">
                  <Plus className="size-3.5" /> {a.addStage}
                </Button>
              }
            />
          </div>
          <ul className="divide-y">
            {[...p.stages].sort((x, y) => x.sortOrder - y.sortOrder).map((st) => (
              <li key={st.id} className="flex items-center gap-3 px-5 py-2.5 text-sm">
                <span className="w-6 text-xs text-muted-foreground tabular">{st.sortOrder}</span>
                <span className="min-w-0 flex-1">
                  <span className="font-medium">{st.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {" "}
                    · {st.probability}% · {a.stuckDays} {st.stuckThresholdDays}
                    {st.isWon ? ` · ${a.won}` : ""}
                    {st.isLost ? ` · ${a.lost}` : ""}
                    {!st.isActive ? ` · ${ss.inactive}` : ""}
                  </span>
                </span>
                <StageDialog
                  stage={st}
                  pipelineId={p.id}
                  nextOrder={st.sortOrder}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

function StageDialog({ stage, pipelineId, nextOrder, trigger }: { stage: Stage | null; pipelineId: string; nextOrder: number; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={a.stageTitle}>
      {(close) => (
        <StageForm
          stage={
            stage ?? {
              id: "",
              pipelineId,
              code: "",
              name: "",
              sortOrder: nextOrder,
              probability: 50,
              isWon: false,
              isLost: false,
              stuckThresholdDays: 7,
              isActive: true,
            }
          }
          close={close}
        />
      )}
    </FormDialog>
  );
}

function StageForm({ stage, close }: { stage: Stage; close: () => void }) {
  const [v, setV] = React.useState(stage);
  const { pending, run } = useErpAction();
  const set = <K extends keyof Stage>(k: K, val: Stage[K]) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveStageAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${a.stageName} *`} htmlFor="st-name">
        <Input id="st-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={a.code} htmlFor="st-code">
        <Input id="st-code" value={v.code} onChange={(e) => set("code", e.target.value)} />
      </Fld>
      <Fld label={a.order} htmlFor="st-order">
        <Input id="st-order" type="number" value={v.sortOrder} onChange={(e) => set("sortOrder", Number(e.target.value))} />
      </Fld>
      <Fld label={a.probability} htmlFor="st-prob">
        <Input id="st-prob" type="number" min={0} max={100} value={v.probability} onChange={(e) => set("probability", Number(e.target.value))} />
      </Fld>
      <Fld label={a.stuckDays} htmlFor="st-stuck">
        <Input id="st-stuck" type="number" min={0} value={v.stuckThresholdDays} onChange={(e) => set("stuckThresholdDays", Number(e.target.value))} />
      </Fld>
      <div className="flex flex-wrap items-end gap-4 text-sm">
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isWon} onCheckedChange={(c) => setV((p) => ({ ...p, isWon: c === true, isLost: c === true ? false : p.isLost }))} /> {a.won}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isLost} onCheckedChange={(c) => setV((p) => ({ ...p, isLost: c === true, isWon: c === true ? false : p.isWon }))} /> {a.lost}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isActive} onCheckedChange={(c) => set("isActive", c === true)} /> {ss.active}
        </label>
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

// Custom fields

export function CustomFieldsManager({ fields }: { fields: CustomField[] }) {
  return (
    <div className="flex flex-col gap-3">
      <div>
        <FieldDialog
          field={null}
          trigger={
            <Button size="sm">
              <Plus className="size-3.5" /> {a.fieldAdd}
            </Button>
          }
        />
      </div>
      {fields.length === 0 ? (
        <p className="rounded-xl border border-dashed py-8 text-center text-sm text-muted-foreground">{ss.none}</p>
      ) : (
        <ul className="divide-y rounded-xl border bg-card">
          {[...fields].sort((x, y) => x.object.localeCompare(y.object) || x.sortOrder - y.sortOrder).map((f) => (
            <li key={f.id} className="flex items-center gap-3 px-5 py-2.5 text-sm">
              <span className="w-16 text-xs text-muted-foreground">{lbl(RECORD_OBJECT, f.object)}</span>
              <span className="min-w-0 flex-1">
                <span className="font-medium">{f.label}</span>
                <span className="text-xs text-muted-foreground">
                  {" "}
                  · <code className="font-mono">{f.key}</code> · {lbl(CUSTOM_FIELD_TYPE, f.fieldType)}
                  {f.isRequired ? ` · ${ss.forms.required}` : ""}
                  {!f.isActive ? ` · ${ss.inactive}` : ""}
                </span>
              </span>
              <FieldDialog
                field={f}
                trigger={
                  <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                    <Pencil className="size-3.5" />
                  </Button>
                }
              />
              <ConfirmAction
                title={ss.confirmDelete}
                action={() => deleteCustomFieldAction(f.id)}
                trigger={
                  <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                    <Trash2 className="size-3.5" />
                  </Button>
                }
              />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function FieldDialog({ field, trigger }: { field: CustomField | null; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={a.fieldTitle}>
      {(close) => <FieldForm field={field} close={close} />}
    </FormDialog>
  );
}

function FieldForm({ field, close }: { field: CustomField | null; close: () => void }) {
  const [v, setV] = React.useState<CustomField>(
    field ?? { id: "", object: "lead", key: "", label: "", fieldType: "text", options: [], isRequired: false, showInList: false, sortOrder: 0, isActive: true },
  );
  const { pending, run } = useErpAction();
  const set = <K extends keyof CustomField>(k: K, val: CustomField[K]) => setV((p) => ({ ...p, [k]: val }));
  const hasOptions = v.fieldType === "picklist" || v.fieldType === "multipicklist";
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveCustomFieldAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={a.object} htmlFor="cf-object">
        <SimpleSelect
          id="cf-object"
          value={v.object}
          onChange={(x) => set("object", x)}
          options={opts(RECORD_OBJECT).filter((o) => ["lead", "deal", "account", "contact"].includes(o.value))}
          disabled={!!field}
        />
      </Fld>
      <Fld label={a.fieldType} htmlFor="cf-type">
        <SimpleSelect id="cf-type" value={v.fieldType} onChange={(x) => set("fieldType", x)} options={opts(CUSTOM_FIELD_TYPE)} />
      </Fld>
      <Fld label={`${ss.forms.label} *`} htmlFor="cf-label">
        <Input id="cf-label" value={v.label} onChange={(e) => set("label", e.target.value)} />
      </Fld>
      <Fld label={`${a.fieldKey} *`} htmlFor="cf-key">
        <Input
          id="cf-key"
          value={v.key}
          onChange={(e) => set("key", e.target.value.replace(/[^a-z0-9_]/gi, "_").toLowerCase())}
          disabled={!!field}
          className="font-mono text-xs"
        />
      </Fld>
      {hasOptions ? (
        <Fld label={ss.forms.options} htmlFor="cf-options" className="sm:col-span-2">
          <Input
            id="cf-options"
            value={v.options.join(", ")}
            onChange={(e) => set("options", e.target.value.split(",").map((o) => o.trimStart()))}
            onBlur={() => set("options", v.options.map((o) => o.trim()).filter(Boolean))}
          />
        </Fld>
      ) : null}
      <Fld label={a.order} htmlFor="cf-order">
        <Input id="cf-order" type="number" value={v.sortOrder} onChange={(e) => set("sortOrder", Number(e.target.value))} />
      </Fld>
      <div className="flex flex-wrap items-end gap-4 text-sm">
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isRequired} onCheckedChange={(c) => set("isRequired", c === true)} /> {ss.forms.required}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.showInList} onCheckedChange={(c) => set("showInList", c === true)} /> {a.showInList}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={v.isActive} onCheckedChange={(c) => set("isActive", c === true)} /> {ss.active}
        </label>
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
