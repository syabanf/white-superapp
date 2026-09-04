"use client";

import * as React from "react";
import { Bookmark, Heart, ImageOff, MessageCircle, Play, Share2 } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { formatCompact, formatDate, formatPercent } from "@/lib/format";
import type { SocialPostRow } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { t } from "@/i18n/id";

/** Grid kartu postingan teratas (maks. 6). */
export function TopPostsGrid({ posts }: { posts: SocialPostRow[] }) {
  if (posts.length === 0) {
    return <p className="rounded-xl border border-dashed py-10 text-center text-sm text-muted-foreground">{s.noPosts}</p>;
  }
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {posts.map((p) => (
        <PostCard key={p.id} post={p} />
      ))}
    </div>
  );
}

function Chip({ icon: Icon, value, label }: { icon: LucideIcon; value: number; label: string }) {
  return (
    <span className="inline-flex items-center gap-1" title={label}>
      <Icon className="size-3.5" aria-hidden />
      <span className="tabular">{formatCompact(value)}</span>
      <span className="sr-only">{label}</span>
    </span>
  );
}

function PostCard({ post }: { post: SocialPostRow }) {
  const [imgError, setImgError] = React.useState(false);

  const card = (
    <Card className="h-full gap-0 overflow-hidden py-0 transition-colors hover:border-foreground/25">
      <div className="relative aspect-square w-full bg-muted">
        {post.thumbnailUrl && !imgError ? (
          // eslint-disable-next-line @next/next/no-img-element -- URL eksternal demo (picsum), bukan aset next/image
          <img
            src={post.thumbnailUrl}
            alt=""
            loading="lazy"
            className="size-full object-cover"
            onError={() => setImgError(true)}
          />
        ) : (
          <div className="flex size-full items-center justify-center text-muted-foreground/50">
            <ImageOff className="size-8" aria-hidden />
          </div>
        )}
        <Badge variant="secondary" className="absolute top-2 left-2 shadow-sm">
          {post.type}
        </Badge>
        {post.username ? (
          <Badge variant="secondary" className="absolute top-2 right-2 max-w-[60%] shadow-sm">
            <span className="truncate">@{post.username}</span>
          </Badge>
        ) : null}
      </div>
      <div className="flex flex-1 flex-col gap-2 p-3">
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>{formatDate(post.publishedAt)}</span>
          <span className="font-semibold text-foreground tabular">
            {formatPercent(post.er)} <span className="font-normal text-muted-foreground">{t.social.er}</span>
          </span>
        </div>
        <p className="line-clamp-3 text-sm leading-snug">{post.captionShort}</p>
        <div className="mt-auto flex flex-wrap gap-x-3 gap-y-1 pt-1 text-xs text-muted-foreground">
          <Chip icon={Heart} value={post.likes} label={t.social.likes} />
          <Chip icon={MessageCircle} value={post.comments} label={t.social.comments} />
          <Chip icon={Share2} value={post.shares} label={t.social.shares} />
          <Chip icon={Bookmark} value={post.saves} label={t.social.saves} />
          {post.views > 0 ? <Chip icon={Play} value={post.views} label={t.social.views} /> : null}
        </div>
      </div>
    </Card>
  );

  if (!post.permalink) return card;
  return (
    <a href={post.permalink} target="_blank" rel="noopener noreferrer" className="block h-full">
      {card}
    </a>
  );
}
