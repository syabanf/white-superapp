import type { ReactNode } from "react";
import Link from "next/link";
import { Bot, Globe, MessageCircle } from "lucide-react";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { Card } from "@/components/ui/card";
import { Money } from "@/features/erp/shared/money-display";
import { formatDate, formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";
import { CHANNEL, CONVERSATION_STATUS, lbl, ss } from "../strings";
import type { Conversation, Deal } from "../types";

/** Titled card section for detail pages. */
export function Panel({
  title,
  actions,
  children,
  className,
  flush,
}: {
  title: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  /** no inner padding (lists that draw their own rows) */
  flush?: boolean;
}) {
  return (
    <Card className={cn("gap-0 py-0", className)}>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b px-5 py-3">
        <h2 className="min-w-0 text-[15px] font-semibold tracking-[-0.015em]">{title}</h2>
        {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
      </div>
      <div className={flush ? undefined : "px-5 py-4"}>{children}</div>
    </Card>
  );
}

/** Deal status badge from the won/lost flags. */
export function DealStatus({ deal }: { deal: Pick<Deal, "isWon" | "isLost" | "stageName"> }) {
  if (deal.isWon) return <StatusBadge kind="good">{ss.pipeline.status.won}</StatusBadge>;
  if (deal.isLost) return <StatusBadge kind="critical">{ss.pipeline.status.lost}</StatusBadge>;
  return <StatusBadge kind="info">{deal.stageName}</StatusBadge>;
}

export function DealRows({ deals, empty }: { deals: Deal[]; empty: string }) {
  if (!deals.length) return <p className="px-5 py-4 text-sm text-muted-foreground">{empty}</p>;
  return (
    <ul className="divide-y">
      {deals.map((d) => (
        <li key={d.id}>
          <Link href={`/erp/sales/pipeline/${d.id}`} className="flex items-center gap-3 px-5 py-3 text-sm hover:bg-accent/50">
            <span className="min-w-0 flex-1">
              <span className="block truncate font-medium">{d.title}</span>
              <span className="text-xs text-muted-foreground">{formatDate(d.createdAt)}</span>
            </span>
            <DealStatus deal={d} />
            <Money value={d.valueFinal || d.valueEstimate} className="w-32 text-right text-sm font-medium" />
          </Link>
        </li>
      ))}
    </ul>
  );
}

export function ConversationRows({ conversations, empty }: { conversations: Conversation[]; empty: string }) {
  if (!conversations.length) return <p className="px-5 py-4 text-sm text-muted-foreground">{empty}</p>;
  return (
    <ul className="divide-y">
      {conversations.map((c) => {
        const Icon = c.channel === "whatsapp" ? MessageCircle : Globe;
        return (
          <li key={c.id}>
            <Link href={`/erp/sales/chatbot?c=${c.id}`} className="flex items-center gap-3 px-5 py-3 text-sm hover:bg-accent/50">
              <Icon className="size-4 shrink-0 text-muted-foreground" aria-label={lbl(CHANNEL, c.channel)} />
              <span className="min-w-0 flex-1">
                <span className="block truncate font-medium">{c.displayName || c.phone || c.email}</span>
                <span className="block truncate text-xs text-muted-foreground">{c.lastMessagePreview}</span>
              </span>
              {c.botActive ? <Bot className="size-3.5 text-muted-foreground" aria-label={ss.chat.botOn} /> : null}
              <span className="text-xs text-muted-foreground">{lbl(CONVERSATION_STATUS, c.status)}</span>
              <span className="w-20 text-right text-xs text-muted-foreground">{formatRelative(c.lastMessageAt)}</span>
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
