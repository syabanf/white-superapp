import Link from "next/link";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel } from "@/features/publishing/time";
import { num } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { ReportPost } from "@/features/content/types";
import { formatCompact, formatNumber } from "@/lib/format";

export function ReportPostsTable({ posts, slug }: { posts: ReportPost[]; slug: string }) {
  if (posts.length === 0) return <p className="px-(--card-pad) py-4 text-sm text-muted-foreground">{c.none}</p>;
  return (
    <div className="overflow-x-auto scrollbar-thin">
      <table className="w-full min-w-[640px] text-[13px]">
        <thead>
          <tr className="label-mono border-b text-left text-muted-foreground">
            <th className="px-(--card-pad) py-2 font-normal">{c.colContent}</th>
            <th className="px-2 py-2 text-right font-normal">{c.engagement}</th>
            <th className="px-2 py-2 text-right font-normal">{c.reach}</th>
            <th className="px-2 py-2 text-right font-normal">{c.likes}</th>
            <th className="px-2 py-2 text-right font-normal">{c.commentsCount}</th>
            <th className="px-2 py-2 text-right font-normal">{c.shares}</th>
            <th className="px-(--card-pad) py-2 text-right font-normal">{c.saves}</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {posts.map((p) => (
            <tr key={p.itemId}>
              <td className="px-(--card-pad) py-2.5">
                <Link href={`/clients/${slug}/content/board/${p.itemId}`} className="block max-w-[36ch] truncate font-medium hover:underline">
                  {p.title}
                </Link>
                <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <PlatformIcon platform={p.platform} className="size-3" /> {platformLabel(p.platform)} · {c.format[p.format]} · {dayLabel(p.plannedDate)}
                  {p.permalink ? (
                    <a href={p.permalink} target="_blank" rel="noreferrer" className="text-brand hover:underline">
                      {c.permalink}
                    </a>
                  ) : null}
                </span>
              </td>
              {p.hasMetrics ? (
                <>
                  <td className="tabular px-2 py-2.5 text-right font-medium">{formatNumber(num(p.engagements))}</td>
                  <td className="tabular px-2 py-2.5 text-right">{formatCompact(num(p.reach))}</td>
                  <td className="tabular px-2 py-2.5 text-right">{formatNumber(num(p.likes))}</td>
                  <td className="tabular px-2 py-2.5 text-right">{formatNumber(num(p.comments))}</td>
                  <td className="tabular px-2 py-2.5 text-right">{formatNumber(num(p.shares))}</td>
                  <td className="tabular px-(--card-pad) py-2.5 text-right">{formatNumber(num(p.saves))}</td>
                </>
              ) : (
                <td colSpan={6} className="px-(--card-pad) py-2.5 text-right text-xs text-muted-foreground">
                  {c.noMetrics}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
