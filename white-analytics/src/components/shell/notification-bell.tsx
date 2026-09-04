"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Bell, CheckCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { markAllNotificationsRead, markNotificationRead } from "@/features/notifications/actions";
import type { NotificationFeed } from "@/features/notifications/queries";
import { formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";
import { t } from "@/i18n/id";

/**
 * In-app notifications (approvals, publish results, rank alerts). The feed is
 * fetched by the app layout on every navigation — no polling; opening the
 * sheet is the "refresh". Unread rows carry a brand dot; the count sits on the bell.
 */
export function NotificationBell({ feed }: { feed: NotificationFeed }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const [pending, startTransition] = React.useTransition();
  const n = t.notifications;

  const openItem = (id: string, read: boolean) => {
    setOpen(false);
    if (read) return;
    startTransition(async () => {
      await markNotificationRead(id);
      router.refresh();
    });
  };

  const readAll = () =>
    startTransition(async () => {
      await markAllNotificationsRead();
      router.refresh();
    });

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon-sm" aria-label={n.aria(feed.unread)} className="relative">
              <Bell className="size-4" />
              {feed.unread > 0 ? (
                <span className="absolute top-1 right-1 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-brand px-0.5 font-mono text-[9px] leading-none font-semibold text-primary-foreground">
                  {feed.unread > 99 ? "99+" : feed.unread}
                </span>
              ) : null}
            </Button>
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent>{n.title}</TooltipContent>
      </Tooltip>

      <SheetContent className="w-full gap-0 overflow-y-auto scrollbar-thin sm:max-w-md">
        <SheetHeader className="border-b">
          <p className="label-mono text-brand">{n.eyebrow}</p>
          <SheetTitle className="text-lg font-bold tracking-[-0.025em]">{n.title}</SheetTitle>
          <SheetDescription>{feed.unread > 0 ? n.unread(feed.unread) : n.allRead}</SheetDescription>
        </SheetHeader>

        {feed.items.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 px-6 py-16 text-center">
            <Bell className="size-6 text-muted-foreground" />
            <p className="text-sm font-medium">{n.emptyTitle}</p>
            <p className="max-w-xs text-xs text-muted-foreground">{n.emptyDesc}</p>
          </div>
        ) : (
          <ul className="divide-y">
            {feed.items.map((item) => {
              const inner = (
                <>
                  <span
                    aria-hidden
                    className={cn(
                      "mt-1.5 size-1.5 shrink-0 rounded-full transition-transform",
                      item.read ? "scale-50 bg-muted-foreground/30" : "bg-brand",
                    )}
                  />
                  <span className="min-w-0 flex-1">
                    <span className={cn("block text-sm leading-snug", item.read ? "text-foreground" : "font-medium text-brand-ink")}>
                      {item.title}
                    </span>
                    {item.body ? <span className="mt-0.5 block text-xs leading-relaxed text-muted-foreground">{item.body}</span> : null}
                    <span className="label-mono mt-1.5 block text-muted-foreground">{formatRelative(item.createdAt)}</span>
                  </span>
                </>
              );
              const cls = "flex w-full items-start gap-3 px-4 py-3.5 text-left transition-colors hover:bg-muted/50";
              return (
                <li key={item.id}>
                  {item.href ? (
                    <Link href={item.href} className={cls} onClick={() => openItem(item.id, item.read)}>
                      {inner}
                    </Link>
                  ) : (
                    <button type="button" className={cls} onClick={() => openItem(item.id, item.read)} disabled={pending}>
                      {inner}
                    </button>
                  )}
                </li>
              );
            })}
          </ul>
        )}

        {feed.unread > 0 ? (
          <div className="mt-auto border-t px-4 py-4">
            <Button variant="outline" className="w-full" onClick={readAll} disabled={pending}>
              <CheckCheck className="size-4" /> {n.markAllRead}
            </Button>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}
