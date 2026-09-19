"use client";

import { Bookmark, Heart, MessageCircle, MoreHorizontal, Music2, Send, Share2, ThumbsUp } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import type { Platform } from "@/features/publishing/lib";
import type { MediaRow } from "@/features/publishing/queries";
import { MediaThumb } from "@/features/publishing/components/composer/media-picker";
import { p } from "@/features/publishing/strings";
import { initials } from "@/lib/format";
import { cn } from "@/lib/utils";

export type PreviewAccount = { username: string; displayName: string; avatarUrl: string | null };
export type PreviewMedia = Pick<MediaRow, "id" | "url" | "thumbnailUrl" | "kind" | "altText" | "filename">;

function Caption({ text, max = 140, bold }: { text: string; max?: number; bold?: string }) {
  const short = text.length > max;
  return (
    <p className="text-[13px] leading-snug whitespace-pre-wrap break-words">
      {bold ? <span className="font-semibold">{bold} </span> : null}
      {short ? text.slice(0, max).trimEnd() : text}
      {short ? <span className="text-muted-foreground">{p.previewSeeMore}</span> : null}
    </p>
  );
}

function Dots({ n }: { n: number }) {
  if (n < 2) return null;
  return (
    <div className="flex justify-center gap-1 py-1.5">
      {Array.from({ length: Math.min(n, 10) }, (_, i) => (
        <span key={i} className={cn("size-1.5 rounded-full", i === 0 ? "bg-brand" : "bg-muted-foreground/30")} />
      ))}
    </div>
  );
}

function InstagramCard({ account, caption, media }: { account: PreviewAccount; caption: string; media: PreviewMedia[] }) {
  const first = media[0];
  return (
    <div className="overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow) text-card-foreground">
      <div className="flex items-center gap-2.5 px-3 py-2.5">
        <Avatar className="size-8 ring-2 ring-[var(--brand-instagram)] ring-offset-1 ring-offset-card">
          <AvatarImage src={account.avatarUrl ?? undefined} alt="" />
          <AvatarFallback className="text-[10px]">{initials(account.displayName)}</AvatarFallback>
        </Avatar>
        <span className="flex-1 text-[13px] font-semibold">{account.username}</span>
        <MoreHorizontal className="size-4 text-muted-foreground" />
      </div>
      {first ? <MediaThumb asset={first} className="aspect-square w-full" /> : <div className="flex aspect-square items-center justify-center bg-muted text-xs text-muted-foreground">{p.sectionMedia}</div>}
      <Dots n={media.length} />
      <div className="flex items-center gap-3 px-3 pt-1">
        <Heart className="size-5" />
        <MessageCircle className="size-5" />
        <Send className="size-5" />
        <Bookmark className="ml-auto size-5" />
      </div>
      <div className="space-y-1 px-3 pt-2 pb-3">
        <p className="text-[13px] font-semibold">{p.previewLikes}</p>
        <Caption text={caption} bold={account.username} />
        <p className="label-mono text-muted-foreground">{p.previewSponsored}</p>
      </div>
    </div>
  );
}

function FacebookCard({ account, caption, media, linkUrl }: { account: PreviewAccount; caption: string; media: PreviewMedia[]; linkUrl?: string }) {
  const first = media[0];
  let host = "";
  try {
    host = linkUrl ? new URL(linkUrl).hostname : "";
  } catch {
    host = "";
  }
  return (
    <div className="overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow) text-card-foreground">
      <div className="flex items-center gap-2.5 px-3 py-2.5">
        <Avatar className="size-9">
          <AvatarImage src={account.avatarUrl ?? undefined} alt="" />
          <AvatarFallback className="text-[10px]">{initials(account.displayName)}</AvatarFallback>
        </Avatar>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] font-semibold">{account.displayName}</span>
          <span className="block text-[11px] text-muted-foreground">{p.previewSponsored} · 🌐</span>
        </span>
        <MoreHorizontal className="size-4 text-muted-foreground" />
      </div>
      <div className="px-3 pb-2">
        <Caption text={caption} max={220} />
      </div>
      {first ? (
        <MediaThumb asset={first} className="aspect-[4/3] w-full" />
      ) : host ? (
        <div className="border-y bg-muted/40 px-3 py-2.5">
          <p className="label-mono text-muted-foreground">{host}</p>
          <p className="truncate text-[13px] font-semibold">{linkUrl}</p>
        </div>
      ) : null}
      {media.length > 1 ? <p className="px-3 pt-1 text-[11px] text-muted-foreground">+{media.length - 1} foto</p> : null}
      <div className="grid grid-cols-3 border-t text-[12px] text-muted-foreground">
        <span className="flex items-center justify-center gap-1.5 py-2"><ThumbsUp className="size-4" /> Suka</span>
        <span className="flex items-center justify-center gap-1.5 py-2"><MessageCircle className="size-4" /> Komentar</span>
        <span className="flex items-center justify-center gap-1.5 py-2"><Share2 className="size-4" /> Bagikan</span>
      </div>
    </div>
  );
}

function TikTokCard({ account, caption, media }: { account: PreviewAccount; caption: string; media: PreviewMedia[] }) {
  const first = media.find((m) => m.kind === "VIDEO") ?? media[0];
  return (
    <div className="relative mx-auto aspect-[9/16] w-full max-w-[260px] overflow-hidden rounded-2xl border bg-black text-white">
      {first ? <MediaThumb asset={first} className="absolute inset-0 opacity-90" /> : null}
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 via-black/40 to-transparent p-3 pr-14">
        <p className="text-[13px] font-semibold">@{account.username}</p>
        <p className="mt-1 line-clamp-4 text-[12px] leading-snug whitespace-pre-wrap">{caption}</p>
        <p className="mt-1.5 flex items-center gap-1 text-[11px] text-white/80"><Music2 className="size-3" /> Suara asli · {account.displayName}</p>
      </div>
      <div className="absolute right-2 bottom-16 flex flex-col items-center gap-4 text-white">
        <Avatar className="size-9 border-2 border-white"><AvatarImage src={account.avatarUrl ?? undefined} alt="" /><AvatarFallback className="text-[10px] text-black">{initials(account.displayName)}</AvatarFallback></Avatar>
        <Heart className="size-6" />
        <MessageCircle className="size-6" />
        <Bookmark className="size-6" />
        <Share2 className="size-6" />
      </div>
    </div>
  );
}

export function PostPreview({
  platform,
  account,
  caption,
  media,
  linkUrl,
  className,
}: {
  platform: Platform;
  account: PreviewAccount;
  caption: string;
  media: PreviewMedia[];
  linkUrl?: string;
  className?: string;
}) {
  return (
    <div className={cn("space-y-2", className)}>
      <p className="label-mono flex items-center gap-1.5 text-muted-foreground">
        <PlatformIcon platform={platform} className="size-3" /> {platform === "INSTAGRAM" ? "Instagram feed" : platform === "FACEBOOK" ? "Facebook Page" : "TikTok"}
      </p>
      {platform === "INSTAGRAM" ? (
        <InstagramCard account={account} caption={caption} media={media} />
      ) : platform === "FACEBOOK" ? (
        <FacebookCard account={account} caption={caption} media={media} linkUrl={linkUrl} />
      ) : (
        <TikTokCard account={account} caption={caption} media={media} />
      )}
    </div>
  );
}
