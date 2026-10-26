"use client";

import Link from "next/link";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { StatusDot, statusColor } from "@/features/publishing/components/post-status";
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
  const color = statusColor(post.status);
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
      // Tinted by status with a solid edge, so a busy month reads at a glance;
      // the dot and the tooltip keep it from relying on colour alone.
      style={{
        background: `color-mix(in oklab, ${color} 11%, transparent)`,
        boxShadow: `inset 2px 0 0 ${color}`,
      }}
      className={cn(
        "lift group/chip rounded-[5px] text-left transition-[filter] hover:brightness-95 dark:hover:brightness-125",
        compact
          ? "flex h-6 items-center gap-1.5 px-2 text-xs"
          : "flex flex-col gap-0.5 px-2.5 py-1.5 text-[13px]",
        draggable && "cursor-grab active:cursor-grabbing",
        className,
      )}
    >
      {/* Month cells are narrow: one line, the title wins and platforms stay in the tooltip. */}
      <span className="flex items-center gap-1.5">
        <StatusDot status={post.status} />
        <span className="tabular shrink-0 text-xs text-muted-foreground">{clock(post.at, timeZone)}</span>
        {compact ? null : (
          <span className="flex shrink-0 items-center gap-0.5">
            {post.platforms.map((pl) => (
              <PlatformIcon key={pl} platform={pl} className="size-3.5" />
            ))}
          </span>
        )}
        {compact ? <span className="min-w-0 flex-1 truncate font-medium">{label}</span> : null}
      </span>
      {compact ? null : <span className="line-clamp-2 font-medium leading-snug">{label}</span>}
    </Link>
  );
}
