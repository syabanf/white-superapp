"use client";

import Link from "next/link";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { StatusDot } from "@/features/publishing/components/post-status";
import type { CalendarPost } from "@/features/publishing/queries";
import { clock } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { cn } from "@/lib/utils";

export const DND_MIME = "application/x-white-post";

export function PostChip({
  post,
  href,
  timeZone,
  draggable,
  compact,
  className,
}: {
  post: CalendarPost;
  href: string;
  timeZone: string;
  draggable?: boolean;
  /** month-cell size (one line) vs agenda size (two lines) */
  compact?: boolean;
  className?: string;
}) {
  const label = post.title || post.bodyShort || p.untitled;
  return (
    <Link
      href={href}
      draggable={draggable}
      onDragStart={(e) => {
        if (!draggable) return;
        e.dataTransfer.setData(DND_MIME, post.id);
        e.dataTransfer.setData("text/plain", post.id);
        e.dataTransfer.effectAllowed = "move";
      }}
      title={`${clock(post.at, timeZone)} · ${post.platforms.join(", ")} · ${label}`}
      className={cn(
        "lift group/chip flex items-center gap-1.5 rounded-full bg-muted/70 text-left transition-colors hover:bg-muted",
        compact ? "h-6 px-2 text-xs" : "rounded-2xl px-3 py-2 text-[13px]",
        draggable && "cursor-grab active:cursor-grabbing",
        className,
      )}
    >
      <StatusDot status={post.status} />
      <span className="tabular shrink-0 text-xs text-muted-foreground">{clock(post.at, timeZone)}</span>
      {/* Month cells are narrow: the title wins, platforms stay in the tooltip and the agenda view. */}
      {compact ? null : (
        <span className="flex shrink-0 items-center -space-x-0.5">
          {post.platforms.map((pl) => (
            <PlatformIcon key={pl} platform={pl} className="size-3.5" />
          ))}
        </span>
      )}
      <span className="min-w-0 flex-1 truncate font-medium">{label}</span>
    </Link>
  );
}
