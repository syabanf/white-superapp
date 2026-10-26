"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { AlertTriangle, Bot, Globe, Loader2, MessageCircle, Send, Sparkles, UserRound } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { formatDateTime, formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";
import { addConversationNoteAction, analyzeConversationAction, conversationAction, replyConversationAction } from "../actions";
import { nameOf } from "../lib";
import { CHANNEL, CONVERSATION_STATUS, SENDER, SENTIMENT, lbl, opts, ss } from "../strings";
import type { Conversation, ConversationDetail, ReplyTemplate } from "../types";
import { ActionButton, type Option, SalesBadge, SimpleSelect } from "./ui";

const s = ss.chat;
const POLL_MS = 10_000;

/** Conversation list; the selected one is the `c` URL param. */
export function ConversationList({ conversations, selected }: { conversations: Conversation[]; selected?: string }) {
  const sp = useSearchParams();
  const href = (id: string) => {
    const next = new URLSearchParams(sp.toString());
    next.set("c", id);
    return `?${next.toString()}`;
  };
  if (!conversations.length) return <p className="px-4 py-10 text-center text-sm text-muted-foreground">{s.empty}</p>;
  return (
    <ul className="divide-y">
      {conversations.map((c) => {
        const Icon = c.channel === "whatsapp" ? MessageCircle : Globe;
        return (
          <li key={c.id}>
            <Link
              href={href(c.id)}
              scroll={false}
              className={cn("flex gap-3 px-4 py-3 hover:bg-accent/50", selected === c.id && "bg-accent")}
            >
              <Icon className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-label={lbl(CHANNEL, c.channel)} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <span className={cn("truncate text-sm", c.unreadCount ? "font-semibold" : "font-medium")}>
                    {c.displayName || c.phone || c.email || c.externalId}
                  </span>
                  <span className="shrink-0 text-[11px] text-muted-foreground">{formatRelative(c.lastMessageAt)}</span>
                </div>
                <p className="truncate text-xs text-muted-foreground">{c.lastMessagePreview}</p>
                <div className="mt-1 flex flex-wrap items-center gap-1.5">
                  {c.botActive ? (
                    <span className="inline-flex items-center gap-1 text-[11px] text-muted-foreground">
                      <Bot className="size-3" /> {s.botOn}
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 text-[11px] text-brand">
                      <UserRound className="size-3" /> {s.botOff}
                    </span>
                  )}
                  {c.isComplaint ? <span className="text-[11px] text-destructive">{s.complaint}</span> : null}
                  {c.slaResponseBreached ? <AlertTriangle className="size-3 text-amber-500" aria-label={s.sla} /> : null}
                  {c.unreadCount ? (
                    <Badge className="ml-auto h-4 min-w-4 rounded-full px-1 text-[10px]" aria-label={s.unread(c.unreadCount)}>
                      {c.unreadCount}
                    </Badge>
                  ) : null}
                </div>
              </div>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}

/** Thread, reply box, handoff controls, notes and AI insight for one conversation. */
export function ConversationPanel({
  detail,
  templates,
  owners,
  canWrite,
  userId,
}: {
  detail: ConversationDetail;
  templates: ReplyTemplate[];
  owners: Option[];
  canWrite: boolean;
  userId?: string;
}) {
  const router = useRouter();
  const c = detail.conversation;
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  const bottom = React.useRef<HTMLDivElement>(null);
  const [reply, setReply] = React.useState("");
  const [note, setNote] = React.useState("");
  const { pending, run } = useErpAction();

  // Live-ish thread: refetch the server page while the panel is open.
  React.useEffect(() => {
    const t = setInterval(() => {
      if (document.visibilityState === "visible") router.refresh();
    }, POLL_MS);
    return () => clearInterval(t);
  }, [router]);

  React.useEffect(() => {
    bottom.current?.scrollIntoView({ block: "end" });
  }, [detail.messages.length]);

  React.useEffect(() => {
    if (canWrite && c.unreadCount > 0) void conversationAction(c.id, "mark_read");
  }, [c.id, c.unreadCount, canWrite]);

  const act = (action: string, extra: { status?: string; value?: boolean } = {}) => () => conversationAction(c.id, action, extra);
  const insight = detail.insight;

  return (
    <div className="grid min-h-0 gap-4 xl:grid-cols-[1fr_300px]">
      <Card className="flex min-h-[560px] flex-col gap-0 overflow-hidden py-0">
        <header className="flex flex-wrap items-center gap-2 border-b px-4 py-3">
          <div className="min-w-0 flex-1">
            <h2 className="truncate font-semibold">{c.displayName || c.phone || c.email}</h2>
            <p className="text-xs text-muted-foreground">
              {lbl(CHANNEL, c.channel)}
              {c.phone ? ` · ${c.phone}` : ""}
              {c.email ? ` · ${c.email}` : ""}
              {c.assignedUserId ? ` · ${nameOf(names, c.assignedUserId)}` : ""}
            </p>
          </div>
          {canWrite ? (
            <div className="flex flex-wrap items-center gap-1.5">
              {c.botActive ? (
                <ActionButton size="xs" action={act("takeover")} success={s.actionDone}>
                  <UserRound className="size-3.5" /> {s.takeover}
                </ActionButton>
              ) : (
                <ActionButton size="xs" variant="outline" action={act("release")} success={s.actionDone}>
                  <Bot className="size-3.5" /> {s.release}
                </ActionButton>
              )}
              {c.assignedUserId === userId ? (
                <ActionButton size="xs" variant="ghost" action={act("unassign")} success={s.actionDone}>
                  {s.unassign}
                </ActionButton>
              ) : (
                <ActionButton size="xs" variant="outline" action={act("assign_me")} success={s.actionDone}>
                  {s.assignMe}
                </ActionButton>
              )}
              <div className="w-44">
                <SimpleSelect
                  size="sm"
                  value={c.status}
                  onChange={(v) => void run(act("set_status", { status: v }), { success: s.actionDone })}
                  options={opts(CONVERSATION_STATUS)}
                />
              </div>
            </div>
          ) : (
            <SalesBadge value={c.status} map={CONVERSATION_STATUS} />
          )}
        </header>
        {c.handoffReason ? (
          <p className="border-b bg-amber-500/10 px-4 py-2 text-xs text-amber-800 dark:text-amber-300">{s.handoff(c.handoffReason)}</p>
        ) : null}
        <div className="flex-1 space-y-2 overflow-y-auto bg-muted/20 px-4 py-4" aria-live="polite">
          {detail.messages.map((m) => {
            const mine = m.direction === "out";
            const system = m.sender === "system";
            if (system)
              return (
                <p key={m.id} className="text-center text-[11px] text-muted-foreground">
                  {m.body}
                </p>
              );
            return (
              <div key={m.id} className={cn("flex", mine ? "justify-end" : "justify-start")}>
                <div
                  className={cn(
                    "max-w-[78%] rounded-2xl px-3 py-2 text-sm",
                    mine ? (m.sender === "bot" ? "bg-card border" : "bg-brand text-white") : "bg-card border",
                  )}
                >
                  <p className="whitespace-pre-line break-words">{m.body}</p>
                  <p className={cn("mt-1 text-[10px]", mine && m.sender !== "bot" ? "text-white/70" : "text-muted-foreground")}>
                    {lbl(SENDER, m.sender)}
                    {m.sentByUserId ? ` · ${nameOf(names, m.sentByUserId)}` : ""} · {formatDateTime(m.createdAt)}
                    {m.status === "failed" ? ` · ${s.failed}${m.errorReason ? `: ${m.errorReason}` : ""}` : ""}
                  </p>
                </div>
              </div>
            );
          })}
          <div ref={bottom} />
        </div>
        {canWrite ? (
          <form
            className="flex flex-col gap-2 border-t p-3"
            onSubmit={(e) => {
              e.preventDefault();
              if (!reply.trim()) return;
              void run(() => replyConversationAction(c.id, reply), { success: s.replied, onOk: () => setReply("") });
            }}
          >
            <Textarea
              value={reply}
              onChange={(e) => setReply(e.target.value)}
              rows={2}
              placeholder={s.replyPlaceholder}
              aria-label={s.reply}
              onKeyDown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) e.currentTarget.form?.requestSubmit();
              }}
            />
            <div className="flex items-center justify-between gap-2">
              <div className="w-56">
                <SimpleSelect
                  size="sm"
                  value=""
                  onChange={(id) => {
                    const t = templates.find((x) => x.id === id);
                    if (t) setReply((r) => (r ? `${r}\n${t.body}` : t.body));
                  }}
                  options={templates.filter((t) => t.isActive).map((t) => ({ value: t.id, label: t.title }))}
                  placeholder={s.template}
                />
              </div>
              <Button type="submit" size="sm" disabled={pending || !reply.trim()}>
                {pending ? <Loader2 className="size-3.5 animate-spin" /> : <Send className="size-3.5" />}
                {s.reply}
              </Button>
            </div>
          </form>
        ) : null}
      </Card>

      <div className="flex flex-col gap-4">
        <Card className="gap-3 px-4 py-4">
          <div className="flex items-center justify-between">
            <h3 className="flex items-center gap-1.5 text-sm font-semibold">
              <Sparkles className="size-3.5" /> {s.insight}
            </h3>
            {canWrite ? (
              <ActionButton size="xs" variant="outline" action={() => analyzeConversationAction(c.id)} success={s.analyzed}>
                {s.analyze}
              </ActionButton>
            ) : null}
          </div>
          {insight && insight.status !== "empty" && insight.summary ? (
            <div className="space-y-2 text-sm">
              <p>{insight.summary}</p>
              <dl className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <dt className="text-muted-foreground">{s.topic}</dt>
                  <dd>{insight.topic || ss.dash}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">{s.sentiment}</dt>
                  <dd>
                    <SalesBadge value={insight.sentiment} map={SENTIMENT} />
                  </dd>
                </div>
              </dl>
              {insight.keywords.length ? (
                <div className="flex flex-wrap gap-1">
                  {insight.keywords.map((k) => (
                    <span key={k} className="rounded-full bg-muted px-2 py-0.5 text-[11px]">
                      {k}
                    </span>
                  ))}
                </div>
              ) : null}
              {insight.isComplaint ? <p className="text-xs text-destructive">{s.complaint}</p> : null}
              <p className="text-[11px] text-muted-foreground">
                {insight.model} · {formatDateTime(insight.analyzedAt)}
              </p>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">{s.noInsight}</p>
          )}
        </Card>

        <Card className="gap-2 px-4 py-4 text-sm">
          {c.leadId ? (
            <Link href={`/erp/sales/leads/${c.leadId}`} className="font-medium hover:underline">
              {s.lead} →
            </Link>
          ) : null}
          {canWrite ? (
            <ActionButton
              size="xs"
              variant={c.isComplaint ? "outline" : "ghost"}
              action={act("set_complaint", { value: !c.isComplaint })}
              success={s.actionDone}
              className="self-start"
            >
              <AlertTriangle className="size-3.5" /> {c.isComplaint ? s.unmarkComplaint : s.markComplaint}
            </ActionButton>
          ) : null}
          {c.slaResponseBreached ? <p className="text-xs text-amber-600">{s.sla}</p> : null}
        </Card>

        <Card className="gap-3 px-4 py-4">
          <h3 className="text-sm font-semibold">{s.notes}</h3>
          {detail.notes.length ? (
            <ul className="space-y-2">
              {detail.notes.map((n) => (
                <li key={n.id} className="rounded-lg bg-muted/50 px-3 py-2 text-sm">
                  <p className="whitespace-pre-line">{n.body}</p>
                  <p className="mt-1 text-[11px] text-muted-foreground">
                    {nameOf(names, n.authorUserId)} · {formatDateTime(n.createdAt)}
                  </p>
                </li>
              ))}
            </ul>
          ) : null}
          {canWrite ? (
            <form
              className="flex flex-col gap-2"
              onSubmit={(e) => {
                e.preventDefault();
                if (!note.trim()) return;
                void run(() => addConversationNoteAction(c.id, note), { success: s.noteSaved, onOk: () => setNote("") });
              }}
            >
              <Textarea value={note} onChange={(e) => setNote(e.target.value)} rows={2} placeholder={s.notePlaceholder} aria-label={s.addNote} />
              <Button type="submit" size="xs" variant="outline" className="self-end" disabled={!note.trim() || pending}>
                {s.addNote}
              </Button>
            </form>
          ) : null}
        </Card>
      </div>
    </div>
  );
}
