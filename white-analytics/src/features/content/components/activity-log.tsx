import { formatDateTime } from "@/lib/format";
import { isItemStatus } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { ItemActivity } from "@/features/content/types";

const statusName = (s?: string) => (isItemStatus(s) ? c.itemStatus[s] : s);

/** Audit log of an item, newest first as the API returns it. */
export function ActivityLog({ activities, timezone }: { activities: ItemActivity[]; timezone: string }) {
  if (activities.length === 0) return <p className="text-sm text-muted-foreground">{c.noActivity}</p>;
  return (
    <ol className="relative space-y-4 border-l pl-4">
      {activities.map((a) => (
        <li key={a.id} className="relative">
          <span aria-hidden className="absolute top-1.5 -left-[21px] size-2.5 rounded-full border-2 border-background bg-muted-foreground/60" />
          <p className="text-[13px]">
            <span className="font-medium">{a.userName || c.system}</span> {c.activity[a.type] ?? a.type.toLowerCase()}
            {a.toStatus ? (
              <span className="text-muted-foreground">
                {" "}
                {a.fromStatus ? `${statusName(a.fromStatus)} → ` : ""}
                {statusName(a.toStatus)}
              </span>
            ) : null}
          </p>
          {a.note ? <p className="mt-0.5 text-xs text-muted-foreground">“{a.note}”</p> : null}
          <p className="label-mono mt-0.5 text-muted-foreground">{formatDateTime(a.createdAt, timezone)}</p>
        </li>
      ))}
    </ol>
  );
}
