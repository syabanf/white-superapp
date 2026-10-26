"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { Loader2, MessageCircle, RotateCcw, Send, UserRound, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { ps } from "../strings";
import type { ChatMessage, PublicChat } from "../types";
import { PublicError, publicClientFetch } from "./client";

const c = ps.chat;
const TOKEN_KEY = "white_chat_token";
const POLL_MS = 4000;
const BASE = "/v1/public/sales/chat";

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}
function writeToken(token: string | null) {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token);
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    // storage blocked (private mode, third-party iframe): the chat still works for this page view
  }
}

function merge(list: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  const byId = new Map(list.map((m) => [m.id, m]));
  for (const m of incoming) byId.set(m.id, m);
  return [...byId.values()].sort((a, b) => (a.createdAt ?? "").localeCompare(b.createdAt ?? ""));
}

/** Website chat with the sales bot; a human agent can take over at any time. */
export function ChatWidget({ className, onClose }: { className?: string; onClose?: () => void }) {
  const [token, setToken] = React.useState<string | null>(null);
  const [botName, setBotName] = React.useState("");
  const [messages, setMessages] = React.useState<ChatMessage[]>([]);
  const [human, setHuman] = React.useState(false);
  const [text, setText] = React.useState("");
  const [error, setError] = React.useState<string | null>(null);
  const [pending, setPending] = React.useState(false);
  const bottom = React.useRef<HTMLDivElement>(null);
  const last = messages.at(-1)?.createdAt;

  const apply = React.useCallback((chat: PublicChat) => {
    setBotName(chat.botName);
    setHuman(chat.humanTakeover);
    setMessages((list) => merge(list, chat.messages));
  }, []);

  const reset = React.useCallback(() => {
    writeToken(null);
    setToken(null);
    setMessages([]);
  }, []);

  // Restore a conversation saved in this browser.
  React.useEffect(() => {
    const saved = readToken();
    if (!saved) return;
    publicClientFetch<PublicChat>(`${BASE}/${encodeURIComponent(saved)}/messages`)
      .then((chat) => {
        setToken(saved);
        apply(chat);
      })
      .catch(() => writeToken(null));
  }, [apply]);

  // Poll for bot and agent replies.
  React.useEffect(() => {
    if (!token) return;
    const t = setInterval(() => {
      if (document.visibilityState !== "visible") return;
      const q = last ? `?after=${encodeURIComponent(last)}` : "";
      publicClientFetch<PublicChat>(`${BASE}/${encodeURIComponent(token)}/messages${q}`)
        .then(apply)
        .catch((e) => {
          if (e instanceof PublicError && e.status === 404) reset();
        });
    }, POLL_MS);
    return () => clearInterval(t);
  }, [token, last, apply, reset]);

  React.useEffect(() => {
    bottom.current?.scrollIntoView({ block: "end" });
  }, [messages.length]);

  const start = async (form: FormData) => {
    setError(null);
    const name = String(form.get("name") ?? "").trim();
    if (!name) return setError(`${c.name}: ${ps.required}`);
    setPending(true);
    try {
      const url = new URL(window.location.href);
      const top = document.referrer && window.top !== window.self ? document.referrer : url.toString();
      const chat = await publicClientFetch<PublicChat>(BASE, {
        method: "POST",
        body: JSON.stringify({
          name,
          email: String(form.get("email") ?? "").trim(),
          phone: String(form.get("phone") ?? "").trim(),
          pageUrl: top,
          attribution: {
            utmSource: url.searchParams.get("utm_source") ?? "",
            utmMedium: url.searchParams.get("utm_medium") ?? "",
            utmCampaign: url.searchParams.get("utm_campaign") ?? "",
            utmTerm: url.searchParams.get("utm_term") ?? "",
            utmContent: url.searchParams.get("utm_content") ?? "",
            landingPage: top,
            referrer: document.referrer,
          },
        }),
      });
      writeToken(chat.token);
      setToken(chat.token);
      apply(chat);
    } catch (err) {
      setError(err instanceof Error ? err.message : ps.offline);
    } finally {
      setPending(false);
    }
  };

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    const body = text.trim();
    if (!body || !token) return;
    setPending(true);
    setError(null);
    try {
      const chat = await publicClientFetch<PublicChat>(`${BASE}/${encodeURIComponent(token)}/messages`, {
        method: "POST",
        body: JSON.stringify({ body }),
      });
      setText("");
      apply(chat);
    } catch (err) {
      setError(err instanceof Error ? err.message : c.failed);
    } finally {
      setPending(false);
    }
  };

  return (
    <section className={cn("flex h-full min-h-0 flex-col overflow-hidden bg-card", className)} aria-label={c.title}>
      <header className="flex items-center gap-3 bg-brand px-4 py-3 text-white">
        <span className="flex size-8 items-center justify-center rounded-full bg-white/20">
          <MessageCircle className="size-4" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold">{botName || c.title}</p>
          {human ? (
            <p className="flex items-center gap-1 text-[11px] text-white/80">
              <UserRound className="size-3" /> {c.human}
            </p>
          ) : null}
        </div>
        {token ? (
          <Button size="icon-sm" variant="ghost" className="text-white hover:bg-white/15 hover:text-white" onClick={reset} aria-label={c.restart}>
            <RotateCcw className="size-4" />
          </Button>
        ) : null}
        {onClose ? (
          <Button size="icon-sm" variant="ghost" className="text-white hover:bg-white/15 hover:text-white" onClick={onClose} aria-label={c.close}>
            <X className="size-4" />
          </Button>
        ) : null}
      </header>

      {!token ? (
        <form
          className="flex flex-1 flex-col gap-3 overflow-y-auto p-4"
          action={(fd) => {
            void start(fd);
          }}
        >
          <p className="text-sm text-muted-foreground">{c.intro}</p>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="chat-name">{c.name} *</Label>
            <Input id="chat-name" name="name" autoComplete="name" required />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="chat-email">{c.email}</Label>
            <Input id="chat-email" name="email" type="email" autoComplete="email" />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="chat-phone">{c.phone}</Label>
            <Input id="chat-phone" name="phone" type="tel" autoComplete="tel" />
          </div>
          {error ? (
            <p className="text-xs text-destructive" role="alert">
              {error}
            </p>
          ) : null}
          <Button type="submit" disabled={pending} className="mt-auto">
            {pending ? <Loader2 className="size-4 animate-spin" /> : null}
            {c.start}
          </Button>
        </form>
      ) : (
        <>
          <div className="flex-1 space-y-2 overflow-y-auto bg-muted/30 px-3 py-4" aria-live="polite">
            {messages.map((m) => {
              if (m.sender === "system")
                return (
                  <p key={m.id} className="text-center text-[11px] text-muted-foreground">
                    {m.body}
                  </p>
                );
              const mine = m.direction === "in";
              return (
                <div key={m.id} className={cn("flex", mine ? "justify-end" : "justify-start")}>
                  <p
                    className={cn(
                      "max-w-[82%] rounded-2xl px-3 py-2 text-sm whitespace-pre-line break-words",
                      mine ? "rounded-br-sm bg-brand text-white" : "rounded-bl-sm border bg-card",
                    )}
                  >
                    {m.body}
                  </p>
                </div>
              );
            })}
            <div ref={bottom} />
          </div>
          <form className="flex items-end gap-2 border-t p-3" onSubmit={send}>
            <Input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={c.placeholder}
              aria-label={c.placeholder}
              maxLength={2000}
              autoComplete="off"
            />
            <Button type="submit" size="icon" disabled={pending || !text.trim()} aria-label={c.send}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
            </Button>
          </form>
          {error ? (
            <p className="px-3 pb-2 text-xs text-destructive" role="alert">
              {error}
            </p>
          ) : null}
        </>
      )}
    </section>
  );
}

const noopSubscribe = () => () => {};

/**
 * Floating chat button that opens the widget in a panel (same-origin pages).
 * Portaled to <body> so animated page wrappers cannot trap the fixed position.
 */
export function ChatBubble() {
  const [open, setOpen] = React.useState(false);
  const mounted = React.useSyncExternalStore(noopSubscribe, () => true, () => false);
  if (!mounted) return null;
  return createPortal(
    <div className="fixed right-4 bottom-4 z-50 flex flex-col items-end gap-3">
      {open ? (
        <div className="h-[min(560px,calc(100dvh-6rem))] w-[min(380px,calc(100vw-2rem))] overflow-hidden rounded-2xl border shadow-2xl">
          <ChatWidget onClose={() => setOpen(false)} />
        </div>
      ) : null}
      <Button
        size="icon-lg"
        className="size-14 rounded-full shadow-lg"
        onClick={() => setOpen((o) => !o)}
        aria-label={open ? c.close : c.open}
        aria-expanded={open}
      >
        {open ? <X className="size-5" /> : <MessageCircle className="size-5" />}
      </Button>
    </div>,
    document.body,
  );
}
