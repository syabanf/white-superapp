import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { formatCurrency, formatDate, formatRelative } from "@/lib/format";
import type { DraftListRow } from "@/features/campaigns/queries";
import { c, objectiveLabel } from "@/features/campaigns/strings";
import { DraftStatusBadge, SimulatedBadge } from "./status";

/** Draft cards: one per draft, status badge, budget, schedule. Stacks to one column on phones. */
export function DraftList({ slug, drafts }: { slug: string; drafts: DraftListRow[] }) {
  return (
    <ul className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
      {drafts.map((d) => (
        <li key={d.id} className="min-w-0">
          <Link
            href={`/clients/${slug}/ads/campaigns/${d.id}`}
            className="lift group/nudge flex h-full min-h-44 flex-col rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <div className="flex flex-wrap items-center gap-2">
              <DraftStatusBadge status={d.status} />
              {d.status === "SUBMITTED" && d.simulated ? <SimulatedBadge short /> : null}
            </div>
            <h3 className="mt-3 line-clamp-2 text-base font-semibold tracking-[-0.01em]">{d.name || c.newTitle}</h3>
            <p className="label-mono mt-1 text-muted-foreground">
              {objectiveLabel(d.objective)} · {c.creativesCount(d.creatives)}
            </p>
            <div className="mt-auto flex items-end justify-between gap-3 pt-4">
              <div className="min-w-0">
                <div className="text-lg font-semibold">
                  {formatCurrency(d.dailyBudget, d.currency)}
                  <span className="text-sm font-normal text-muted-foreground">{c.perDay}</span>
                </div>
                <div className="truncate text-xs text-muted-foreground">
                  {formatDate(d.startDate)}
                  {d.endDate ? ` – ${formatDate(d.endDate)}` : ` · ${c.openEnded}`}
                  {" · "}
                  {d.submittedAt ? c.submittedAt(formatRelative(d.submittedAt)) : c.updated(formatRelative(d.updatedAt))}
                </div>
              </div>
              <ArrowRight className="nudge size-4 shrink-0 text-muted-foreground" />
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}
