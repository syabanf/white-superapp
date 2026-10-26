"use client";

import * as React from "react";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { deleteReplyTemplateAction, saveChatbotSettingsAction, saveReplyTemplateAction } from "../actions";
import { ss } from "../strings";
import type { ChatbotSettings, ReplyTemplate } from "../types";
import { ChipToggle, ConfirmAction, Fld, FormDialog, FormFooter, type Option } from "./ui";

const s = ss.chat;
const st = s.settings;
const noop = () => () => {};

/** Bot persona, pricing hints, hours and handoff rules. Editable for ADMIN. */
export function ChatbotSettingsForm({ settings, owners, admin }: { settings: ChatbotSettings; owners: Option[]; admin: boolean }) {
  const [v, setV] = React.useState(settings);
  const { pending, run } = useErpAction();
  const set = <K extends keyof ChatbotSettings>(k: K, val: ChatbotSettings[K]) => setV((p) => ({ ...p, [k]: val }));
  const area = (k: "persona" | "servicesOffered" | "pricingHints" | "handoffRules" | "greeting", label: string, hint?: string, rows = 4) => (
    <Fld label={label} htmlFor={`bot-${k}`} hint={hint} className="sm:col-span-2">
      <Textarea id={`bot-${k}`} rows={rows} value={v[k]} onChange={(e) => set(k, e.target.value)} disabled={!admin} />
    </Fld>
  );
  const origin = React.useSyncExternalStore(
    noop,
    () => window.location.origin,
    () => "",
  );
  const snippet = `<iframe src="${origin}/chat" title="Chat" style="position:fixed;right:16px;bottom:16px;width:380px;height:560px;max-width:calc(100vw - 32px);border:0;border-radius:16px;box-shadow:0 12px 40px rgba(0,0,0,.18);z-index:2147483000"></iframe>`;

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <Card className="px-5 py-5 lg:col-span-2">
        <form
          className="grid gap-4 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => saveChatbotSettingsAction(v), { success: st.saved });
          }}
        >
          <label className="flex items-center gap-2 text-sm font-medium sm:col-span-2">
            <Switch checked={v.enabled} onCheckedChange={(c) => set("enabled", c)} disabled={!admin} /> {st.enabled}
          </label>
          <Fld label={st.botName} htmlFor="bot-name">
            <Input id="bot-name" value={v.botName} onChange={(e) => set("botName", e.target.value)} disabled={!admin} />
          </Fld>
          <Fld label={st.minConfidence} htmlFor="bot-conf">
            <Input
              id="bot-conf"
              type="number"
              min={0}
              max={1}
              step={0.05}
              value={v.minConfidence}
              onChange={(e) => set("minConfidence", Number(e.target.value))}
              disabled={!admin}
            />
          </Fld>
          <Fld label={st.hoursStart} htmlFor="bot-start">
            <Input id="bot-start" type="number" min={0} max={23} value={v.businessHoursStart} onChange={(e) => set("businessHoursStart", Number(e.target.value))} disabled={!admin} />
          </Fld>
          <Fld label={st.hoursEnd} htmlFor="bot-end">
            <Input id="bot-end" type="number" min={0} max={24} value={v.businessHoursEnd} onChange={(e) => set("businessHoursEnd", Number(e.target.value))} disabled={!admin} />
          </Fld>
          {area("greeting", st.greeting, undefined, 2)}
          {area("persona", st.persona, undefined, 3)}
          {area("servicesOffered", st.services)}
          {area("pricingHints", st.pricing, st.pricingHint)}
          {area("handoffRules", st.handoffRules)}
          <Fld label={st.handoffUsers} className="sm:col-span-2">
            {admin ? (
              <ChipToggle value={v.handoffUserIds} onChange={(x) => set("handoffUserIds", x)} options={owners} />
            ) : (
              <p className="text-sm">{owners.filter((o) => v.handoffUserIds.includes(o.value)).map((o) => o.label).join(", ") || ss.dash}</p>
            )}
          </Fld>
          {admin ? (
            <div className="flex justify-end sm:col-span-2">
              <Button type="submit" disabled={pending}>
                {pending ? <Loader2 className="size-4 animate-spin" /> : null}
                {ss.save}
              </Button>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground sm:col-span-2">{ss.adminOnly}</p>
          )}
        </form>
      </Card>
      <div className="flex flex-col gap-4">
        <Card className="gap-2 px-5 py-4">
          <h3 className="text-sm font-semibold">{st.whatsapp}</h3>
          <p className={v.whatsappEnabled ? "text-sm text-positive" : "text-sm text-muted-foreground"}>
            {v.whatsappEnabled ? st.whatsappOn : st.whatsappOff}
          </p>
        </Card>
        <Card className="gap-2 px-5 py-4">
          <h3 className="text-sm font-semibold">{st.embed}</h3>
          <p className="text-xs text-muted-foreground">{st.embedHint}</p>
          <Textarea readOnly rows={6} value={snippet} className="font-mono text-[11px]" onFocus={(e) => e.currentTarget.select()} aria-label={st.embed} />
        </Card>
      </div>
    </div>
  );
}

/** Quick reply templates for agents. */
export function ReplyTemplates({ templates, canWrite }: { templates: ReplyTemplate[]; canWrite: boolean }) {
  return (
    <div className="flex flex-col gap-3">
      {canWrite ? (
        <div>
          <TemplateDialog
            template={null}
            trigger={
              <Button size="sm">
                <Plus className="size-3.5" /> {s.tpl.add}
              </Button>
            }
          />
        </div>
      ) : null}
      <ul className="divide-y rounded-xl border bg-card">
        {templates.map((t) => (
          <li key={t.id} className="flex items-start gap-3 px-5 py-3">
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">
                {t.title}
                {!t.isActive ? <span className="ml-2 text-xs text-muted-foreground">{ss.inactive}</span> : null}
              </p>
              <p className="text-sm whitespace-pre-line text-muted-foreground">{t.body}</p>
            </div>
            {canWrite ? (
              <div className="flex shrink-0">
                <TemplateDialog
                  template={t}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.edit}>
                      <Pencil className="size-3.5" />
                    </Button>
                  }
                />
                <ConfirmAction
                  title={s.tpl.deleteTitle}
                  action={() => deleteReplyTemplateAction(t.id)}
                  trigger={
                    <Button size="icon-sm" variant="ghost" aria-label={ss.delete}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </div>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function TemplateDialog({ template, trigger }: { template: ReplyTemplate | null; trigger: React.ReactNode }) {
  return (
    <FormDialog trigger={trigger} title={s.tpl.title}>
      {(close) => <TemplateForm template={template} close={close} />}
    </FormDialog>
  );
}

function TemplateForm({ template, close }: { template: ReplyTemplate | null; close: () => void }) {
  const [v, setV] = React.useState<ReplyTemplate>(template ?? { id: "", title: "", body: "", isActive: true });
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveReplyTemplateAction(v), { success: ss.saved, onOk: close });
      }}
    >
      <Fld label={`${s.tpl.name} *`} htmlFor="rt-title">
        <Input id="rt-title" value={v.title} onChange={(e) => setV((p) => ({ ...p, title: e.target.value }))} />
      </Fld>
      <Fld label={`${s.tpl.body} *`} htmlFor="rt-body">
        <Textarea id="rt-body" rows={4} value={v.body} onChange={(e) => setV((p) => ({ ...p, body: e.target.value }))} />
      </Fld>
      <label className="flex items-center gap-2 text-sm">
        <Checkbox checked={v.isActive} onCheckedChange={(x) => setV((p) => ({ ...p, isActive: x === true }))} /> {ss.active}
      </label>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}
