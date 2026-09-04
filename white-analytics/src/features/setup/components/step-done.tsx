import Link from "next/link";
import { ArrowRight, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { finishSetupAction } from "@/features/setup/actions";
import { reviewProgress } from "@/features/setup/integrations";
import type { SetupOverview } from "@/features/setup/queries";
import { ts } from "@/features/setup/strings";

export function StepDone({ overview }: { overview: SetupOverview }) {
  const { statuses } = overview;
  const review = reviewProgress(overview.metaReview);
  const rows: { label: string; real: boolean; note: string }[] = [
    { label: ts.labelMeta, real: statuses.META.configured, note: statuses.META.configured ? ts.publishMode(review.publishReady) : ts.modeDemo },
    { label: ts.labelGoogle, real: statuses.GOOGLE.configured, note: statuses.GOOGLE.configured ? ts.modeReal : ts.modeDemo },
    { label: ts.labelDfs, real: statuses.DATAFORSEO.configured, note: statuses.DATAFORSEO.configured ? ts.modeReal : ts.modeDemo },
    { label: ts.labelApify, real: statuses.APIFY.configured, note: statuses.APIFY.configured ? ts.apifyMode : ts.modeDemo },
    { label: ts.labelAi, real: statuses.OPENROUTER.configured, note: statuses.OPENROUTER.configured ? ts.modeReal : ts.modeDemo },
    { label: ts.labelTiktok, real: false, note: ts.tiktokMode },
  ];
  return (
    <div className="space-y-6">
      <div className="flex items-start gap-3">
        <CheckCircle2 className="mt-0.5 size-5 text-brand" />
        <div>
          <h3 className="text-base font-semibold">{ts.doneTitle}</h3>
          <p className="mt-1 text-sm text-muted-foreground">{ts.doneDesc}</p>
        </div>
      </div>

      <ul className="divide-y rounded-lg border">
        {rows.map((r) => (
          <li key={r.label} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5 text-sm">
            <span className="font-medium">{r.label}</span>
            <span className="flex items-center gap-2 text-xs text-muted-foreground">
              {r.note}
              <StatusBadge kind={r.real ? "good" : "neutral"}>{r.real ? ts.modeReal : ts.modeDemo}</StatusBadge>
            </span>
          </li>
        ))}
      </ul>

      <p className="text-sm text-muted-foreground">{ts.doneNext}</p>

      <div className="flex flex-col gap-2 border-t pt-5 sm:flex-row sm:justify-end">
        <Button asChild variant="ghost">
          <Link href="/">{ts.toPortfolio}</Link>
        </Button>
        <form action={finishSetupAction}>
          <Button type="submit" className="group/nudge w-full sm:w-auto">
            {ts.finish} <ArrowRight className="nudge nudge-x size-4" />
          </Button>
        </form>
      </div>
    </div>
  );
}
