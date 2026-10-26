"use client";

import * as React from "react";
import { portalPost } from "./client";

/**
 * Records proctor flags of a running session through the API: tab switch,
 * paste, leaving fullscreen (only after the page entered it) and a dropped
 * connection with its length. Fire and forget: a failed post never blocks
 * the candidate.
 */
export function useProctoring(path: string, active: boolean) {
  React.useEffect(() => {
    if (!active) return;
    let offlineSince: number | null = null;
    const send = (eventType: string, meta?: Record<string, string>) =>
      void portalPost(path, { eventType, meta }).catch(() => undefined);

    const onVisibility = () => {
      if (document.hidden) send("tab_blur");
    };
    const onPaste = () => send("paste");
    const onFullscreen = () => {
      if (!document.fullscreenElement) send("fullscreen_exit");
    };
    const onOffline = () => {
      offlineSince = Date.now();
    };
    const onOnline = () => {
      const since = offlineSince;
      offlineSince = null;
      send("disconnect", since ? { offline_seconds: String(Math.round((Date.now() - since) / 1000)) } : undefined);
    };

    document.addEventListener("visibilitychange", onVisibility);
    document.addEventListener("paste", onPaste);
    document.addEventListener("fullscreenchange", onFullscreen);
    window.addEventListener("offline", onOffline);
    window.addEventListener("online", onOnline);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      document.removeEventListener("paste", onPaste);
      document.removeEventListener("fullscreenchange", onFullscreen);
      window.removeEventListener("offline", onOffline);
      window.removeEventListener("online", onOnline);
    };
  }, [path, active]);
}

/** Asks for fullscreen on desktop pointers; must run inside a click handler. */
export function requestFullscreen() {
  if (typeof window === "undefined" || !window.matchMedia("(pointer: fine)").matches) return;
  const el = document.documentElement;
  if (!document.fullscreenElement && el.requestFullscreen) el.requestFullscreen().catch(() => undefined);
}

/** Seconds left until `endsAt` (null without a deadline); `onExpire` fires once at zero. */
export function useCountdown(endsAt: string | null, onExpire: () => void): number | null {
  const [now, setNow] = React.useState(() => Date.now());
  const expire = React.useEffectEvent(onExpire);
  React.useEffect(() => {
    if (!endsAt) return;
    const end = new Date(endsAt).getTime();
    let fired = false;
    const timer = setInterval(() => {
      const t = Date.now();
      setNow(t);
      if (!fired && t >= end) {
        fired = true;
        expire();
      }
    }, 1000);
    return () => clearInterval(timer);
  }, [endsAt]);
  if (!endsAt) return null;
  return Math.max(0, Math.floor((new Date(endsAt).getTime() - now) / 1000));
}
