import { ExternalLink } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { PostStatusBadge, StatusDot } from "@/features/publishing/components/post-status";
import type { PostActivityView, PostTargetView } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { formatDateTime, initials } from "@/lib/format";

export function TargetsList({ targets, timezone }: { targets: PostTargetView[]; timezone: string }) {
  if (targets.length === 0) return <p className="text-sm text-muted-foreground">{p.err.nothingToPublish}</p>;
  return (
    <ul className="divide-y rounded-xl border bg-card">
      {targets.map((tg) => (
        <li key={tg.id} className="flex items-start gap-3 px-3 py-2.5">
          <div className="relative shrink-0">
            <Avatar className="size-8">
              <AvatarImage src={tg.avatarUrl ?? undefined} alt="" />
              <AvatarFallback className="text-[10px]">{initials(tg.displayName || tg.username)}</AvatarFallback>
            </Avatar>
            <PlatformIcon platform={tg.platform} className="absolute -right-1 -bottom-1 size-3.5 rounded-full bg-card p-px" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span className="truncate text-[13px] font-medium">@{tg.username}</span>
              <PostStatusBadge status={tg.status} />
              {tg.bodyOverride ? <span className="label-mono text-brand">{p.overrideBadge}</span> : null}
            </div>
            {tg.status === "PUBLISHED" && tg.permalink ? (
              <a href={tg.permalink} target="_blank" rel="noreferrer" className="mt-1 inline-flex items-center gap-1 text-xs text-brand underline-offset-4 hover:underline">
                <ExternalLink className="size-3" /> {p.openPermalink}
                {tg.publishedAt ? <span className="text-muted-foreground"> · {formatDateTime(tg.publishedAt, timezone)}</span> : null}
              </a>
            ) : null}
            {tg.error ? <p className="mt-1 text-xs break-words text-destructive">{tg.error}</p> : null}
            {tg.attempts > 0 && tg.status !== "PUBLISHED" ? <p className="label-mono mt-1 text-muted-foreground">{p.attempts(tg.attempts)}</p> : null}
          </div>
        </li>
      ))}
    </ul>
  );
}

function activityDetail(a: PostActivityView): string | null {
  const m = a.meta ?? {};
  if (typeof m.note === "string" && m.note) return `“${m.note}”`;
  if (typeof m.scheduledAt === "string") return null;
  if (Array.isArray(m.errors) && m.errors.length) {
    const first = m.errors[0] as { target?: string; error?: string };
    return `${first.target ?? ""}: ${first.error ?? ""}`.trim();
  }
  if (typeof m.published === "number" && typeof m.failed === "number" && (m.published > 0 || m.failed > 0)) {
    return `${m.published} terbit · ${m.failed} gagal${m.mock ? ` · ${p.simulated}` : ""}`;
  }
  if (typeof m.from === "string" && typeof m.to === "string" && a.type === "RESCHEDULED") return null;
  return null;
}

export function ActivityTimeline({ activities, timezone }: { activities: PostActivityView[]; timezone: string }) {
  if (activities.length === 0) return <p className="text-sm text-muted-foreground">–</p>;
  return (
    <ol className="relative space-y-4 border-l pl-4">
      {activities.map((a) => {
        const detail = activityDetail(a);
        const status = a.type === "PUBLISHED" ? "PUBLISHED" : a.type === "PUBLISH_FAILED" ? "FAILED" : a.type === "REJECTED" ? "REJECTED" : a.type === "APPROVED" ? "APPROVED" : a.type === "SCHEDULED" ? "SCHEDULED" : a.type === "SUBMITTED" ? "IN_REVIEW" : "DRAFT";
        return (
          <li key={a.id} className="relative">
            <StatusDot status={status} className="absolute top-1.5 -left-[21px] ring-4 ring-background" />
            <p className="text-[13px] font-medium">{p.activity[a.type] ?? a.type}</p>
            <p className="text-xs text-muted-foreground">
              {a.user ?? p.system} · {formatDateTime(a.createdAt, timezone)}
            </p>
            {detail ? <p className="mt-0.5 text-xs break-words text-muted-foreground">{detail}</p> : null}
          </li>
        );
      })}
    </ol>
  );
}
